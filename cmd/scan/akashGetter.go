package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ---- Akash console GPU price summary (no key) ----
// GET https://console-api.akash.network/v1/gpu-prices
// One entry per GPU model across all providers; price is USD per GPU per hour.
type akashResp struct {
	Models []akashModel `json:"models"`
}

type akashModel struct {
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	Ram          string `json:"ram"`       // e.g. "80Gi"
	Interface    string `json:"interface"` // e.g. "SXM5", "PCIe"
	Availability struct {
		Total     int `json:"total"`
		Available int `json:"available"`
	} `json:"availability"`
	Price *struct {
		Min float64 `json:"min"`
	} `json:"price"`
}

func akashVramGB(s string) int {
	v, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(s), "Gi"))
	return v
}

func parseAkash(r io.Reader) ([]GPU, error) {
	var ar akashResp
	if err := json.NewDecoder(r).Decode(&ar); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0, len(ar.Models))
	for _, m := range ar.Models {
		if m.Availability.Available <= 0 || m.Price == nil || m.Price.Min <= 0 {
			continue
		}
		vramGB := akashVramGB(m.Ram)
		flops, memBWGBs, name := gpuSpecs(proBlackwellName(fmt.Sprintf("%s %s %dgb %s", m.Vendor, m.Model, vramGB, m.Interface)))
		if name == "unknown" {
			continue
		}
		totalFlops := flops / 1e12

		newGpu := GPU{
			_Id: fmt.Sprintf("%s-%s-%s", m.Model, m.Ram, m.Interface),

			Name:              name,
			Vram:              vramGB * 1024,
			TotalFlops:        totalFlops,
			GpuMemoryBandwith: memBWGBs,
			NumGPUs:           1,

			TotalCostPH:      m.Price.Min,
			GpuCostPH:        m.Price.Min,
			FlopsPerDollarPH: totalFlops / m.Price.Min,
			Source:           "akash",
			Url:              "https://console.akash.network/",
		}
		newGpu.Score = calculateScore(newGpu)
		newGpu.ScoreDPH = newGpu.Score / newGpu.TotalCostPH
		out = append(out, newGpu)
	}
	return out, nil
}

func akashGetter() ([]GPU, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get("https://console-api.akash.network/v1/gpu-prices")
	if err != nil {
		return nil, fmt.Errorf("fetch akash prices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("akash API status %s", resp.Status)
	}

	out, err := parseAkash(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d Akash GPUs\n", len(out))
	return out, nil
}
