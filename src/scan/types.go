package main

import "fmt"

type GPU struct {
	// Instance details
	id          int64
	location    string
	reliability float64
	duration    float64
	// GPU details
	name              string
	vram              int
	totalFlops        float64
	gpuLanes          int
	gpuMemoryBandwith float64
	architecture      string
	numGPUs           int
	// CPU specs
	cpuCores   float64
	cpuName    string
	cpuGhz     float64
	cpuArch    string
	computeCap int
	// Ram
	ram int
	// SSD
	diskSpace float64
	diskBW    float64
	diskName  string
	// Internet
	uploadSpeed   float64
	downloadSpeed float64
	// Cost
	totalCostPH      float64 // PH = per hour
	gpuCostPH        float64
	diskCostPH       float64
	uploadCostPH     float64
	downloadCostPH   float64
	flopsPerDollarPH float64
}

type Getter func() ([]GPU, error)

func (g GPU) toString() string {
	return fmt.Sprintf("GPU Details:\n"+
		"ID: %d\n"+
		"Name: %s\n"+
		"Location: %s\n"+
		"Architecture: %s\n"+
		"Total GPUs: %d\n"+
		"VRAM: %d MB\n"+
		"Total FLOPS: %.2e\n"+
		"FLOPS/Dollar/Hour: %.2f\n"+
		"Reliability: %.2f%%\n"+
		"Duration: %.2f hours\n"+
		"GPU Lanes: %d\n"+
		"GPU Memory Bandwidth: %.2f GB/s\n"+
		"CPU: %s (%s)\n"+
		"CPU Cores: %.1f\n"+
		"CPU Clock: %.2f GHz\n"+
		"Compute Capability: %d\n"+
		"RAM: %d GB\n"+
		"Storage: %.2f GB (%s)\n"+
		"Disk Bandwidth: %.2f GB/s\n"+
		"Network: Up %.2f Mbps, Down %.2f Mbps\n"+
		"Hourly Costs:\n"+
		"  Total: $%.2f\n"+
		"  GPU: $%.2f\n"+
		"  Disk: $%.4f\n"+
		"  Upload: $%.6f\n"+
		"  Download: $%.6f",
		g.id,
		g.name,
		g.location,
		g.architecture,
		g.numGPUs,
		g.vram,
		g.totalFlops,
		g.flopsPerDollarPH,
		g.reliability*100,
		g.duration,
		g.gpuLanes,
		g.gpuMemoryBandwith,
		g.cpuName,
		g.cpuArch,
		g.cpuCores,
		g.cpuGhz,
		g.computeCap,
		g.ram,
		g.diskSpace,
		g.diskName,
		g.diskBW,
		g.uploadSpeed,
		g.downloadSpeed,
		g.totalCostPH,
		g.gpuCostPH,
		g.diskCostPH,
		g.uploadCostPH,
		g.downloadCostPH,
	)
}
