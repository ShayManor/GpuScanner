package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ---- Akamai Cloud (Linode) public plan list (no key) ----
// GET https://api.linode.com/v4/linode/types
// memory is in MB and disk in MB; price.hourly is the whole plan in USD.
type linodeResp struct {
	Data []linodeType `json:"data"`
}

type linodeType struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"` // e.g. "RTX4000 Ada x2 Medium"
	Class  string  `json:"class"`
	GPUs   int     `json:"gpus"`
	VCPUs  float64 `json:"vcpus"`
	Memory int     `json:"memory"`
	Disk   float64 `json:"disk"`
	Price  struct {
		Hourly float64 `json:"hourly"`
	} `json:"price"`
}

func parseLinode(r io.Reader) ([]GPU, error) {
	var lr linodeResp
	if err := json.NewDecoder(r).Decode(&lr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0)
	for _, t := range lr.Data {
		if t.Class != "gpu" || t.GPUs <= 0 || t.Price.Hourly <= 0 {
			continue
		}
		perGPUFlops, memBWGBs, name := gpuSpecs(t.Label)
		if name == "unknown" {
			continue
		}
		totalFlops := float64(t.GPUs) * perGPUFlops / 1e12

		newGpu := GPU{
			_Id: t.ID,

			Name:              name,
			TotalFlops:        totalFlops,
			GpuMemoryBandwith: memBWGBs,
			NumGPUs:           t.GPUs,

			CpuCores: t.VCPUs,
			Ram:      t.Memory,

			DiskSpace: t.Disk / 1024,

			TotalCostPH:      t.Price.Hourly,
			GpuCostPH:        t.Price.Hourly / float64(t.GPUs),
			FlopsPerDollarPH: totalFlops / t.Price.Hourly,
			Source:           "linode",
			Url:              "https://www.linode.com/pricing/",
		}
		newGpu.Score = calculateScore(newGpu)
		newGpu.ScoreDPH = newGpu.Score / newGpu.TotalCostPH
		out = append(out, newGpu)
	}
	return out, nil
}

func linodeGetter() ([]GPU, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get("https://api.linode.com/v4/linode/types")
	if err != nil {
		return nil, fmt.Errorf("fetch linode types: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("linode API status %s", resp.Status)
	}

	out, err := parseLinode(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d Linode GPUs\n", len(out))
	return out, nil
}
