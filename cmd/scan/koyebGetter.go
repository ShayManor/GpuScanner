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

// ---- Koyeb public instance catalog (no key) ----
// GET https://app.koyeb.com/v1/catalog/instances?limit=100
// GPU instances carry a gpu block; an empty regions list means the instance
// cannot be deployed anywhere right now.
type koyebResp struct {
	Instances []koyebInstance `json:"instances"`
}

type koyebInstance struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Status      string   `json:"status"`
	VCPU        float64  `json:"vcpu"`
	Memory      string   `json:"memory"` // e.g. "180GB"
	Disk        string   `json:"disk"`
	PriceHourly string   `json:"price_hourly"`
	Regions     []string `json:"regions"`
	GPU         *struct {
		Count  int    `json:"count"`
		Brand  string `json:"brand"`
		Memory string `json:"memory"` // whole instance, e.g. "160GB"
		Name   string `json:"name"`   // e.g. "gpu-nvidia-a100-sxm"
	} `json:"gpu"`
}

func gbString(s string) int {
	v, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(s), "GB"))
	return v
}

func parseKoyeb(r io.Reader) ([]GPU, error) {
	var kr koyebResp
	if err := json.NewDecoder(r).Decode(&kr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0)
	for _, in := range kr.Instances {
		price := atof(in.PriceHourly)
		if in.GPU == nil || in.GPU.Count <= 0 || in.Status != "AVAILABLE" || len(in.Regions) == 0 || price <= 0 {
			continue
		}
		num := in.GPU.Count
		vramGB := gbString(in.GPU.Memory) / num
		perGPUFlops, memBWGBs, name := gpuSpecs(proBlackwellName(fmt.Sprintf("%s %dgb", in.GPU.Name, vramGB)))
		if name == "unknown" {
			continue
		}
		totalFlops := float64(num) * perGPUFlops / 1e12

		newGpu := GPU{
			_Id:      in.ID,
			Location: strings.Join(in.Regions, ", "),

			Name:              name,
			Vram:              vramGB * 1024,
			TotalFlops:        totalFlops,
			GpuMemoryBandwith: memBWGBs,
			NumGPUs:           num,

			CpuCores: in.VCPU,
			Ram:      gbString(in.Memory) * 1024,

			DiskSpace: float64(gbString(in.Disk)),

			TotalCostPH:      price,
			GpuCostPH:        price / float64(num),
			FlopsPerDollarPH: totalFlops / price,
			Source:           "koyeb",
			Url:              "https://www.koyeb.com/pricing",
		}
		newGpu.Score = calculateScore(newGpu)
		newGpu.ScoreDPH = newGpu.Score / newGpu.TotalCostPH
		out = append(out, newGpu)
	}
	return out, nil
}

func koyebGetter() ([]GPU, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get("https://app.koyeb.com/v1/catalog/instances?limit=100")
	if err != nil {
		return nil, fmt.Errorf("fetch koyeb instances: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("koyeb API status %s", resp.Status)
	}

	out, err := parseKoyeb(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d Koyeb GPUs\n", len(out))
	return out, nil
}
