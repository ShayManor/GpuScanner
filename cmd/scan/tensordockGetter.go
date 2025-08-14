package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func getTensorDockURL(o GPU) string {
	// Format: https://marketplace.tensordock.com/deploy?gpu=GPU_NAME&location=LOCATION
	gpuParam := strings.ReplaceAll(strings.ToLower(o.Name), " ", "_")

	return fmt.Sprintf(
		"https://marketplace.tensordock.com/deploy?gpu=%s&ram=%d&vcpus=%.0f&storage=%d",
		gpuParam,
		o.Ram/1000, // Convert MB to GB
		o.CpuCores,
		int(o.DiskSpace),
	)
}

// ---- TensorDock API shapes (see docs) ----
// GET https://dashboard.tensordock.com/api/v2/hostnodes (Bearer token)
// Ref: docs pages "Fetch Hostnodes and Locations" + "Getting Started".
type tdHostnodesResp struct {
	Data struct {
		Hostnodes []tdHostnode `json:"hostnodes"`
	} `json:"data"`
}

type tdHostnode struct {
	ID                 string  `json:"id"`
	UptimePercentage   float64 `json:"uptime_percentage"`
	AvailableResources struct {
		GPUs []tdGPU `json:"gpus"`

		VCPUCount int     `json:"vcpu_count"`
		RAMGB     int     `json:"ram_gb"`
		StorageGB float64 `json:"storage_gb"`
		HasPubIP  bool    `json:"has_public_ip_available"`
	} `json:"available_resources"`

	Pricing struct {
		PerVcpuHr      float64 `json:"per_vcpu_hr"`
		PerGBRamHr     float64 `json:"per_gb_ram_hr"`
		PerGBStorageHr float64 `json:"per_gb_storage_hr"`
	} `json:"pricing"`

	Location struct {
		City                   string  `json:"city"`
		StateProvince          string  `json:"stateprovince"`
		Country                string  `json:"country"`
		NetworkSpeedGbps       float64 `json:"network_speed_gbps"`
		NetworkSpeedUploadGbps float64 `json:"network_speed_upload_gbps"`
		Tier                   int     `json:"tier"`
	} `json:"location"`
}

type tdGPU struct {
	V0Name         string  `json:"v0Name"` // e.g., "h100-sxm5-80gb", "geforcertx4090-pcie-24gb"
	AvailableCount int     `json:"availableCount"`
	PricePerHr     float64 `json:"price_per_hr"`
}

// ---- Helper: parse VRAM from the v0Name (returns MB; 0 if unknown) ----
var vramRe = regexp.MustCompile(`(?i)(\d+)\s*gb`)

func parseVRAMMB(name string) int {
	m := vramRe.FindStringSubmatch(name)
	if len(m) >= 2 {
		gb, err := strconv.Atoi(m[1])
		if err == nil {
			return gb * 1024
		}
	}
	return 0
}

// Env var required: TENSORDOCK_TOKEN
func tensordockGetter() ([]GPU, error) {
	token := os.Getenv("TENSORDOCK_TOKEN")
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("TENSORDOCK_TOKEN is not set")
	}

	req, err := http.NewRequest("GET", "https://dashboard.tensordock.com/api/v2/hostnodes", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch tensordock hostnodes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("tensordock API status %s", resp.Status)
	}

	var hr tdHostnodesResp
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0, 256)
	for _, hn := range hr.Data.Hostnodes {
		loc := strings.TrimSpace(fmt.Sprintf("%s, %s", hn.Location.City, hn.Location.Country))
		// Bandwidth in docs is Gbps; your struct prints Mbps. Convert.
		downMbps := hn.Location.NetworkSpeedGbps * 1000
		upMbps := hn.Location.NetworkSpeedUploadGbps * 1000

		for _, g := range hn.AvailableResources.GPUs {
			if g.AvailableCount <= 0 {
				continue
			}
			totalFlops, memBWGBs := gpuSpecs(g.V0Name)
			totalFlops = totalFlops / 1e12
			newGpu := GPU{
				_Id:         hn.ID,
				Location:    loc,
				Reliability: hn.UptimePercentage / 100.0, // docs give percent
				Duration:    0,                           // not exposed

				Name:              g.V0Name,
				Vram:              parseVRAMMB(g.V0Name),
				TotalFlops:        totalFlops, // not exposed
				GpuMemoryBandwith: memBWGBs,   // not exposed
				NumGPUs:           g.AvailableCount,

				CpuCores: float64(hn.AvailableResources.VCPUCount),
				CpuName:  "",
				CpuGhz:   0,
				CpuArch:  "",

				Ram: hn.AvailableResources.RAMGB * 1024, // GB

				DiskSpace: hn.AvailableResources.StorageGB, // GB
				DiskBW:    0,
				DiskName:  "",

				UploadSpeed:   upMbps,
				DownloadSpeed: downMbps,

				// Prices: TensorDock exposes GPU price/hr, plus unit prices for CPU/RAM/Storage.
				// To keep semantics consistent with Vast, we set totalCostPH to GPU price here.
				TotalCostPH:      g.PricePerHr,
				GpuCostPH:        g.PricePerHr,
				DiskCostPH:       g.PricePerHr, // per-GB rate exists, but we avoid mixing units here
				UploadCostPH:     0,
				DownloadCostPH:   0,
				FlopsPerDollarPH: totalFlops / g.PricePerHr,
				Source:           "tensordock",
			}
			newGpu.Url = getTensorDockURL(newGpu)
			out = append(out, newGpu)

		}
	}

	fmt.Printf("Found %d TensorDock GPUs\n", len(out))
	return out, nil
}
