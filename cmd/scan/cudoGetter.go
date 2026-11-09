package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// ---- Cudo Compute public machine types (no key) ----
// GET https://rest.compute.cudo.org/v1/vms/machine-types
// One entry per data center and machine type. VMs are priced per part, so a
// row is the smallest 1-GPU VM the type allows: the GPU plus its minimum vCPUs
// and the memory those vCPUs require.
type cudoResp struct {
	MachineTypes []cudoMachineType `json:"machineTypes"`
}

type cudoPrice struct {
	Value string `json:"value"`
}

type cudoMachineType struct {
	DataCenterID        string    `json:"dataCenterId"`
	MachineType         string    `json:"machineType"`
	GPUModel            string    `json:"gpuModel"`
	MinVcpuPerGpu       float64   `json:"minVcpuPerGpu"`
	MaxVcpuPerMemoryGib float64   `json:"maxVcpuPerMemoryGib"`
	VcpuPriceHr         cudoPrice `json:"vcpuPriceHr"`
	MemoryGibPriceHr    cudoPrice `json:"memoryGibPriceHr"`
	GPUPriceHr          cudoPrice `json:"gpuPriceHr"`
	MaxGpuFree          int       `json:"maxGpuFree"`
}

func parseCudo(r io.Reader) ([]GPU, error) {
	var cr cudoResp
	if err := json.NewDecoder(r).Decode(&cr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0, len(cr.MachineTypes))
	for _, m := range cr.MachineTypes {
		gpuPrice := atof(m.GPUPriceHr.Value)
		if m.GPUModel == "" || m.MaxGpuFree <= 0 || gpuPrice <= 0 {
			continue
		}
		flops, memBWGBs, name := gpuSpecs(m.GPUModel)
		if name == "unknown" {
			continue
		}
		vcpus := math.Max(m.MinVcpuPerGpu, 1)
		memGiB := 1.0
		if m.MaxVcpuPerMemoryGib > 0 {
			memGiB = math.Ceil(vcpus / m.MaxVcpuPerMemoryGib)
		}
		price := gpuPrice + vcpus*atof(m.VcpuPriceHr.Value) + memGiB*atof(m.MemoryGibPriceHr.Value)
		totalFlops := flops / 1e12

		newGpu := GPU{
			_Id:      m.DataCenterID + "/" + m.MachineType,
			Location: m.DataCenterID,

			Name:              name,
			TotalFlops:        totalFlops,
			GpuMemoryBandwith: memBWGBs,
			NumGPUs:           1,

			CpuCores: vcpus,
			Ram:      int(memGiB) * 1024,

			TotalCostPH:      price,
			GpuCostPH:        gpuPrice,
			FlopsPerDollarPH: totalFlops / price,
			Source:           "cudo",
			Url:              "https://www.cudocompute.com/",
		}
		newGpu.Score = calculateScore(newGpu)
		newGpu.ScoreDPH = newGpu.Score / newGpu.TotalCostPH
		out = append(out, newGpu)
	}
	return out, nil
}

func cudoGetter() ([]GPU, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get("https://rest.compute.cudo.org/v1/vms/machine-types")
	if err != nil {
		return nil, fmt.Errorf("fetch cudo machine types: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("cudo API status %s", resp.Status)
	}

	out, err := parseCudo(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d Cudo GPUs\n", len(out))
	return out, nil
}
