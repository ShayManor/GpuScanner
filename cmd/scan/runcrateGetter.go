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

// ---- Runcrate public pricing feed (no key) ----
// GET https://www.runcrate.ai/api/gpu-pricing
// One entry per GPU type; prices are per GPU per hour and specs describe a
// representative configuration of specs.num_gpus GPUs.
const runcratePricingURL = "https://www.runcrate.ai/pricing?utm_source=gpufindr&utm_medium=directory&utm_campaign=gpu_pricing"

type runcrateResp struct {
	GPUs []runcrateGPU `json:"gpus"`
}

type runcrateGPU struct {
	GPUType          string   `json:"gpu_type"`
	Name             string   `json:"name"`
	VRAM             string   `json:"vram"` // e.g. "80GB"
	MinPriceHr       float64  `json:"min_price_hr"`
	AvailableOptions int      `json:"available_options"`
	Regions          []string `json:"regions"`
	Specs            struct {
		VCPUs     float64 `json:"vcpus"`
		MemoryGB  int     `json:"memory_gb"`
		StorageGB float64 `json:"storage_gb"`
		NumGPUs   int     `json:"num_gpus"`
	} `json:"specs"`
}

// "80GB" -> 80; anything unparsable -> 0.
func runcrateVramGB(s string) int {
	v, _ := strconv.Atoi(strings.TrimSuffix(strings.ToUpper(strings.TrimSpace(s)), "GB"))
	return v
}

func parseRuncrate(r io.Reader) ([]GPU, error) {
	var rr runcrateResp
	if err := json.NewDecoder(r).Decode(&rr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0, len(rr.GPUs))
	for _, g := range rr.GPUs {
		if g.AvailableOptions <= 0 || g.MinPriceHr <= 0 {
			continue
		}
		perGPUFlops, memBWGBs, name := gpuSpecs(proBlackwellName(g.Name))
		if name == "unknown" {
			continue
		}
		num := maxInt(g.Specs.NumGPUs, 1)
		totalFlops := float64(num) * perGPUFlops / 1e12
		totalPrice := g.MinPriceHr * float64(num)

		location := ""
		if len(g.Regions) > 0 {
			location = g.Regions[0]
		}

		newGpu := GPU{
			_Id:      g.GPUType,
			Location: location,

			Name:              name,
			Vram:              runcrateVramGB(g.VRAM) * 1024,
			TotalFlops:        totalFlops,
			GpuMemoryBandwith: memBWGBs,
			NumGPUs:           num,

			CpuCores: g.Specs.VCPUs,
			Ram:      g.Specs.MemoryGB * 1024,

			DiskSpace: g.Specs.StorageGB,

			TotalCostPH:      totalPrice,
			GpuCostPH:        g.MinPriceHr,
			FlopsPerDollarPH: totalFlops / totalPrice,
			Source:           "runcrate",
			Url:              runcratePricingURL,
		}
		newGpu.Score = calculateScore(newGpu)
		newGpu.ScoreDPH = newGpu.Score / newGpu.TotalCostPH
		out = append(out, newGpu)
	}
	return out, nil
}

func runcrateGetter() ([]GPU, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get("https://www.runcrate.ai/api/gpu-pricing")
	if err != nil {
		return nil, fmt.Errorf("fetch runcrate pricing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("runcrate API status %s", resp.Status)
	}

	out, err := parseRuncrate(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d Runcrate GPUs\n", len(out))
	return out, nil
}
