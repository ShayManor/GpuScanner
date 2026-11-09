package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ---- Theta EdgeCloud VM catalog (no key) ----
// GET https://api.thetaedgecloud.com/resource/vm/list
// Every field is a string: cpu in millicores, mem in MB, storage in GB and
// price_hour in US cents for the whole VM.
type thetaResp struct {
	Body struct {
		VMs []thetaVM `json:"vms"`
	} `json:"body"`
}

type thetaVM struct {
	ID        string `json:"id"`
	Name      string `json:"name"` // e.g. "G-H100_80GB-x1"
	Resources struct {
		CPU      string `json:"cpu"`
		GPU      string `json:"gpu"`
		Mem      string `json:"mem"`
		Storage  string `json:"storage"`
		GPUModel string `json:"gpu_model"` // e.g. "nvidia_H100_80G"
	} `json:"resources"`
	PriceHour string `json:"price_hour"`
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

// "G-H100_80GB-x1" -> 80; 0 when the name carries no size.
func thetaVramGB(name string) int {
	for _, part := range strings.FieldsFunc(name, func(r rune) bool { return r == '_' || r == '-' }) {
		if strings.HasSuffix(part, "GB") {
			if v, err := strconv.Atoi(strings.TrimSuffix(part, "GB")); err == nil {
				return v
			}
		}
	}
	return 0
}

func parseTheta(r io.Reader) ([]GPU, error) {
	var tr thetaResp
	if err := json.NewDecoder(r).Decode(&tr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0, len(tr.Body.VMs))
	for _, vm := range tr.Body.VMs {
		num := int(atof(vm.Resources.GPU))
		price := atof(vm.PriceHour) / 100.0
		if num <= 0 || price <= 0 || !strings.HasPrefix(vm.Resources.GPUModel, "nvidia") {
			continue
		}
		perGPUFlops, memBWGBs, name := gpuSpecs(vm.Resources.GPUModel)
		if name == "unknown" {
			continue
		}
		totalFlops := float64(num) * perGPUFlops / 1e12

		newGpu := GPU{
			_Id: vm.ID,

			Name:              name,
			Vram:              thetaVramGB(vm.Name) * 1024,
			TotalFlops:        totalFlops,
			GpuMemoryBandwith: memBWGBs,
			NumGPUs:           num,

			CpuCores: atof(vm.Resources.CPU) / 1000.0,
			Ram:      int(atof(vm.Resources.Mem)),

			DiskSpace: atof(vm.Resources.Storage),

			TotalCostPH:      price,
			GpuCostPH:        price / float64(num),
			FlopsPerDollarPH: totalFlops / price,
			Source:           "theta",
			Url:              "https://www.thetaedgecloud.com/",
		}
		newGpu.Score = calculateScore(newGpu)
		newGpu.ScoreDPH = newGpu.Score / newGpu.TotalCostPH
		out = append(out, newGpu)
	}
	return out, nil
}

func thetaGetter() ([]GPU, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get("https://api.thetaedgecloud.com/resource/vm/list")
	if err != nil {
		return nil, fmt.Errorf("fetch theta vms: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("theta API status %s", resp.Status)
	}

	out, err := parseTheta(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d Theta EdgeCloud GPUs\n", len(out))
	return out, nil
}
