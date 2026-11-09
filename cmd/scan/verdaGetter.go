package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ---- Verda (formerly DataCrunch) public instance catalog (no key) ----
// GET https://api.datacrunch.io/v1/instance-types
// One entry per instance type; price_per_hour is the on-demand price of the
// whole instance as a string.
type verdaInstance struct {
	InstanceType string `json:"instance_type"`
	Name         string `json:"name"` // e.g. "H100 SXM5 80GB"
	PricePerHour string `json:"price_per_hour"`
	CPU          struct {
		NumberOfCores float64 `json:"number_of_cores"`
	} `json:"cpu"`
	GPU struct {
		NumberOfGPUs int `json:"number_of_gpus"`
	} `json:"gpu"`
	GPUMemory struct {
		SizeInGigabytes int `json:"size_in_gigabytes"` // whole instance
	} `json:"gpu_memory"`
	Memory struct {
		SizeInGigabytes int `json:"size_in_gigabytes"`
	} `json:"memory"`
}

func parseVerda(r io.Reader) ([]GPU, error) {
	var instances []verdaInstance
	if err := json.NewDecoder(r).Decode(&instances); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0, len(instances))
	for _, in := range instances {
		num := in.GPU.NumberOfGPUs
		price := atof(in.PricePerHour)
		if num <= 0 || price <= 0 {
			continue
		}
		perGPUFlops, memBWGBs, name := gpuSpecs(proBlackwellName(in.Name))
		if name == "unknown" {
			continue
		}
		totalFlops := float64(num) * perGPUFlops / 1e12

		newGpu := GPU{
			_Id: in.InstanceType,

			Name:              name,
			Vram:              in.GPUMemory.SizeInGigabytes / num * 1024,
			TotalFlops:        totalFlops,
			GpuMemoryBandwith: memBWGBs,
			NumGPUs:           num,

			CpuCores: in.CPU.NumberOfCores,
			Ram:      in.Memory.SizeInGigabytes * 1024,

			TotalCostPH:      price,
			GpuCostPH:        price / float64(num),
			FlopsPerDollarPH: totalFlops / price,
			Source:           "verda",
			Url:              "https://verda.com/",
		}
		newGpu.Score = calculateScore(newGpu)
		newGpu.ScoreDPH = newGpu.Score / newGpu.TotalCostPH
		out = append(out, newGpu)
	}
	return out, nil
}

func verdaGetter() ([]GPU, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get("https://api.datacrunch.io/v1/instance-types")
	if err != nil {
		return nil, fmt.Errorf("fetch verda instance types: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("verda API status %s", resp.Status)
	}

	out, err := parseVerda(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d Verda GPUs\n", len(out))
	return out, nil
}
