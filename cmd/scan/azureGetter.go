package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// ---- Azure Retail Prices API (no key) ----
// GET https://prices.azure.com/api/retail/prices?$filter=...
// The API prices VM sizes but does not describe their hardware, so the GPU VM
// sizes below carry their specs from Azure's VM size documentation.
type azureSize struct {
	GPU    string // gpuSpecs input
	NumGPU int
	VramGB int // per GPU
	VCPUs  float64
	RamGB  int
}

var azureGPUSizes = map[string]azureSize{
	"Standard_NC4as_T4_v3":       {"T4", 1, 16, 4, 28},
	"Standard_NC64as_T4_v3":      {"T4", 4, 16, 64, 440},
	"Standard_NV36ads_A10_v5":    {"A10", 1, 24, 36, 440},
	"Standard_NC24ads_A100_v4":   {"A100 PCIe 80GB", 1, 80, 24, 220},
	"Standard_NC48ads_A100_v4":   {"A100 PCIe 80GB", 2, 80, 48, 440},
	"Standard_NC96ads_A100_v4":   {"A100 PCIe 80GB", 4, 80, 96, 880},
	"Standard_ND96asr_v4":        {"A100 SXM 40GB", 8, 40, 96, 900},
	"Standard_ND96amsr_A100_v4":  {"A100 SXM 80GB", 8, 80, 96, 1900},
	"Standard_NC40ads_H100_v5":   {"H100 NVL", 1, 94, 40, 320},
	"Standard_NC80adis_H100_v5":  {"H100 NVL", 2, 94, 80, 640},
	"Standard_ND96isr_H100_v5":   {"H100 SXM", 8, 80, 96, 1900},
	"Standard_ND96isr_H200_v5":   {"H200", 8, 141, 96, 1850},
	"Standard_ND96isr_MI300X_v5": {"MI300X", 8, 192, 96, 1850},
}

type azurePriceResp struct {
	Items        []azurePriceItem `json:"Items"`
	NextPageLink string           `json:"NextPageLink"`
}

type azurePriceItem struct {
	UnitPrice     float64 `json:"unitPrice"`
	ArmRegionName string  `json:"armRegionName"`
	Location      string  `json:"location"`
	SkuName       string  `json:"skuName"`
	ProductName   string  `json:"productName"`
	ArmSkuName    string  `json:"armSkuName"`
}

func azurePricesURL() string {
	skus := make([]string, 0, len(azureGPUSizes))
	for s := range azureGPUSizes {
		skus = append(skus, fmt.Sprintf("armSkuName eq '%s'", s))
	}
	sort.Strings(skus)
	filter := "serviceName eq 'Virtual Machines' and priceType eq 'Consumption' and (" + strings.Join(skus, " or ") + ")"
	return "https://prices.azure.com/api/retail/prices?$filter=" + url.PathEscape(filter)
}

// cheapestAzurePrices keeps the cheapest Linux pay-as-you-go price of each VM
// size. Spot, low-priority and Windows meters are dropped, and so are the
// AT&T edge zones (armRegionName "att..."), which ordinary accounts cannot use.
func cheapestAzurePrices(items []azurePriceItem) map[string]azurePriceItem {
	best := map[string]azurePriceItem{}
	for _, it := range items {
		if it.UnitPrice <= 0 ||
			strings.Contains(it.SkuName, "Spot") ||
			strings.Contains(it.SkuName, "Low Priority") ||
			strings.Contains(it.ProductName, "Windows") ||
			strings.HasPrefix(it.ArmRegionName, "att") {
			continue
		}
		if cur, ok := best[it.ArmSkuName]; !ok || it.UnitPrice < cur.UnitPrice {
			best[it.ArmSkuName] = it
		}
	}
	return best
}

func azureRows(best map[string]azurePriceItem) []GPU {
	out := make([]GPU, 0, len(best))
	for sku, it := range best {
		size, ok := azureGPUSizes[sku]
		if !ok {
			continue
		}
		perGPUFlops, memBWGBs, name := gpuSpecs(size.GPU)
		totalFlops := float64(size.NumGPU) * perGPUFlops / 1e12

		newGpu := GPU{
			_Id:      sku,
			Location: it.Location,
			// Azure's SLA for a single VM on Premium SSD.
			Reliability: 0.999,

			Name:              name,
			Vram:              size.VramGB * 1024,
			TotalFlops:        totalFlops,
			GpuMemoryBandwith: memBWGBs,
			NumGPUs:           size.NumGPU,

			CpuCores: size.VCPUs,
			Ram:      size.RamGB * 1024,

			TotalCostPH:      it.UnitPrice,
			GpuCostPH:        it.UnitPrice / float64(size.NumGPU),
			FlopsPerDollarPH: totalFlops / it.UnitPrice,
			Source:           "azure",
			Url:              "https://azure.microsoft.com/en-us/pricing/details/virtual-machines/linux/",
		}
		newGpu.Score = calculateScore(newGpu)
		newGpu.ScoreDPH = newGpu.Score / newGpu.TotalCostPH
		out = append(out, newGpu)
	}
	return out
}

func parseAzurePage(r io.Reader) (azurePriceResp, error) {
	var pr azurePriceResp
	if err := json.NewDecoder(r).Decode(&pr); err != nil {
		return pr, fmt.Errorf("decode: %w", err)
	}
	return pr, nil
}

func azureGetter() ([]GPU, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	var items []azurePriceItem
	next := azurePricesURL()
	for page := 0; next != "" && page < 20; page++ {
		resp, err := client.Get(next)
		if err != nil {
			return nil, fmt.Errorf("fetch azure prices: %w", err)
		}
		if resp.StatusCode/100 != 2 {
			resp.Body.Close()
			return nil, fmt.Errorf("azure API status %s", resp.Status)
		}
		pr, err := parseAzurePage(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		items = append(items, pr.Items...)
		next = pr.NextPageLink
	}

	out := azureRows(cheapestAzurePrices(items))
	fmt.Printf("Found %d Azure GPUs\n", len(out))
	return out, nil
}
