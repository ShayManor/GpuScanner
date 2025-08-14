package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
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
	ID               int     `json:"machine_id"`
	GPUName          string  `json:"gpu_name"`
	CPUCores         float64 `json:"cpu_cores_effective"`
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

func fetchVastOffers(limit int) ([]offer, error) {
	body := fmt.Sprintf(`{"q":{"limit":%d,"rentable":"true"}}`, limit)
	fmt.Println(string(body))
	req, _ := http.NewRequest("PUT", "https://console.vast.ai/api/v0/search/asks/", strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var sr Response
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	fmt.Println(sr.Offers)
	return sr.Offers, nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// convertGPUNameToURLFormat converts GPU names from API format to URL parameter format
// Examples: "RTX 4090" -> "rtx4090", "RTX 4090 D" -> "rtx4090D", "H200 NVL" -> "h200Nvl"
func convertGPUNameToURLFormat(gpuName string) string {
	// Handle empty string
	if gpuName == "" {
		return ""
	}

	// Convert to lowercase and split by spaces
	parts := strings.Fields(strings.ToLower(gpuName))
	if len(parts) == 0 {
		return ""
	}

	// Start with the first part (e.g., "rtx", "h200", "b200")
	result := parts[0]
	if len(parts) > 1 && parts[0] == "rtx" {
		name := parts[1]
		switch {
		case strings.HasSuffix(name, "ti") && isAllDigits(name[:len(name)-2]):
			parts[1] = name[:len(name)-2]
			parts = append(parts, "ti")
		case strings.HasSuffix(name, "s") && isAllDigits(name[:len(name)-1]):
			// Guard against L40S-style workstation naming: we only do this for RTX + digits
			parts[1] = name[:len(name)-1]
			parts = append(parts, "s") // will map to "Super" below
		}
	}

	// Process remaining parts - capitalize first letter of each
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if len(part) > 0 {
			// Special cases for known suffixes
			switch strings.ToLower(part) {
			case "ti":
				result += "Ti"
			case "s", "super":
				result += "Super"
			case "nvl":
				result += "Nvl"
			case "sxm":
				result += "Sxm"
			case "ada":
				result += "Ada"
			case "generation":
				result += "Generation"
			case "blackwell":
				result += "Blackwell"
			case "workstation":
				result += "Workstation"
			case "laptop":
				result += "Laptop"
			case "d":
				result += "D"
			default:
				// Capitalize first letter
				result += strings.Title(part)
			}
		}
	}

	// Special handling for "RTX A" series (e.g., "RTX A6000" -> "rtxA6000")
	result = strings.Replace(result, "rtxa", "rtxA", 1)
	result = strings.Replace(result, "rtxpro", "rtxPro", 1)

	return result
}

func vastGetter() ([]GPU, error) {
	sr, err := fetchVastOffers(4096)
	if err != nil {
		return nil, err
	}

	out := make([]GPU, 0, len(sr))
	for _, o := range sr {
		if o.Rentable {
			urlParams := fmt.Sprintf(
				"gpuModelNames=%s&"+
					"instanceType=onDemand&"+
					"isOfferAvailable=true&"+
					"isOfferCompatible=true&"+
					"isOfferVerified=%t&"+
					"machineCpuCoresMin=%.1f&"+
					"machineCpuRamMin=8000&"+
					"instanceDiskSizeMin=%.1f&"+
					"machineReliabilityMin=%.2f&"+
					"machineReliabilityMax=%.2f&"+
					// Explicitly reset all other filters to defaults
					"isHostSecure=false&"+
					"isMachineIpStatic=false&"+
					"isAvxSupported=false&"+
					"isQueryInverted=false&"+
					"instanceDurationMin=0&"+
					"machineMegabitDownloadMin=0&"+
					"machineMegabitUploadMin=0&"+
					"machineCpuCoresMax=512&"+
					"machineCpuRamMax=8000000&"+ // Empty = no max
					"isOfferCompatible=false&"+
					"instanceDiskSizeMin=32&"+
					"sorts=priceInstanceHourly-asc&"+
					"priceInstanceHourlyMax=%.4f&"+
					"priceInstanceHourlyMin=%.4f&"+
					"pageSize=256",
				convertGPUNameToURLFormat(o.GPUName),
				o.Verified,
				math.Max(0, o.CPUCores-0.1),
				math.Max(0, o.DiskSpace-0.1),
				math.Max(0, o.Reliability-0.01),
				math.Max(0, o.Reliability+0.01),
				o.DPHTotal+0.01,
				o.DPHTotal-0.01,
			)

			out = append(out, GPU{
				_Id:               strconv.Itoa(o.ID) + "v",
				Location:          o.Location,
				Reliability:       o.Reliability,
				Duration:          o.Duration,
				Source:            "vast",
				Url:               fmt.Sprintf("https://cloud.vast.ai/create/?%s", urlParams),
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
	fmt.Println("Found", len(out), "Vast GPUs")
	return out, nil
}
