// runpodGetter.go - Fetches GPU configurations from RunPod API - Written by Claude <3
// August 2025 values because the api does not return prices
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type rpGPUType struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	MemoryInGb  int    `json:"memoryInGb"`
}

type rpResp struct {
	Data struct {
		GpuTypes []rpGPUType `json:"gpuTypes"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// Updated prices from RunPod's website (August 2025)
var runpodPrices = map[string]float64{
	"RTX 3070":       0.11,
	"RTX 3080":       0.14,
	"RTX 3080 Ti":    0.16,
	"RTX 3090":       0.44,
	"3090":           0.44,
	"RTX 3090 Ti":    0.49,
	"RTX 4070 Ti":    0.35,
	"RTX 4080":       0.56,
	"RTX 4090":       0.74,
	"4090":           0.74,
	"RTX 5080":       0.85,
	"RTX 5090":       1.11,
	"5090":           1.11,
	"A100 PCIe":      2.09,
	"A100 SXM":       2.17,
	"A100":           2.09,
	"H100 PCIe":      3.35,
	"H100 SXM":       3.58,
	"H100 NVL":       3.89,
	"H100":           3.58,
	"H200 SXM":       3.99,
	"H200":           3.99,
	"L4":             0.48,
	"L40":            1.33,
	"L40S":           1.33,
	"A30":            0.69,
	"A40":            0.85,
	"RTX A2000":      0.11,
	"A2000":          0.11,
	"RTX A4000":      0.40,
	"A4000":          0.40,
	"RTX A4500":      0.40,
	"A4500":          0.40,
	"RTX A5000":      0.48,
	"A5000":          0.48,
	"RTX A6000":      0.85,
	"A6000":          0.85,
	"RTX 6000 Ada":   1.33,
	"6000 Ada":       1.33,
	"RTX 4000 Ada":   0.40,
	"4000 Ada":       0.40,
	"RTX 5000 Ada":   0.56,
	"5000 Ada":       0.56,
	"RTX 2000 Ada":   0.40,
	"2000 Ada":       0.40,
	"Tesla V100":     0.79,
	"V100":           0.79,
	"V100 FHHL":      0.79,
	"V100 SXM2":      0.89,
	"V100 SXM2 32GB": 0.99,
	"MI300X":         4.89,
	"B200":           5.99,
	"T4":             0.39,
}

func getRunPodURL(o GPU) string {
	// Format: https://www.runpod.io/console/gpu-cloud/secure-cloud
	gpuParam := strings.ReplaceAll(strings.ToUpper(o.Name), " ", "%20")

	// RunPod doesn't have direct filtering via URL params like Vast
	// Best we can do is link to the marketplace with a GPU type hint
	return fmt.Sprintf(
		"https://www.console.runpod.io/deploy/?gpu=%s&count=%d&template=runpod-torch-v280",
		gpuParam,
		o.NumGPUs,
	)
}

func runpodGetter() ([]GPU, error) {
	apiKey := strings.TrimSpace(os.Getenv("RUNPOD_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("RUNPOD_API_KEY is not set")
	}

	query := `
query {
  gpuTypes {
    id
    displayName
    memoryInGb
  }
}`

	body, _ := json.Marshal(map[string]string{"query": query})
	req, err := http.NewRequest("POST", "https://api.runpod.io/graphql", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch runpod gpuTypes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("runpod API status %s", resp.Status)
	}

	var rr rpResp
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0, len(rr.Data.GpuTypes)*2)

	for _, t := range rr.Data.GpuTypes {
		// Skip unknown GPUs
		if t.DisplayName == "unknown" || t.ID == "unknown" {
			continue
		}

		// Get price from our updated table
		price := 0.0
		found := false

		// Try to match price by display name
		for key, p := range runpodPrices {
			if strings.Contains(strings.ToLower(t.DisplayName), strings.ToLower(key)) {
				price = p
				found = true
				break
			}
		}

		// If not found, try ID
		if !found {
			for key, p := range runpodPrices {
				if strings.Contains(strings.ToLower(t.ID), strings.ToLower(key)) {
					price = p
					found = true
					break
				}
			}
		}

		// Skip if no price found
		if price <= 0 {
			continue
		}

		// VRAM calculation
		vramMB := t.MemoryInGb * 1024
		if vramMB == 0 {
			vramMB = parseVRAMMB(t.DisplayName)
			if vramMB == 0 {
				vramMB = parseVRAMMB(t.ID)
			}
		}

		// Get FLOPS and bandwidth using enhanced lookup
		totalFlops, memBW := gpuSpecs(t.DisplayName)

		// Create configurations for different GPU counts
		gpuCounts := []int{1}

		// Add multi-GPU configs for high-end cards
		if strings.Contains(t.DisplayName, "A100") || strings.Contains(t.DisplayName, "H100") ||
			strings.Contains(t.DisplayName, "H200") || strings.Contains(t.DisplayName, "4090") ||
			strings.Contains(t.DisplayName, "3090") {
			gpuCounts = append(gpuCounts, 2, 4)
		}

		for _, gpuCount := range gpuCounts {
			// Scale resources
			totalSystemFlops := totalFlops * float64(gpuCount) / 1e12

			totalPrice := price * float64(gpuCount)

			// Estimate other resources based on GPU type and count
			vcpus := gpuCount * 8
			memory := gpuCount * 32 * 1024
			disk := gpuCount * 100

			// Premium GPUs get more resources
			if strings.Contains(t.DisplayName, "A100") || strings.Contains(t.DisplayName, "H100") ||
				strings.Contains(t.DisplayName, "H200") || strings.Contains(t.DisplayName, "MI300X") ||
				strings.Contains(t.DisplayName, "B200") {
				vcpus = gpuCount * 16
				memory = gpuCount * 64 * 1024
				disk = gpuCount * 200
			}

			// Calculate flops per dollar
			flopsPerDollar := 0.0
			if totalPrice > 0 && totalSystemFlops > 0 {
				flopsPerDollar = totalSystemFlops / totalPrice
			}

			// Determine reliability and cloud type based on price
			reliability := 0.95 // Community cloud default
			cloudType := "Community Cloud"
			if price > 1.5 {
				reliability = 0.99 // Premium GPUs usually on secure cloud
				cloudType = "Secure Cloud"
			}

			newGpu := GPU{
				Id:          fmt.Sprintf("%s-%dx", strings.ReplaceAll(t.ID, " ", "-"), gpuCount),
				Location:    cloudType,
				Reliability: reliability,
				Duration:    0, // not exposed

				Source:            "runpod",
				Name:              t.DisplayName,
				Vram:              vramMB,
				TotalFlops:        totalSystemFlops,
				GpuMemoryBandwith: memBW,
				NumGPUs:           gpuCount,

				CpuCores: float64(vcpus),
				CpuName:  "AMD EPYC",
				CpuGhz:   2.5,
				CpuArch:  "x86_64",

				Ram: memory,

				DiskSpace: float64(disk),
				DiskBW:    2000, // NVMe typical bandwidth
				DiskName:  "NVMe SSD",

				UploadSpeed:   10000, // 10 Gbps default
				DownloadSpeed: 10000, // 10 Gbps default

				TotalCostPH:      totalPrice,
				GpuCostPH:        totalPrice,
				DiskCostPH:       0, // Included in total
				UploadCostPH:     0, // Included in total
				DownloadCostPH:   0, // Included in total
				FlopsPerDollarPH: flopsPerDollar,
			}
			newGpu.Url = getRunPodURL(newGpu)
			out = append(out, newGpu)
		}
	}

	fmt.Printf("Found %d RunPod GPU configurations\n", len(out))
	return out, nil
}
