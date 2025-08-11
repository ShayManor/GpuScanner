package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---- TensorDock API shapes (see docs) ----
// GET https://dashboard.tensordock.com/api/v2/hostnodes (Bearer token)
// Ref: docs pages "Fetch Hostnodes and Locations" + "Getting Started".
type tdHostnodesResp struct {
	Data struct {
		Hostnodes []tdHostnode `json:"hostnodes"`
	} `json:"data"`
}

type tdHostnode struct {
	ID                 string  `json:"id"`
	UptimePercentage   float64 `json:"uptime_percentage"`
	AvailableResources struct {
		GPUs []tdGPU `json:"gpus"`

		VCPUCount int     `json:"vcpu_count"`
		RAMGB     int     `json:"ram_gb"`
		StorageGB float64 `json:"storage_gb"`
		HasPubIP  bool    `json:"has_public_ip_available"`
	} `json:"available_resources"`

	Pricing struct {
		PerVcpuHr      float64 `json:"per_vcpu_hr"`
		PerGBRamHr     float64 `json:"per_gb_ram_hr"`
		PerGBStorageHr float64 `json:"per_gb_storage_hr"`
	} `json:"pricing"`

	Location struct {
		City                   string  `json:"city"`
		StateProvince          string  `json:"stateprovince"`
		Country                string  `json:"country"`
		NetworkSpeedGbps       float64 `json:"network_speed_gbps"`
		NetworkSpeedUploadGbps float64 `json:"network_speed_upload_gbps"`
		Tier                   int     `json:"tier"`
	} `json:"location"`
}

type tdGPU struct {
	V0Name         string  `json:"v0Name"` // e.g., "h100-sxm5-80gb", "geforcertx4090-pcie-24gb"
	AvailableCount int     `json:"availableCount"`
	PricePerHr     float64 `json:"price_per_hr"`
}

// ---- Helper: parse VRAM from the v0Name (returns MB; 0 if unknown) ----
var vramRe = regexp.MustCompile(`(?i)(\d+)\s*gb`)

func parseVRAMMB(name string) int {
	m := vramRe.FindStringSubmatch(name)
	if len(m) >= 2 {
		gb, err := strconv.Atoi(m[1])
		if err == nil {
			return gb * 1024
		}
	}
	return 0
}

func hashToInt64(s string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	u := h.Sum64()
	return int64(u & 0x7fffffffffffffff) // keep it positive
}

func lookupGPUHardware(name string) (totalFlops float64, lanes int, memBWGBs float64, ok bool) {
	type spec struct {
		flops float64 // FP32 FLOP/s
		lanes int
		bwGBs float64
	}
	// Canonical table (sources: NVIDIA datasheets/architecture whitepapers).
	table := map[string]spec{
		// GeForce / Ada & Ampere
		"rtx4090": {flops: 82.6e12, lanes: 16, bwGBs: 1008}, // Ada, 24GB GDDR6X
		"rtx3090": {flops: 35.6e12, lanes: 16, bwGBs: 936},  // Ampere, 24GB GDDR6X

		// Data center Ampere
		"a100-40gb-pcie": {flops: 19.5e12, lanes: 16, bwGBs: 1555},
		"a100-80gb-pcie": {flops: 19.5e12, lanes: 16, bwGBs: 1935},
		"a100-40gb-sxm":  {flops: 19.5e12, lanes: 0, bwGBs: 1555},
		"a100-80gb-sxm":  {flops: 19.5e12, lanes: 0, bwGBs: 2039},

		// Hopper
		"h100-80gb-sxm":  {flops: 67e12, lanes: 0, bwGBs: 3350},
		"h100-80gb-pcie": {flops: 51e12, lanes: 16, bwGBs: 2000},

		// Ada data center
		"l40s": {flops: 91.6e12, lanes: 16, bwGBs: 864},
		"l40":  {flops: 90.5e12, lanes: 16, bwGBs: 864},
		"l4":   {flops: 30.3e12, lanes: 16, bwGBs: 300},

		// Ampere data center midrange
		"a10": {flops: 31.2e12, lanes: 16, bwGBs: 600},
		"t4":  {flops: 8.1e12, lanes: 16, bwGBs: 300},
	}

	// Normalize input (lowercase, remove separators) and infer canonical key.
	n := strings.ToLower(name)
	n = strings.ReplaceAll(n, "_", "-")
	n = strings.ReplaceAll(n, " ", "-")

	// Helper: contains any of the substrings
	contains := func(s string, subs ...string) bool {
		for _, sub := range subs {
			if strings.Contains(s, sub) {
				return true
			}
		}
		return false
	}

	// Try direct simple model keys first.
	switch {
	case contains(n, "4090"):
		return table["rtx4090"].flops, table["rtx4090"].lanes, table["rtx4090"].bwGBs, true
	case contains(n, "3090"):
		return table["rtx3090"].flops, table["rtx3090"].lanes, table["rtx3090"].bwGBs, true
	}

	// A100 variants
	if contains(n, "a100") {
		isSXM := contains(n, "sxm")
		gb80 := contains(n, "80gb", "80g", "-80")
		gb40 := contains(n, "40gb", "40g", "-40")
		if isSXM {
			if gb80 {
				return table["a100-80gb-sxm"].flops, table["a100-80gb-sxm"].lanes, table["a100-80gb-sxm"].bwGBs, true
			}
			if gb40 {
				return table["a100-40gb-sxm"].flops, table["a100-40gb-sxm"].lanes, table["a100-40gb-sxm"].bwGBs, true
			}
			// default unknown-capacity SXM -> 80GB as a reasonable default
			return table["a100-80gb-sxm"].flops, table["a100-80gb-sxm"].lanes, table["a100-80gb-sxm"].bwGBs, true
		}
		// Assume PCIe if not SXM
		if gb80 {
			return table["a100-80gb-pcie"].flops, table["a100-80gb-pcie"].lanes, table["a100-80gb-pcie"].bwGBs, true
		}
		if gb40 {
			return table["a100-40gb-pcie"].flops, table["a100-40gb-pcie"].lanes, table["a100-40gb-pcie"].bwGBs, true
		}
		return table["a100-80gb-pcie"].flops, table["a100-80gb-pcie"].lanes, table["a100-80gb-pcie"].bwGBs, true
	}

	// H100 variants
	if contains(n, "h100") {
		isSXM := contains(n, "sxm")
		if isSXM || contains(n, "nvl") {
			return table["h100-80gb-sxm"].flops, table["h100-80gb-sxm"].lanes, table["h100-80gb-sxm"].bwGBs, true
		}
		return table["h100-80gb-pcie"].flops, table["h100-80gb-pcie"].lanes, table["h100-80gb-pcie"].bwGBs, true
	}

	// Ada DC: L40S/L40/L4
	switch {
	case contains(n, "l40s"):
		return table["l40s"].flops, table["l40s"].lanes, table["l40s"].bwGBs, true
	case contains(n, "l40"):
		return table["l40"].flops, table["l40"].lanes, table["l40"].bwGBs, true
	case contains(n, "l4"):
		return table["l4"].flops, table["l4"].lanes, table["l4"].bwGBs, true
	}

	// Ampere DC midrange
	switch {
	case contains(n, "a10"):
		return table["a10"].flops, table["a10"].lanes, table["a10"].bwGBs, true
	case contains(n, "t4"):
		return table["t4"].flops, table["t4"].lanes, table["t4"].bwGBs, true
	}

	// Not found
	return 0, 0, 0, false
}

