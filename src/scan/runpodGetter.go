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

// Enhanced GPU hardware lookup specifically for RunPod display names
func runpodGPULookup(displayName string) (totalFlops float64, memBWGBs float64) {
	// Clean up the display name for matching
	name := strings.ToLower(displayName)

	// RTX Consumer cards
	if strings.Contains(name, "4090") || strings.Contains(name, "rtx 4090") {
		return 82.6e12, 1008
	}
	if strings.Contains(name, "3090") || strings.Contains(name, "rtx 3090") {
		return 35.6e12, 936
	}
	if strings.Contains(name, "3080") {
		if strings.Contains(name, "ti") {
			return 34.1e12, 912
		}
		return 29.8e12, 760
	}
	if strings.Contains(name, "3070") {
		return 20.3e12, 448
	}
	if strings.Contains(name, "4070") {
		return 29.1e12, 504
	}
	if strings.Contains(name, "4080") {
		return 48.7e12, 716
	}
	if strings.Contains(name, "5090") {
		return 150e12, 1792 // Estimated based on rumors
	}
	if strings.Contains(name, "5080") {
		return 100e12, 960 // Estimated
	}

	// Data center cards
	if strings.Contains(name, "a100") {
		if strings.Contains(name, "sxm") {
			return 19.5e12, 2039 // A100 SXM 80GB
		}
		return 19.5e12, 1935 // A100 PCIe 80GB
	}
	if strings.Contains(name, "h100") {
		if strings.Contains(name, "nvl") {
			return 67e12, 3350 // H100 NVL
		}
		if strings.Contains(name, "sxm") {
			return 67e12, 3350 // H100 SXM
		}
		return 51e12, 2000 // H100 PCIe
	}
	if strings.Contains(name, "h200") {
		return 67e12, 4800 // H200 SXM
	}
	if strings.Contains(name, "b200") {
		return 144e12, 8000 // B200 estimated
	}

	// Professional/Workstation cards
	if strings.Contains(name, "l40s") {
		return 91.6e12, 864
	}
	if strings.Contains(name, "l40") && !strings.Contains(name, "l40s") {
		return 90.5e12, 864
	}
	if strings.Contains(name, "l4") {
		return 30.3e12, 300
	}
	if strings.Contains(name, "a30") {
		return 10.3e12, 933
	}
	if strings.Contains(name, "a40") {
		return 37.4e12, 696
	}
	if strings.Contains(name, "a6000") || strings.Contains(name, "rtx a6000") {
		return 38.7e12, 768
	}
	if strings.Contains(name, "a5000") || strings.Contains(name, "rtx a5000") {
		return 27.8e12, 768
	}
	if strings.Contains(name, "a4500") || strings.Contains(name, "rtx a4500") {
		return 23.7e12, 640
	}
	if strings.Contains(name, "a4000") || strings.Contains(name, "rtx a4000") {
		return 19.2e12, 448
	}
	if strings.Contains(name, "a2000") || strings.Contains(name, "rtx a2000") {
		return 8e12, 288
	}
	if strings.Contains(name, "6000 ada") || strings.Contains(name, "rtx 6000") {
		return 91.1e12, 960
	}
	if strings.Contains(name, "5000 ada") {
		return 65.3e12, 640
	}
	if strings.Contains(name, "4000 ada") {
		return 26.7e12, 360
	}
	if strings.Contains(name, "2000 ada") {
		return 12e12, 288
	}

	// Tesla cards
	if strings.Contains(name, "v100") {
		if strings.Contains(name, "32gb") {
			return 15.7e12, 900
		}
		return 14e12, 900
	}
	if strings.Contains(name, "t4") {
		return 8.1e12, 300
	}

	// AMD cards
	if strings.Contains(name, "mi300x") {
		return 163.4e12, 5300
	}
	if strings.Contains(name, "mi250") {
		return 95.7e12, 3200
	}

	// Default fallback - try the original lookup
	flops, _, bw, _ := lookupGPUHardware(displayName)
	return flops, bw
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
		totalFlops, memBW := runpodGPULookup(t.DisplayName)

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
			totalSystemFlops := totalFlops * float64(gpuCount)
			totalPrice := price * float64(gpuCount)

			// Estimate other resources based on GPU type and count
			vcpus := gpuCount * 8
			memory := gpuCount * 32
			disk := gpuCount * 100

			// Premium GPUs get more resources
			if strings.Contains(t.DisplayName, "A100") || strings.Contains(t.DisplayName, "H100") ||
				strings.Contains(t.DisplayName, "H200") || strings.Contains(t.DisplayName, "MI300X") ||
				strings.Contains(t.DisplayName, "B200") {
				vcpus = gpuCount * 16
				memory = gpuCount * 64
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

			out = append(out, GPU{
				id:          fmt.Sprintf("%s-%dx", strings.ReplaceAll(t.ID, " ", "-"), gpuCount),
				location:    cloudType,
				reliability: reliability,
				duration:    0, // not exposed

				source:            "runpod",
				name:              t.DisplayName,
				vram:              vramMB,
				totalFlops:        totalSystemFlops,
				gpuMemoryBandwith: memBW,
				numGPUs:           gpuCount,

				cpuCores: float64(vcpus),
				cpuName:  "AMD EPYC",
				cpuGhz:   2.5,
				cpuArch:  "x86_64",

				ram: memory,

				diskSpace: float64(disk),
				diskBW:    2000, // NVMe typical bandwidth
				diskName:  "NVMe SSD",

				uploadSpeed:   10000, // 10 Gbps default
				downloadSpeed: 10000, // 10 Gbps default

				totalCostPH:      totalPrice,
				gpuCostPH:        totalPrice,
				diskCostPH:       0, // Included in total
				uploadCostPH:     0, // Included in total
				downloadCostPH:   0, // Included in total
				flopsPerDollarPH: flopsPerDollar,
			})
		}
	}

	fmt.Printf("Found %d RunPod GPU configurations\n", len(out))
	for _, o := range out {
		fmt.Println(o.toString())
	}
	return out, nil
}
