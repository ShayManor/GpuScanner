package main

import (
	"fmt"
	"time"
)

type GPU struct {
	// Instance details
	Id          string  `json:"id" db:"id"`
	Location    string  `json:"location" db:"location"`
	Reliability float64 `json:"reliability" db:"reliability"`
	Duration    float64 `json:"duration" db:"duration"`
	Source      string  `json:"source" db:"source"` // e.g., "tensordock", "vast", etc.
	// GPU details
	Name              string  `json:"name" db:"name"`
	Vram              int     `json:"vram" db:"vram_mb"`
	TotalFlops        float64 `json:"totalFlops" db:"total_flops"`
	GpuMemoryBandwith float64 `json:"gpuMemoryBandwidth" db:"gpu_mem_bandwidth_gbps"`
	NumGPUs           int     `json:"numGPUs" db:"num_gpus"`
	// CPU specs
	CpuCores float64 `json:"cpuCores" db:"cpu_cores"`
	CpuName  string  `json:"cpuName" db:"cpu_name"`
	CpuGhz   float64 `json:"cpuGhz" db:"cpu_ghz"`
	CpuArch  string  `json:"cpuArch" db:"cpu_arch"`
	// Ram
	Ram int `json:"ram" db:"ram_mb"`
	// SSD
	DiskSpace float64 `json:"diskSpace" db:"disk_space_gb"`
	DiskBW    float64 `json:"diskBW" db:"disk_bw_gbps"`
	DiskName  string  `json:"diskName" db:"disk_name"`
	// Internet
	UploadSpeed   float64 `json:"uploadSpeed" db:"upload_speed_mbps"`
	DownloadSpeed float64 `json:"downloadSpeed" db:"download_speed_mbps"`
	// Cost
	TotalCostPH      float64 `json:"totalCostPH" db:"total_cost_ph"` // PH = per hour
	GpuCostPH        float64 `json:"gpuCostPH" db:"gpu_cost_ph"`
	DiskCostPH       float64 `json:"diskCostPH" db:"disk_cost_ph"`
	UploadCostPH     float64 `json:"uploadCostPH" db:"upload_cost_ph"`
	DownloadCostPH   float64 `json:"downloadCostPH" db:"download_cost_ph"`
	FlopsPerDollarPH float64 `json:"flopsPerDollarPH" db:"flops_per_dollar_ph"`

	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type Getter func() ([]GPU, error)

func (g GPU) toString() string {
	return fmt.Sprintf("GPU Details:\n"+
		"ID: %s\n"+
		"Name: %s\n"+
		"Location: %s\n"+
		"Total GPUs: %d\n"+
		"VRAM: %d MB\n"+
		"Total FLOPS: %.2e\n"+
		"FLOPS/Dollar/Hour: %.2f\n"+
		"Reliability: %.2f%%\n"+
		"Duration: %.2f hours\n"+
		"GPU Memory Bandwidth: %.2f GB/s\n"+
		"CPU: %s (%s)\n"+
		"CPU Cores: %.1f\n"+
		"CPU Clock: %.2f GHz\n"+
		"RAM: %d GB\n"+
		"Storage: %.2f GB (%s)\n"+
		"Disk Bandwidth: %.2f GB/s\n"+
		"Network: Up %.2f Mbps, Down %.2f Mbps\n"+
		"Hourly Costs:\n"+
		"  Total: $%.2f\n"+
		"  GPU: $%.2f\n"+
		"  Disk: $%.4f\n"+
		"  Upload: $%.6f\n"+
		"  Download: $%.6f"+
		"\nSource: %s",
		g.Id,
		g.Name,
		g.Location,
		g.NumGPUs,
		g.Vram,
		g.TotalFlops,
		g.FlopsPerDollarPH,
		g.Reliability*100,
		g.Duration,
		g.GpuMemoryBandwith,
		g.CpuName,
		g.CpuArch,
		g.CpuCores,
		g.CpuGhz,
		g.Ram,
		g.DiskSpace,
		g.DiskName,
		g.DiskBW,
		g.UploadSpeed,
		g.DownloadSpeed,
		g.TotalCostPH,
		g.GpuCostPH,
		g.DiskCostPH,
		g.UploadCostPH,
		g.DownloadCostPH,
		g.Source,
	)
}
