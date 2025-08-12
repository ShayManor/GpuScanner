package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
	ID               int     `json:"id"`
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
	MemoryBandwith   float64 `json:"gpu_mem_bw"`
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

	out := make([]GPU, 0, len(sr.Offers))
	for _, o := range sr.Offers {
		if o.Rentable {
			out = append(out, GPU{
				Id:          strconv.Itoa(o.ID),
				Location:    o.Location,
				Reliability: o.Reliability,
				Duration:    o.Duration,
				Source:      "vast",

				Name:              o.GPUName,
				Vram:              o.Vram,
				TotalFlops:        o.Flops,
				GpuMemoryBandwith: o.MemoryBandwith,
				NumGPUs:           o.NumGPUs,

				CpuCores: o.CPUCores,
				CpuName:  o.CPUName,
				CpuGhz:   o.CPUGhz,
				CpuArch:  o.CPUArch,

				Ram: o.Ram,

				DiskSpace: o.DiskSpace,
				DiskBW:    o.DiskBandwith,
				DiskName:  o.DiskName,

				UploadSpeed:   o.Upload,
				DownloadSpeed: o.Download,

				TotalCostPH:      o.DPHTotal,
				GpuCostPH:        o.Search.GpuCostPerHour,
				DiskCostPH:       o.Search.DiskHour,
				UploadCostPH:     o.UploadCost,
				DownloadCostPH:   o.DownloadCost,
				FlopsPerDollarPH: o.FlopsPerDollarPH,
			})
		}
	}
	fmt.Printf("Found %d vast gpus\n", len(out))
	return out, nil
}
