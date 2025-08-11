package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Response struct {
	Offers []offer `json:"offers"`
}

type offer struct {
	ID        int64   `json:"id"`
	GPUName   string  `json:"gpu_name"`
	NumGPUs   int     `json:"num_gpus"`
	GPURAM    int     `json:"gpu_ram"`
	DPHTotal  float64 `json:"dph_total"`
	Verified  bool    `json:"verified"`
	Rentable  bool    `json:"rentable"`
	MachineID int64   `json:"machine_id"`
}

type search struct {
	Cost string `json:"gpuCostPerHour"`
}

type vastGpuJson struct {
	Id   string `json:"id" mapstructure:"id"`
	Name int    `json:"gpu_name" mapstructure:"name"`
	Ram  string `json:"diskHour" mapstructure:"ram"`
	//Cost search `json:"search" mapstructure:"float64"`
}

func vastGetter() ([]GPU, error) {
	response, err := http.Get("https://console.vast.ai/api/v0/search/asks/")
	if err != nil {

		return nil, fmt.Errorf("Error fetching vast: %s\n", err)
	}
	defer response.Body.Close()

	var sr Response
	if err := json.NewDecoder(response.Body).Decode(&sr); err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil, fmt.Errorf("decode: %w", err)
	}

	// Project to your lean slice.
	out := make([]GPU, 0, len(sr.Offers))
	for _, o := range sr.Offers {
		out = append(out, GPU{
			id:   o.ID,
			name: o.GPUName,
			ram:  o.GPURAM,
			cost: o.DPHTotal,
		})
	}
	fmt.Printf("Found %d vast gpus\n", len(out))
	for _, o := range out {
		fmt.Println(o.toString())
	}
	return out, nil
}
