package main

import (
	"fmt"
	"time"
)

type GPU struct {
	// Instance details
	Id          string  `json:"id" bson:"_id"`
	Location    string  `json:"location" bson:"location"`
	Reliability float64 `json:"reliability" bson:"reliability"`
	Duration    float64 `json:"duration_hours" bson:"duration_hours"`
	Source      string  `json:"source" bson:"source"` // e.g., "tensordock", "vast", etc.
	// GPU details
	Name              string  `json:"name" bson:"name"`
	Vram              int     `json:"vram_mb" bson:"vram_mb"`
	TotalFlops        float64 `json:"total_flops" bson:"total_flops"`
	GpuMemoryBandwith float64 `json:"gpu_mem_bw_gbps" bson:"gpu_mem_bw_gbps"`
	NumGPUs           int     `json:"num_gpus" bson:"num_gpus"`
	// CPU specs
	CpuCores float64 `json:"cpu_cores" bson:"cpu_cores"`
	CpuName  string  `json:"cpu_name" bson:"cpu_name"`
	CpuGhz   float64 `json:"cpu_ghz" bson:"cpu_ghz"`
	CpuArch  string  `json:"cpu_arch" bson:"cpu_arch"`
	// Ram
	Ram int `json:"ram_mb" bson:"ram_mb"`
	// SSD
	DiskSpace float64 `json:"disk_space_gb" bson:"disk_space_gb"`
	DiskBW    float64 `json:"disk_bw_gbps" bson:"disk_bw_gbps"`
	DiskName  string  `json:"disk_name" bson:"disk_name"`
	// Internet
	UploadSpeed   float64 `json:"upload_mbps" bson:"upload_mbps"`
	DownloadSpeed float64 `json:"download_mbps" bson:"download_mbps"`
	// Cost
	TotalCostPH      float64 `json:"total_cost_ph" bson:"total_cost_ph"` // PH = per hour
	GpuCostPH        float64 `json:"gpu_cost_ph" bson:"gpu_cost_ph"`
	DiskCostPH       float64 `json:"disk_cost_ph" bson:"disk_cost_ph"`
	UploadCostPH     float64 `json:"upload_cost_ph" bson:"upload_cost_ph"`
	DownloadCostPH   float64 `json:"download_cost_ph" bson:"download_cost_ph"`
	FlopsPerDollarPH float64 `json:"flops_per_dollar_ph" bson:"flops_per_dollar_ph"`

	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
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
