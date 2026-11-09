package main

import (
	"math"
	"strings"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// An 8-GPU RTX PRO 6000 in stock, a sold-out B200 and an unknown accelerator.
const runcrateFixture = `{"gpus":[
 {"gpu_type":"RTXPro6000","name":"RTXPro6000","vram":"96GB","min_price_hr":2.41,"available_options":14,
  "regions":["Kansas City, MO, United States","Beltsville, MD, United States"],
  "specs":{"vcpus":256,"memory_gb":1003,"storage_gb":2000,"num_gpus":8}},
 {"gpu_type":"B200","name":"B200","vram":"180GB","min_price_hr":8.64,"available_options":0,
  "regions":["Ashburn, VA, United States"],"specs":{"vcpus":208,"memory_gb":2854,"storage_gb":2000,"num_gpus":8}},
 {"gpu_type":"GAUDI2","name":"GAUDI2","vram":"—","min_price_hr":1.56,"available_options":3,
  "regions":["Houston, TX, United States"],"specs":{"vcpus":152,"memory_gb":940,"storage_gb":27000,"num_gpus":8}}
],"fetched_at":"2026-11-09T16:34:14.576Z"}`

func TestParseRuncrate(t *testing.T) {
	gpus, err := parseRuncrate(strings.NewReader(runcrateFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 1 {
		t.Fatalf("got %d rows, want 1 (sold-out and unknown GPUs dropped)", len(gpus))
	}
	g := gpus[0]
	if g.Name != "RTX PRO 6000 Blackwell" || g.NumGPUs != 8 || g.Vram != 96*1024 {
		t.Errorf("got %s x%d %dMB", g.Name, g.NumGPUs, g.Vram)
	}
	if !approx(g.GpuCostPH, 2.41) || !approx(g.TotalCostPH, 2.41*8) {
		t.Errorf("prices = %v/%v, want per-GPU 2.41 and node 19.28", g.GpuCostPH, g.TotalCostPH)
	}
	if g.Location != "Kansas City, MO, United States" || g.CpuCores != 256 || g.Ram != 1003*1024 {
		t.Errorf("location/cpu/ram = %q/%v/%v", g.Location, g.CpuCores, g.Ram)
	}
	if g.Source != "runcrate" || !strings.HasPrefix(g.Url, "https://www.runcrate.ai/pricing") {
		t.Errorf("source/url = %q/%q", g.Source, g.Url)
	}
}

// An available H100, a model with no price, and a model with none available.
const akashFixture = `{"availability":{"total":40,"available":9},"models":[
 {"vendor":"nvidia","model":"h100","ram":"80Gi","interface":"SXM5","availability":{"total":32,"available":9},
  "price":{"currency":"USD","min":2.56,"max":3.1,"avg":2.8,"weightedAverage":2.9,"med":2.86}},
 {"vendor":"nvidia","model":"pro6000we","ram":"96Gi","interface":"PCIe","availability":{"total":1,"available":1},"price":null},
 {"vendor":"nvidia","model":"rtx4090","ram":"24Gi","interface":"PCIe","availability":{"total":3,"available":0},
  "price":{"currency":"USD","min":0.36,"max":0.36,"avg":0.36,"weightedAverage":0.36,"med":0.36}}
]}`

func TestParseAkash(t *testing.T) {
	gpus, err := parseAkash(strings.NewReader(akashFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 1 {
		t.Fatalf("got %d rows, want 1", len(gpus))
	}
	g := gpus[0]
	if g.Name != "H100 SXM" || g.Vram != 80*1024 || !approx(g.TotalCostPH, 2.56) || g.Source != "akash" {
		t.Errorf("got %s %dMB $%v %s", g.Name, g.Vram, g.TotalCostPH, g.Source)
	}
}

// A six-GPU market, a single-GPU A4000 market without metadata, a private
// client market and a market with no price.
const nosanaFixture = `[
 {"address":"addr6x","slug":"nvidia-pro6000-6x","name":"NVIDIA Pro 6000 (6x)","type":"PREMIUM",
  "usd_reward_per_hour":12,"network_fee_percentage":10,"client":false,"metadata":[{"key":"vram","value":"576GB"}]},
 {"address":"addrA4","slug":"nvidia-a4000-community","name":"NVIDIA 4000/A4000 Community","type":"COMMUNITY",
  "usd_reward_per_hour":0.0873,"network_fee_percentage":10,"client":false,"metadata":[]},
 {"address":"addrPriv","slug":"zephyr-skyline-4090","name":"Zephyr Skyline 4090","type":"PREMIUM",
  "usd_reward_per_hour":0.35,"network_fee_percentage":10,"client":true,"metadata":[]},
 {"address":"addr8x","slug":"nvidia-8x-a5000","name":"Enterprise 8xA5000","type":"OTHER",
  "usd_reward_per_hour":null,"network_fee_percentage":10,"client":false,"metadata":[]}
]`

func TestParseNosana(t *testing.T) {
	gpus, err := parseNosana(strings.NewReader(nosanaFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 2 {
		t.Fatalf("got %d rows, want 2", len(gpus))
	}
	six, a4 := gpus[0], gpus[1]
	if six.Name != "RTX PRO 6000 Blackwell" || six.NumGPUs != 6 || six.Vram != 96*1024 {
		t.Errorf("6x market = %s x%d %dMB", six.Name, six.NumGPUs, six.Vram)
	}
	if !approx(six.TotalCostPH, 13.2) || !approx(six.GpuCostPH, 2.2) {
		t.Errorf("6x prices = %v/%v, want 13.2 total incl. 10%% fee and 2.2 per GPU", six.TotalCostPH, six.GpuCostPH)
	}
	if a4.Name != "RTX A4000" || a4.NumGPUs != 1 || a4.Vram != 0 {
		t.Errorf("A4000 market = %s x%d %dMB", a4.Name, a4.NumGPUs, a4.Vram)
	}
	if six.Url != "https://dashboard.nosana.com/markets/addr6x" || six.Source != "nosana" {
		t.Errorf("url/source = %q/%q", six.Url, six.Source)
	}
}

// A 4x H200 VM, a CPU-only VM and an AWS Neuron VM.
const thetaFixture = `{"status":"success","body":{"vms":[
 {"id":"vm_gh200x4","name":"G-H200_141GB-x4","resources":{"cpu":"32000","gpu":"4","mem":"577536","storage":"850","gpu_model":"nvidia_H200_141G"},"price_hour":"1476"},
 {"id":"vm_c1","name":"C1","resources":{"cpu":"2000","mem":"2048","storage":"32"},"price_hour":"5"},
 {"id":"vm_ginf2_xl_x1","name":"G-INF2_xl-x1","resources":{"cpu":"3000","gpu":"1","mem":"14336","storage":"256","gpu_model":"neuron_inf2"},"price_hour":"76"}
]}}`

func TestParseTheta(t *testing.T) {
	gpus, err := parseTheta(strings.NewReader(thetaFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 1 {
		t.Fatalf("got %d rows, want 1", len(gpus))
	}
	g := gpus[0]
	if g.Name != "H200" || g.NumGPUs != 4 || g.Vram != 141*1024 {
		t.Errorf("got %s x%d %dMB", g.Name, g.NumGPUs, g.Vram)
	}
	if !approx(g.TotalCostPH, 14.76) || !approx(g.GpuCostPH, 3.69) {
		t.Errorf("prices = %v/%v, want 14.76 VM and 3.69 per GPU (feed is in cents)", g.TotalCostPH, g.GpuCostPH)
	}
	if g.CpuCores != 32 || g.Ram != 577536 || g.DiskSpace != 850 || g.Source != "theta" {
		t.Errorf("cpu/ram/disk/source = %v/%v/%v/%q", g.CpuCores, g.Ram, g.DiskSpace, g.Source)
	}
}

func TestGpuSpecsA4000IsNotA40(t *testing.T) {
	if _, _, name := gpuSpecs("RTX A4000"); name != "RTX A4000" {
		t.Errorf("gpuSpecs(RTX A4000) = %q", name)
	}
	if _, _, name := gpuSpecs("A40"); name != "a40" {
		t.Errorf("gpuSpecs(A40) = %q", name)
	}
}
