package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type LambdaSpecs struct {
	Ram         int `json:"memory_gib"`
	GPUs        int `json:"gpus"`
	StorageSize int `json:"storage_gib"`
	VCPUs       int `json:"vcpus"`
}

type LambdaInstance struct {
	PricePerHour   float64     `json:"price_cents_per_hour"`
	Specs          LambdaSpecs `json:"specs"`
	Description    string      `json:"description"`
	GPUDescription string      `json:"gpu_description"`
}

type LambdaRegion struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type LambdaInstanceType struct {
	Name     string         `json:"name"`
	Instance LambdaInstance `json:"instance_type"`
	Region   []LambdaRegion `json:"regions_with_capacity_available"`
}

type LambdaInstanceTypesResponse struct {
	Data map[string]LambdaInstanceType `json:"data"`
}

func fetchLambdaInstanceTypes() (map[string]LambdaInstanceType, error) {
	req, err := http.NewRequest("GET", "https://cloud.lambda.ai/api/v1/instance-types", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	token := os.Getenv("LAMBDA_TOKEN")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch lambda hostnodes: %w", err)
	}
	defer resp.Body.Close()
	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Body:", resp.Body)

	var response LambdaInstanceTypesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func getLambdaURL() string {
	return fmt.Sprintf("https://cloud.lambda.ai/instances")
}

func lambdaGetter() ([]GPU, error) {
	instanceTypes, err := fetchLambdaInstanceTypes()
	if err != nil {
		return nil, err
	}

	out := make([]GPU, 0, len(instanceTypes))

	for typeName, instance := range instanceTypes {
		// Extract GPU name from description (e.g., "8x NVIDIA A100 (40 GB)" -> "A100")
		gpuName := extractGPUName(instance.Instance.GPUDescription)

		// Convert price from cents to dollars per hour
		pricePerHour := instance.Instance.PricePerHour / 100.0

		// Extract VRAM from GPU description
		vram := extractVRAM(instance.Instance.GPUDescription)
		if len(instance.Region) > 0 {
			flops, _ := runpodGPULookup(typeName)
			flops = float64(instance.Instance.Specs.GPUs) * flops / 10e11
			region := instance.Region[0].Name
			out = append(out, GPU{
				Id:               typeName,
				Location:         region,
				Source:           "lambda",
				Url:              getLambdaURL(),
				Name:             gpuName,
				Vram:             vram * 1024,
				NumGPUs:          instance.Instance.Specs.GPUs,
				Reliability:      0.99,
				TotalFlops:       flops,
				FlopsPerDollarPH: flops / pricePerHour,

				UploadSpeed:   10000,
				DownloadSpeed: 10000,

				DiskBW: 12_000,

				CpuCores: float64(instance.Instance.Specs.VCPUs),
				Ram:      instance.Instance.Specs.Ram * 1000, // Convert GB to MB

				DiskSpace: float64(instance.Instance.Specs.StorageSize),
				DiskName:  "NVMe SSD",

				TotalCostPH: pricePerHour,
				GpuCostPH:   pricePerHour,
			})
		}

	}

	fmt.Printf("Found %d Lambda Labs GPUs\n", len(out))
	return out, nil
}

// Helper function to extract GPU name from description
func extractGPUName(description string) string {
	// Pattern: "8x NVIDIA A100 (40 GB)" -> "A100"
	parts := strings.Fields(description)
	for i, part := range parts {
		if strings.Contains(strings.ToUpper(part), "NVIDIA") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	// Fallback: return the whole description
	return description
}

// Helper function to extract VRAM from description
func extractVRAM(description string) int {
	// Pattern: "8x NVIDIA A100 (40 GB)" -> 40
	if idx := strings.Index(description, "("); idx != -1 {
		if endIdx := strings.Index(description[idx:], "GB"); endIdx != -1 {
			vramStr := strings.TrimSpace(description[idx+1 : idx+endIdx])
			vram, _ := strconv.Atoi(vramStr)
			return vram
		}
	}
	return 0
}
