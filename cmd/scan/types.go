package main

import (
	"fmt"
	"math"
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
	Score       float64 `json:"score" bson:"score"`
	ScoreDPH    float64 `json:"score_dollar_ph" bson:"score_dollar_ph"`
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
		"Score: %.2f\n"+
		"Score:/$/Hour %.2f\n"+
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
		g.Score,
		g.ScoreDPH,
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

func gpuSpecs(displayName string) (flops float64, memBWGBs float64, name string) {
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
		return 104.8e12, 1792, "RTX 5090"
	case has("5080", "rtx 5080", "rtx5080"):
		return 56.3e12, 960, "rtx 5080"
	case has("4090", "rtx 4090", "rtx4090"):
		return 82.6e12, 1008, "RTX 4090"
	case has("4080", "rtx 4080"):
		return 48.7e12, 716.8, "RTX 4080"
	case has("4070", "rtx 4070"):
		return 29.1e12, 504, "RTX 4070"
	case has("3080 ti", "rtx 3080 ti", "3080ti"):
		return 34.1e12, 912, "RTX 3080 ti"
	case has("3080", "rtx 3080"):
		return 29.8e12, 760, "RTX 3080" // (10GB model; providers usually list this one)
	case has("3070", "rtx 3070"):
		return 20.3e12, 448, "RTX 3070"
	case has("3090", "rtx 3090"):
		return 35.6e12, 936, "RTX 3090"
	}

	// ---------- NVIDIA Data Center ----------
	// A100 (Ampere) variants
	if has("a100") {
		sxm := has("sxm")
		gb80 := has("80gb", "80g", " 80 ")
		gb40 := has("40gb", "40g", " 40 ")
		switch {
		case sxm && gb80:
			return 19.5e12, 2039, "A100 SXM4"
		case sxm && gb40:
			return 19.5e12, 1555, "A100 SXM4"
		case !sxm && gb80:
			return 19.5e12, 1935, "A100 PCIE"
		case !sxm && gb40:
			return 19.5e12, 1555, "A100 PCIE"
		default: // unknown capacity -> assume PCIe 80GB
			return 19.5e12, 1935, "A100"
		}
	}

	// H100 (Hopper) variants
	if has("h100") {
		// Treat NVL like PCIe for per-GPU FP32/BW (two PCIe H100s bridged)
		if has("sxm") {
			return 67e12, 3350, "H100 SXM"
		}
		return 51e12, 2000, "H100 PCIE" // PCIe & NVL per-GPU
	}

	// H200
	if has("h200") {
		return 67e12, 4800, "H200"
	}

	if has("gb200") || has("gb200") {
		return 5760e12, 8000, "GB200"
	}

	// B200 (Blackwell) — public FP32/BW still in flux
	if has("b200") || has("B200") {
		return 2200e12, 8000, "B200"
	}

	// L40 family (Ada data center / pro)
	switch {
	case has("l40s"):
		return 91.6e12, 864, "l40s"
	case has("l40") && !has("l40s"):
		return 90.5e12, 864, "l40"
	case has("l4"):
		return 30.3e12, 300, "l4"
	}

	// Ampere DC midrange / workstation
	switch {
	case has("a10"):
		return 31.2e12, 600, "a10"
	case has("a30"):
		return 10.3e12, 933, "a30"
	case has("a40"):
		return 37.4e12, 696, "a40"
	}

	// Professional/Workstation (RTX A-series & Ada Pro)
	switch {
	case has("rtx a6000", "rtxa6000", "a6000", "6000ada"):
		return 38.7e12, 768, "RTX a6000"
	case has("rtx a5000", "rtxa5000", "a5000", "5000 ada", "5000ada"): // include Ada 5000 wording you had
		// Prefer exact A5000 mapping unless the string clearly says "ada"
		if has("ada") {
			return 65.3e12, 640, "RTX A5000 ada" // RTX 5000 Ada
		}
		return 27.8e12, 768, "RTX A5000" // RTX A5000 (Ampere)
	case has("rtx a4500", "rtxa4500", "a4500"):
		return 23.7e12, 640, "RTX A4500"
	case has("rtx a4000", "rtxa4000", "a4000"):
		return 19.2e12, 448, "RTX A4000"
	case has("rtx a2000", "rtxa2000", "a2000"):
		return 8e12, 288, "RTX A2000"

	// Ada “Pro RTX” naming used by some marketplaces
	case has("6000 ada", "rtx 6000 ada", "pro 6000", "pro6000"):
		return 91.1e12, 960, "RTX 6000 ada pro"
	case has("4000 ada", "rtx 4000 ada"):
		return 26.7e12, 360, "RTX 4000 ada"
	case has("2000 ada", "rtx 2000 ada"):
		return 12e12, 288, "RTX 2000 ada"
	}

	// Tesla/Older DC
	switch {
	case has("v100"):
		if has("32gb", "32g") {
			return 15.7e12, 900, "V100"
		}
		return 14e12, 900, "V100"
	case has("t4"):
		return 8.1e12, 300, "t4"
	}

	// AMD Instinct
	switch {
	case has("mi300x"):
		return 163.4e12, 5300, "mi300x"
	case has("mi250x"): // allow both
		return 95.7e12, 3200, "mi250x" // FP32 (matrix) headline; bandwidth per AMD
	case has("mi250"):
		return 90.5e12, 3200, "mi250" // MI250 (non-X) matrix FP32; bandwidth
	}

	// Fallback: unknown
	return 0, 0, "unknown"
}

