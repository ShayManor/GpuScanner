package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ---- Lium public nodes feed (no key, no signup) ----
// GET https://lium.io/api/public/v1/nodes
// Ref: https://docs.lium.io/developers/public-nodes-feed
type liumNodesResp struct {
	GeneratedAt string     `json:"generated_at"`
	Count       int        `json:"count"`
	Nodes       []liumNode `json:"nodes"`
}

type liumNode struct {
	ID                string  `json:"id"`
	GPUModel          string  `json:"gpu_model"`
	GPUCount          int     `json:"gpu_count"`
	AvailableGPUCount int     `json:"available_gpu_count"`
	GPUMemoryGB       int     `json:"gpu_memory_gb"` // per GPU
	CPUCount          float64 `json:"cpu_count"`
	CPUModel          string  `json:"cpu_model"`
	RAMGB             int     `json:"ram_gb"`
	DiskGB            float64 `json:"disk_gb"`
	DiskType          string  `json:"disk_type"`

	NetworkUploadMbps   float64 `json:"network_upload_mbps"`
	NetworkDownloadMbps float64 `json:"network_download_mbps"`
	ReliabilityScore    float64 `json:"reliability_score"` // percent
	UptimeMinutes       int     `json:"uptime_minutes"`

	City    string `json:"city"`
	Country string `json:"country"`
	RentURL string `json:"rent_url"`

	PricePerGPUHour  float64 `json:"price_per_gpu_hour"`
	PricePerNodeHour float64 `json:"price_per_node_hour"`
}

// The feed publishes "unknown" for nodes whose disk type it has not resolved.
func liumDiskName(diskType string) string {
	if strings.EqualFold(diskType, "unknown") {
		return ""
	}
	return diskType
}

func parseLiumNodes(r io.Reader) ([]GPU, error) {
	var nr liumNodesResp
	if err := json.NewDecoder(r).Decode(&nr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0, len(nr.Nodes))
	for _, n := range nr.Nodes {
		if n.AvailableGPUCount <= 0 || n.PricePerNodeHour <= 0 {
			continue
		}
		totalFlops, memBWGBs, name := gpuSpecs(n.GPUModel)
		totalFlops = float64(n.GPUCount) * totalFlops / 1e12

		newGpu := GPU{
			_Id:         n.ID,
			Location:    strings.TrimSpace(fmt.Sprintf("%s, %s", n.City, n.Country)),
			Reliability: n.ReliabilityScore / 100.0, // feed gives percent
			Duration:    0,                          // not exposed

			Name:              name,
			Vram:              n.GPUMemoryGB * 1024,
			TotalFlops:        totalFlops,
			GpuMemoryBandwith: memBWGBs,
			NumGPUs:           n.GPUCount,

			CpuCores: n.CPUCount,
			CpuName:  n.CPUModel,
			CpuGhz:   0,  // not exposed
			CpuArch:  "", // not exposed

			Ram: n.RAMGB * 1024, // GB

			DiskSpace: n.DiskGB, // GB
			DiskBW:    0,        // not exposed
			DiskName:  liumDiskName(n.DiskType),

			UploadSpeed:   n.NetworkUploadMbps,
			DownloadSpeed: n.NetworkDownloadMbps,

			// Lium keeps the two prices separate: per GPU and per whole node.
			TotalCostPH:      n.PricePerNodeHour,
			GpuCostPH:        n.PricePerGPUHour,
			DiskCostPH:       0,
			UploadCostPH:     0,
			DownloadCostPH:   0,
			FlopsPerDollarPH: totalFlops / n.PricePerNodeHour,
			Source:           "lium",
			Url:              n.RentURL,
		}
		newGpu.Score = calculateScore(newGpu)
		newGpu.ScoreDPH = newGpu.Score / newGpu.TotalCostPH
		out = append(out, newGpu)
	}
	return out, nil
}

func liumGetter() ([]GPU, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get("https://lium.io/api/public/v1/nodes")
	if err != nil {
		return nil, fmt.Errorf("fetch lium nodes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("lium API status %s", resp.Status)
	}

	out, err := parseLiumNodes(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d Lium GPUs\n", len(out))
	return out, nil
}