// Env var required: TENSORDOCK_TOKEN
func tensordockGetter() ([]GPU, error) {
	token := os.Getenv("TENSORDOCK_TOKEN")
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("TENSORDOCK_TOKEN is not set")
	}

	req, err := http.NewRequest("GET", "https://dashboard.tensordock.com/api/v2/hostnodes", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch tensordock hostnodes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("tensordock API status %s", resp.Status)
	}

	var hr tdHostnodesResp
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0, 256)
	for _, hn := range hr.Data.Hostnodes {
		loc := strings.TrimSpace(fmt.Sprintf("%s, %s", hn.Location.City, hn.Location.Country))
		// Bandwidth in docs is Gbps; your struct prints Mbps. Convert.
		downMbps := hn.Location.NetworkSpeedGbps * 1000
		upMbps := hn.Location.NetworkSpeedUploadGbps * 1000

		for _, g := range hn.AvailableResources.GPUs {
			if g.AvailableCount <= 0 {
				continue
			}
			totalFlops, _, memBWGBs, _ := lookupGPUHardware(g.V0Name)
			out = append(out, GPU{
				id:          hn.ID,
				location:    loc,
				reliability: hn.UptimePercentage / 100.0, // docs give percent
				duration:    0,                           // not exposed

				name:              g.V0Name,
				vram:              parseVRAMMB(g.V0Name),
				totalFlops:        totalFlops, // not exposed
				gpuMemoryBandwith: memBWGBs,   // not exposed
				numGPUs:           g.AvailableCount,

				cpuCores: float64(hn.AvailableResources.VCPUCount),
				cpuName:  "",
				cpuGhz:   0,
				cpuArch:  "",

				ram: hn.AvailableResources.RAMGB, // GB

				diskSpace: hn.AvailableResources.StorageGB, // GB
				diskBW:    0,
				diskName:  "",

				uploadSpeed:   upMbps,
				downloadSpeed: downMbps,

				// Prices: TensorDock exposes GPU price/hr, plus unit prices for CPU/RAM/Storage.
				// To keep semantics consistent with Vast, we set totalCostPH to GPU price here.
				totalCostPH:      g.PricePerHr,
				gpuCostPH:        g.PricePerHr,
				diskCostPH:       g.PricePerHr, // per-GB rate exists, but we avoid mixing units here
				uploadCostPH:     0,
				downloadCostPH:   0,
				flopsPerDollarPH: totalFlops / g.PricePerHr,
				Source:           "tensordock",
			})
		}
	}

	fmt.Printf("Found %d TensorDock GPUs\n", len(out))
	for _, o := range out {
		fmt.Println(o.toString())
	}
	return out, nil
}