// calculateScore returns a 0..100 score focused on hardware capability and stability.
// Only these fields are considered: NumGPUs, VRAM, TotalFlops, GpuMemoryBandwith,
// CpuCores, Ram, Reliability.
func calculateScore(g GPU) float64 {
	// --- Safety & derived values ---
	num := float64(maxInt(g.NumGPUs, 1))

	// Treat TotalFlops as node-total if multi-GPU; reduce to per-GPU for fair normalization.
	perGPUFlops := safeDiv(g.TotalFlops, num) // if provider already provides per-GPU, NumGPUs==1 → no change

	// VRAM per GPU (GB)
	perGPUVramGB := float64(g.Vram) / 1024.0

	// GPU memory bandwidth per GPU.
	// NOTE: We don't assume exact units (GB/s vs Gbit/s). The cap below is deliberately high to be unit-tolerant.
	memBW := g.GpuMemoryBandwith

	// System RAM (GB)
	sysRamGB := float64(g.Ram) / 1024.0

	// Reliability clamped to [0,1]
	rel := clamp01(g.Reliability)

	// --- Normalization caps (tunable). Chosen to comfortably cover 30–40 series, A40/L40, H100/H200, MI300X, GB200. ---
	const (
		capPerGPUFlops = 3000.0 // TFLOPS (BF16/FP16 class). Plenty for H200/MI300/GB200; older cards scale below.
		capMemBW       = 6000.0 // "GB/s"-ish; roomy enough to absorb HBM3e. If data is Gbit/s, still ranks relatively.
		capVramGB      = 192.0  // 192 GB per GPU (MI300X). H200 (141GB), H100 (80GB) scale well below.
		capNumGPUs     = 8.0    // Diminishing returns saturate by 8; >8 yields tiny incremental value.
		capCpuCores    = 128.0  // Dual-socket EPYCs, Grace/CPU heavy nodes handled.
		capSysRamGB    = 2048.0 // 2 TB ceiling for normalization.
	)

	// Normalizers
	nFlops := normCapped(perGPUFlops, capPerGPUFlops)
	nMemBW := normCapped(memBW, capMemBW)
	nVram := normCapped(perGPUVramGB, capVramGB)
	nCpu := normCapped(g.CpuCores, capCpuCores)
	nRam := normCapped(sysRamGB, capSysRamGB)

	// Diminishing returns on multi-GPU count (smooth, monotone). sqrt is simple and robust.
	nNum := normCapped(math.Sqrt(num), math.Sqrt(capNumGPUs))

	// Combine compute & bandwidth for a vendor-neutral GPU “core” score.
	// If bandwidth is missing/zero, fall back to compute.
	var nGpuCore float64
	if nMemBW > 0 {
		// Geometric mean rewards balance (memory-bound vs compute-bound).
		nGpuCore = math.Sqrt(nFlops * nMemBW)
	} else {
		nGpuCore = nFlops
	}

	// --- Weights (sum = 1.0) ---
	// Emphasis on GPU capability and VRAM; reliability is meaningful but not dominant.
	const (
		wGpuCore = 0.3
		wVram    = 0.18
		wNum     = 0.26
		wRel     = 0.12
		wCpu     = 0.07
		wRam     = 0.07
	)

	score01 := wGpuCore*nGpuCore +
		wVram*nVram +
		wNum*nNum +
		wRel*rel +
		wCpu*nCpu +
		wRam*nRam

	return clamp(score01*100.0, 0, 100)
}

// --- helpers ---

func normCapped(v, cap float64) float64 {
	if cap <= 0 {
		return 0
	}
	return clamp(v/cap, 0, 1)
}

func clamp01(x float64) float64 { return clamp(x, 0, 1) }

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
