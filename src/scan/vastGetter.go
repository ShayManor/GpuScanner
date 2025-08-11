package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Response struct {
	Offers []offer `json:"offers"`
}

type Search struct {
	GpuCostPerHour float64 `json:"gpuCostPerHour"`
	DiskHour       float64 `json:"diskHour"`
	TotalHour      float64 `json:"totalHour"`
}

type offer struct {
	ID               int64   `json:"id"`
	GPUName          string  `json:"gpu_name"`
	CPUCores         float64 `json:"cpu_cores"`
	NumGPUs          int     `json:"num_gpus"`
	Vram             int     `json:"gpu_ram"`
	Ram              int     `json:"cpu_ram"`
	DPHTotal         float64 `json:"discounted_dph_total"`
	Verified         bool    `json:"verified"`
	Rentable         bool    `json:"rentable"`
	Location         string  `json:"geolocation"`
	Reliability      float64 `json:"reliability"`
	Duration         float64 `json:"duration"`
	Flops            float64 `json:"total_flops"`
	Lanes            int     `json:"gpu_lanes"`
	MemoryBandwith   float64 `json:"gpu_mem_bw"`
	Architecture     string  `json:"gpu_arch"`
	CPUName          string  `json:"cpu_name"`
	CPUGhz           float64 `json:"cpu_ghz"`
	CPUArch          string  `json:"cpu_arch"`
	ComputeCap       int     `json:"compute_cap"`
	DiskSpace        float64 `json:"disk_space"`
	DiskBandwith     float64 `json:"disk_bw"`
	DiskName         string  `json:"disk_name"`
	Upload           float64 `json:"inet_up"`
	Download         float64 `json:"inet_down"`
	GPUCostPH        float64 `json:"gpu_cost_ph"`
	DiskCostPH       float64 `json:"disk_cost_ph"`
	UploadCost       float64 `json:"inet_up_cost"`
	DownloadCost     float64 `json:"inet_down_cost"`
	FlopsPerDollarPH float64 `json:"flops_per_dphtotal"`
	Search           Search  `json:"search"`
}

func vastGetter() ([]GPU, error) {
	response, err := http.Get("https://console.vast.ai/api/v0/search/asks/")
	if err != nil {

		return nil, fmt.Errorf("Error fetching vast: %s\n", err)
	}
	defer response.Body.Close()

	var sr Response
	if err := json.NewDecoder(response.Body).Decode(&sr); err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil, fmt.Errorf("decode: %w", err)
	}

	// Project to your lean slice.
	out := make([]GPU, 0, len(sr.Offers))
	for _, o := range sr.Offers {
		if o.Rentable {
			out = append(out, GPU{
				id:          o.ID,
				location:    o.Location,
				reliability: o.Reliability,
				duration:    o.Duration,

				name:              o.GPUName,
				vram:              o.Vram,
				totalFlops:        o.Flops,
				gpuLanes:          o.Lanes,
				gpuMemoryBandwith: o.MemoryBandwith,
				architecture:      o.Architecture,
				numGPUs:           o.NumGPUs,

				cpuCores:   o.CPUCores,
				cpuName:    o.CPUName,
				cpuGhz:     o.CPUGhz,
				cpuArch:    o.CPUArch,
				computeCap: o.ComputeCap,

				ram: o.Ram,

				diskSpace: o.DiskSpace,
				diskBW:    o.DiskBandwith,
				diskName:  o.DiskName,

				uploadSpeed:   o.Upload,
				downloadSpeed: o.Download,

				totalCostPH:      o.DPHTotal,
				gpuCostPH:        o.Search.GpuCostPerHour,
				diskCostPH:       o.Search.DiskHour,
				uploadCostPH:     o.UploadCost,
				downloadCostPH:   o.DownloadCost,
				flopsPerDollarPH: o.FlopsPerDollarPH,
			})
		}
	}
	fmt.Printf("Found %d vast gpus\n", len(out))
	for _, o := range out {
		fmt.Println(o.toString())
	}
	return out, nil
}
