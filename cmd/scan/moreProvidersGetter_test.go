package main

import (
	"strings"
	"testing"
)

// A 2x H100 instance and a CPU-only instance.
const verdaFixture = `[
 {"instance_type":"2H100.80S.80V","name":"H100 SXM5 80GB","price_per_hour":"9.974",
  "cpu":{"number_of_cores":80},"gpu":{"number_of_gpus":2},"gpu_memory":{"size_in_gigabytes":160},"memory":{"size_in_gigabytes":340}},
 {"instance_type":"CPU.4V.16G","name":"AMD Rome+","price_per_hour":"0.048",
  "cpu":{"number_of_cores":4},"gpu":{"number_of_gpus":0},"gpu_memory":{"size_in_gigabytes":0},"memory":{"size_in_gigabytes":16}}
]`

func TestParseVerda(t *testing.T) {
	gpus, err := parseVerda(strings.NewReader(verdaFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 1 {
		t.Fatalf("got %d rows, want 1 (CPU-only dropped)", len(gpus))
	}
	g := gpus[0]
	if g.Name != "H100 SXM" || g.NumGPUs != 2 || g.Vram != 80*1024 || g.Ram != 340*1024 {
		t.Errorf("got %s x%d %dMB ram %dMB", g.Name, g.NumGPUs, g.Vram, g.Ram)
	}
	if !approx(g.TotalCostPH, 9.974) || !approx(g.GpuCostPH, 4.987) || g.Source != "verda" {
		t.Errorf("prices/source = %v/%v/%q", g.TotalCostPH, g.GpuCostPH, g.Source)
	}
}

// A 2x RTX 4000 Ada plan and a standard (non-GPU) plan.
const linodeFixture = `{"data":[
 {"id":"g2-gpu-rtx4000a2-m","label":"RTX4000 Ada x2 Medium","class":"gpu","gpus":2,"vcpus":16,"memory":65536,"disk":1048576,"price":{"hourly":1.34}},
 {"id":"g6-nanode-1","label":"Nanode 1GB","class":"nanode","gpus":0,"vcpus":1,"memory":1024,"disk":25600,"price":{"hourly":0.0075}}
]}`

func TestParseLinode(t *testing.T) {
	gpus, err := parseLinode(strings.NewReader(linodeFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 1 {
		t.Fatalf("got %d rows, want 1", len(gpus))
	}
	g := gpus[0]
	if g.Name != "RTX 4000 ada" || g.NumGPUs != 2 || g.Ram != 65536 || g.DiskSpace != 1024 {
		t.Errorf("got %s x%d ram %dMB disk %vGB", g.Name, g.NumGPUs, g.Ram, g.DiskSpace)
	}
	if g.Vram != 20*1024 || g.Reliability != 0.9999 {
		t.Errorf("vram/reliability = %dMB/%v, want 20GB from the RTX 4000 Ada and Akamai's 99.99%% SLA", g.Vram, g.Reliability)
	}
	if !approx(g.GpuCostPH, 0.67) || g.Source != "linode" {
		t.Errorf("gpu price/source = %v/%q", g.GpuCostPH, g.Source)
	}
}

// A deployable 2x H100, a GPU instance with no regions, and a CPU instance.
const koyebFixture = `{"instances":[
 {"id":"2-gpu-nvidia-h100","type":"gpu","status":"AVAILABLE","vcpu":30,"memory":"360GB","disk":"640GB","price_hourly":"5.0",
  "regions":["na"],"gpu":{"count":2,"brand":"nvidia","memory":"160GB","name":"gpu-nvidia-h100"}},
 {"id":"gpu-nvidia-b200","type":"gpu","status":"AVAILABLE","vcpu":15,"memory":"240GB","disk":"640GB","price_hourly":"5.50",
  "regions":[],"gpu":{"count":1,"brand":"nvidia","memory":"180GB","name":"gpu-nvidia-b200"}},
 {"id":"nano","type":"standard","status":"AVAILABLE","vcpu":0.25,"memory":"256MB","disk":"2GB","price_hourly":"0.002","regions":["fra"],"gpu":null}
]}`

func TestParseKoyeb(t *testing.T) {
	gpus, err := parseKoyeb(strings.NewReader(koyebFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 1 {
		t.Fatalf("got %d rows, want 1", len(gpus))
	}
	g := gpus[0]
	if g.NumGPUs != 2 || g.Vram != 80*1024 || g.Ram != 360*1024 || g.Location != "na" {
		t.Errorf("got x%d %dMB ram %dMB %q", g.NumGPUs, g.Vram, g.Ram, g.Location)
	}
	if !approx(g.TotalCostPH, 5) || !approx(g.GpuCostPH, 2.5) || g.Source != "koyeb" {
		t.Errorf("prices/source = %v/%v/%q", g.TotalCostPH, g.GpuCostPH, g.Source)
	}
}

// An L40S type with free GPUs, a sold-out H100 type and a CPU-only type.
const cudoFixture = `{"machineTypes":[
 {"dataCenterId":"no-kristiansand-1","machineType":"epyc-genoa-l40s","gpuModel":"L40S","minVcpuPerGpu":2,"maxVcpuPerMemoryGib":0.5,
  "vcpuPriceHr":{"value":"0.0049"},"memoryGibPriceHr":{"value":"0.0049"},"gpuPriceHr":{"value":"0.87"},"maxGpuFree":2},
 {"dataCenterId":"no-kristiansand-1","machineType":"sapphire-rapids-h100","gpuModel":"H100 SXM","minVcpuPerGpu":1,"maxVcpuPerMemoryGib":0.2,
  "vcpuPriceHr":{"value":"0.0059"},"memoryGibPriceHr":{"value":"0.0049"},"gpuPriceHr":{"value":"1.79"},"maxGpuFree":0},
 {"dataCenterId":"se-skarholmen-1","machineType":"epyc-genoa","gpuModel":"","minVcpuPerGpu":0,"maxVcpuPerMemoryGib":1,
  "vcpuPriceHr":{"value":"0.0049"},"memoryGibPriceHr":{"value":"0.0098"},"gpuPriceHr":{"value":"0"},"maxGpuFree":0}
]}`

func TestParseCudo(t *testing.T) {
	gpus, err := parseCudo(strings.NewReader(cudoFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 1 {
		t.Fatalf("got %d rows, want 1", len(gpus))
	}
	g := gpus[0]
	// 2 vCPUs at most 0.5 vCPU per GiB need 4 GiB: 0.87 + 2*0.0049 + 4*0.0049.
	if g.Name != "l40s" || g.CpuCores != 2 || g.Ram != 4*1024 || g.Vram != 48*1024 {
		t.Errorf("got %s cpu %v ram %dMB", g.Name, g.CpuCores, g.Ram)
	}
	if !approx(g.TotalCostPH, 0.87+6*0.0049) || !approx(g.GpuCostPH, 0.87) || g.Location != "no-kristiansand-1" {
		t.Errorf("prices/location = %v/%v/%q", g.TotalCostPH, g.GpuCostPH, g.Location)
	}
}

func TestCheapestAzurePricesSkipsSpotWindowsAndEdgeZones(t *testing.T) {
	items := []azurePriceItem{
		{UnitPrice: 0.343, ArmRegionName: "attdallas1", Location: "ATT Dallas 1", SkuName: "NC4as T4 v3", ProductName: "Virtual Machines NCasT4 v3 Series", ArmSkuName: "Standard_NC4as_T4_v3"},
		{UnitPrice: 0.1, ArmRegionName: "eastus", Location: "US East", SkuName: "NC4as T4 v3 Spot", ProductName: "Virtual Machines NCasT4 v3 Series", ArmSkuName: "Standard_NC4as_T4_v3"},
		{UnitPrice: 0.2, ArmRegionName: "eastus", Location: "US East", SkuName: "NC4as T4 v3", ProductName: "Virtual Machines NCasT4 v3 Series Windows", ArmSkuName: "Standard_NC4as_T4_v3"},
		{UnitPrice: 0.558, ArmRegionName: "swedencentral", Location: "SE Central", SkuName: "NC4as T4 v3", ProductName: "Virtual Machines NCasT4 v3 Series", ArmSkuName: "Standard_NC4as_T4_v3"},
		{UnitPrice: 0.526, ArmRegionName: "eastus2", Location: "US East 2", SkuName: "NC4as T4 v3", ProductName: "Virtual Machines NCasT4 v3 Series", ArmSkuName: "Standard_NC4as_T4_v3"},
	}
	rows := azureRows(cheapestAzurePrices(items))
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	g := rows[0]
	if !approx(g.TotalCostPH, 0.526) || g.Location != "US East 2" {
		t.Errorf("picked $%v in %q, want $0.526 in US East 2", g.TotalCostPH, g.Location)
	}
	if g.Reliability != 0.999 {
		t.Errorf("reliability = %v, want Azure's 99.9%% single-VM SLA", g.Reliability)
	}
	if g.Name != "t4" || g.NumGPUs != 1 || g.Vram != 16*1024 || g.Source != "azure" {
		t.Errorf("got %s x%d %dMB %q", g.Name, g.NumGPUs, g.Vram, g.Source)
	}
}

func TestAzureSizesAllMapToKnownGPUs(t *testing.T) {
	for sku, size := range azureGPUSizes {
		if _, _, name := gpuSpecs(size.GPU); name == "unknown" {
			t.Errorf("%s: gpuSpecs(%q) is unknown", sku, size.GPU)
		}
	}
}

func TestNosanaVramFromMarketName(t *testing.T) {
	const fx = `[{"address":"a","slug":"nvidia-a100-80gb-community","name":"NVIDIA A100 80GB Community","type":"COMMUNITY",
  "usd_reward_per_hour":0.61364,"network_fee_percentage":10,"client":false,"metadata":[]}]`
	gpus, err := parseNosana(strings.NewReader(fx))
	if err != nil || len(gpus) != 1 {
		t.Fatalf("rows=%d err=%v", len(gpus), err)
	}
	if gpus[0].Vram != 80*1024 {
		t.Errorf("Vram = %dMB, want 80GB from the market name", gpus[0].Vram)
	}
}

func TestParseLiumNodesKeepsFeedNameForUnknownGPU(t *testing.T) {
	const fx = `{"nodes":[{"id":"n1","gpu_model":"RTX 5070 Ti","gpu_count":1,"available_gpu_count":1,
  "gpu_memory_gb":16,"price_per_gpu_hour":0.2,"price_per_node_hour":0.2}]}`
	gpus, err := parseLiumNodes(strings.NewReader(fx))
	if err != nil || len(gpus) != 1 {
		t.Fatalf("rows=%d err=%v", len(gpus), err)
	}
	if gpus[0].Name != "RTX 5070 Ti" {
		t.Errorf("Name = %q, want the feed's model name", gpus[0].Name)
	}
}

func TestStandardVramKeysAreGpuSpecsNames(t *testing.T) {
	known := map[string]bool{}
	for _, probe := range []string{"RTX 5090", "RTX 5080", "RTX 4090", "RTX 4080", "RTX 4070", "RTX 3090", "RTX 3080 Ti", "RTX 3080",
		"RTX 3070", "H100 SXM", "H100 PCIe", "H200", "L40S", "L40", "L4", "A10", "A30", "A40", "T4", "RTX A6000", "RTX A5000",
		"RTX A4500", "RTX A4000", "RTX PRO 6000 Blackwell", "RTX 6000 Ada", "RTX 4000 Ada", "RTX 2000 Ada", "MI300X"} {
		_, _, name := gpuSpecs(probe)
		known[name] = true
	}
	for name := range standardVramGB {
		if !known[name] {
			t.Errorf("standardVramGB key %q is not a name gpuSpecs returns", name)
		}
	}
}
