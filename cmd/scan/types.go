package main

import (
	"fmt"
	"strings"
	"time"
)

type GPU struct {
	// Instance details
	Id          string  `json:"id" bson:"id"`   // UUID
	_Id         string  `json:"_id" bson:"_id"` // Set by providers
	Location    string  `json:"location" bson:"location"`
	Reliability float64 `json:"reliability" bson:"reliability"`
	Duration    float64 `json:"duration_hours" bson:"duration_hours"`
	Source      string  `json:"source" bson:"source"` // e.g., "tensordock", "vast", etc.
	Url         string  `json:"url" bson:"url"`
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
		"Url: %s\n"+
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
		g._Id,
		g.Name,
		g.Location,
		g.Url,
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

func gpuSpecs(displayName string) (flops float64, memBWGBs float64) {
	n := strings.ToLower(displayName)
	n = strings.ReplaceAll(n, "_", " ")
	n = strings.ReplaceAll(n, "-", " ")

	// helper
	has := func(ss ...string) bool {
		for _, s := range ss {
			if strings.Contains(n, s) {
				return true
			}
		}
		return false
	}

	// ---------- GeForce / Ada & Ampere (consumer) ----------
	switch {
	case has("5090", "rtx 5090", "rtx5090"):
		return 104.8e12, 1792
	case has("5080", "rtx 5080", "rtx5080"):
		return 56.3e12, 960
	case has("4090", "rtx 4090", "rtx4090"):
		return 82.6e12, 1008
	case has("4080", "rtx 4080"):
		return 48.7e12, 716.8
	case has("4070", "rtx 4070"):
		return 29.1e12, 504
	case has("3080 ti", "rtx 3080 ti", "3080ti"):
		return 34.1e12, 912
	case has("3080", "rtx 3080"):
		return 29.8e12, 760 // (10GB model; providers usually list this one)
	case has("3070", "rtx 3070"):
		return 20.3e12, 448
	case has("3090", "rtx 3090"):
		return 35.6e12, 936
	}

	// ---------- NVIDIA Data Center ----------
	// A100 (Ampere) variants
	if has("a100") {
		sxm := has("sxm")
		gb80 := has("80gb", "80g", " 80 ")
		gb40 := has("40gb", "40g", " 40 ")
		switch {
		case sxm && gb80:
			return 19.5e12, 2039
		case sxm && gb40:
			return 19.5e12, 1555
		case !sxm && gb80:
			return 19.5e12, 1935
		case !sxm && gb40:
			return 19.5e12, 1555
		default: // unknown capacity -> assume PCIe 80GB
			return 19.5e12, 1935
		}
	}

	// H100 (Hopper) variants
	if has("h100") {
		// Treat NVL like PCIe for per-GPU FP32/BW (two PCIe H100s bridged)
		if has("sxm") {
			return 67e12, 3350
		}
		return 51e12, 2000 // PCIe & NVL per-GPU
	}

	// H200
	if has("h200") {
		return 67e12, 4800
	}

	if has("gb200") || has("gb200") {
		return 5760e12, 8000
	}

	// B200 (Blackwell) — public FP32/BW still in flux
	if has("b200") || has("B200") {
		return 2200e12, 8000
	}

	// L40 family (Ada data center / pro)
	switch {
	case has("l40s"):
		return 91.6e12, 864
	case has("l40") && !has("l40s"):
		return 90.5e12, 864
	case has("l4"):
		return 30.3e12, 300
	}

	// Ampere DC midrange / workstation
	switch {
	case has("a10"):
		return 31.2e12, 600
	case has("a30"):
		return 10.3e12, 933
	case has("a40"):
		return 37.4e12, 696
	}

	// Professional/Workstation (RTX A-series & Ada Pro)
	switch {
	case has("rtx a6000", "rtxa6000", "a6000", "6000ada"):
		return 38.7e12, 768
	case has("rtx a5000", "rtxa5000", "a5000", "5000 ada", "5000ada"): // include Ada 5000 wording you had
		// Prefer exact A5000 mapping unless the string clearly says "ada"
		if has("ada") {
			return 65.3e12, 640 // RTX 5000 Ada
		}
		return 27.8e12, 768 // RTX A5000 (Ampere)
	case has("rtx a4500", "rtxa4500", "a4500"):
		return 23.7e12, 640
	case has("rtx a4000", "rtxa4000", "a4000"):
		return 19.2e12, 448
	case has("rtx a2000", "rtxa2000", "a2000"):
		return 8e12, 288

	// Ada “Pro RTX” naming used by some marketplaces
	case has("6000 ada", "rtx 6000 ada", "pro 6000", "pro6000"):
		return 91.1e12, 960
	case has("4000 ada", "rtx 4000 ada"):
		return 26.7e12, 360
	case has("2000 ada", "rtx 2000 ada"):
		return 12e12, 288
	}

	// Tesla/Older DC
	switch {
	case has("v100"):
		if has("32gb", "32g") {
			return 15.7e12, 900
		}
		return 14e12, 900
	case has("t4"):
		return 8.1e12, 300
	}

	// AMD Instinct
	switch {
	case has("mi300x"):
		return 163.4e12, 5300
	case has("mi250x"): // allow both
		return 95.7e12, 3200 // FP32 (matrix) headline; bandwidth per AMD
	case has("mi250"):
		return 90.5e12, 3200 // MI250 (non-X) matrix FP32; bandwidth
	}

	// Fallback: unknown
	return 0, 0
}
