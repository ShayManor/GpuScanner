package main

import (
	"math"
	"strings"
	"testing"
)

// One node per case: a multi-GPU B300, a single RTX 5090 with an unreported disk
// type, a sold-out node and a zero-priced node.
const liumFixture = `{
  "generated_at": "2026-09-21T15:01:09.817165Z",
  "count": 4,
  "nodes": [
    {
      "id": "b300-node",
      "gpu_model": "B300 SXM6 AC",
      "gpu_count": 8,
      "available_gpu_count": 8,
      "price_per_gpu_hour": 4.95,
      "min_rentable_gpu_count": 8,
      "gpu_memory_gb": 279,
      "cpu_count": 224,
      "ram_gb": 2949,
      "disk_gb": 28000,
      "cpu_model": "INTEL(R) XEON(R) PLATINUM 8570",
      "network_upload_mbps": 1200.5,
      "network_download_mbps": 980.25,
      "uptime_minutes": 41355,
      "reliability_score": 99.9,
      "disk_type": "nvme",
      "tier": "secure",
      "country": "United States",
      "country_code": "US",
      "region": "NJ",
      "city": "Secaucus",
      "rent_url": "https://lium.io/browse-pods/b300-node",
      "price_per_node_hour": 39.6
    },
    {
      "id": "rtx5090-node",
      "gpu_model": "RTX 5090",
      "gpu_count": 1,
      "available_gpu_count": 1,
      "price_per_gpu_hour": 0.49,
      "min_rentable_gpu_count": 1,
      "gpu_memory_gb": 32,
      "cpu_count": 16,
      "ram_gb": 64,
      "disk_gb": 500,
      "cpu_model": "AMD Ryzen 9 7950X",
      "network_upload_mbps": 584.1,
      "network_download_mbps": 461.7,
      "uptime_minutes": 30,
      "reliability_score": 95.4,
      "disk_type": "unknown",
      "tier": "secure",
      "country": "Germany",
      "country_code": "DE",
      "region": "HE",
      "city": "Frankfurt",
      "rent_url": "https://lium.io/browse-pods/rtx5090-node",
      "price_per_node_hour": 0.49
    },
    {
      "id": "sold-out-node",
      "gpu_model": "H200",
      "gpu_count": 8,
      "available_gpu_count": 0,
      "price_per_gpu_hour": 3.0,
      "gpu_memory_gb": 140,
      "cpu_count": 24,
      "ram_gb": 236,
      "disk_gb": 500,
      "cpu_model": "INTEL(R) XEON(R) PLATINUM 8592+",
      "network_upload_mbps": 584.1,
      "network_download_mbps": 461.7,
      "reliability_score": 99.9,
      "disk_type": "hdd",
      "tier": "secure",
      "country": "United States",
      "city": "Secaucus",
      "rent_url": "https://lium.io/browse-pods/sold-out-node",
      "price_per_node_hour": 24.0
    },
    {
      "id": "free-node",
      "gpu_model": "RTX 3090",
      "gpu_count": 1,
      "available_gpu_count": 1,
      "price_per_gpu_hour": 0,
      "gpu_memory_gb": 24,
      "cpu_count": 8,
      "ram_gb": 32,
      "disk_gb": 200,
      "cpu_model": "AMD Ryzen 7 3700X",
      "network_upload_mbps": 300,
      "network_download_mbps": 300,
      "reliability_score": 98.0,
      "disk_type": "ssd",
      "tier": "secure",
      "country": "France",
      "city": "Paris",
      "rent_url": "https://lium.io/browse-pods/free-node",
      "price_per_node_hour": 0
    }
  ]
}`

func parseFixture(t *testing.T) map[string]GPU {
	t.Helper()
	gpus, err := parseLiumNodes(strings.NewReader(liumFixture))
	if err != nil {
		t.Fatalf("parseLiumNodes: %v", err)
	}
	byID := make(map[string]GPU, len(gpus))
	for _, g := range gpus {
		byID[g._Id] = g
	}
	return byID
}

func TestParseLiumNodesSkipsUnavailableAndUnpricedNodes(t *testing.T) {
	byID := parseFixture(t)

	if len(byID) != 2 {
		t.Fatalf("got %d nodes, want 2", len(byID))
	}
	if _, ok := byID["sold-out-node"]; ok {
		t.Error("node with available_gpu_count 0 should be skipped")
	}
	if _, ok := byID["free-node"]; ok {
		t.Error("node with price_per_node_hour 0 should be skipped")
	}
}

func TestParseLiumNodesMapsReliabilityScore(t *testing.T) {
	byID := parseFixture(t)

	if got := byID["b300-node"].Reliability; math.Abs(got-0.999) > 1e-9 {
		t.Errorf("Reliability = %v, want 0.999", got)
	}
	if got := byID["rtx5090-node"].Reliability; math.Abs(got-0.954) > 1e-9 {
		t.Errorf("Reliability = %v, want 0.954", got)
	}
}

func TestParseLiumNodesMapsCPUModelAndNetworkSpeeds(t *testing.T) {
	byID := parseFixture(t)
	g := byID["b300-node"]

	if g.CpuName != "INTEL(R) XEON(R) PLATINUM 8570" {
		t.Errorf("CpuName = %q, want the published cpu_model", g.CpuName)
	}
	if g.UploadSpeed != 1200.5 {
		t.Errorf("UploadSpeed = %v, want 1200.5", g.UploadSpeed)
	}
	if g.DownloadSpeed != 980.25 {
		t.Errorf("DownloadSpeed = %v, want 980.25", g.DownloadSpeed)
	}
}

func TestParseLiumNodesLeavesDiskNameBlankWhenTypeUnknown(t *testing.T) {
	byID := parseFixture(t)

	if got := byID["b300-node"].DiskName; got != "nvme" {
		t.Errorf("DiskName = %q, want %q", got, "nvme")
	}
	if got := byID["rtx5090-node"].DiskName; got != "" {
		t.Errorf("DiskName = %q, want empty for disk_type \"unknown\"", got)
	}
}

func TestParseLiumNodesKeepsGPUAndNodePricesSeparate(t *testing.T) {
	byID := parseFixture(t)
	g := byID["b300-node"]

	if g.GpuCostPH != 4.95 {
		t.Errorf("GpuCostPH = %v, want 4.95", g.GpuCostPH)
	}
	if g.TotalCostPH != 39.6 {
		t.Errorf("TotalCostPH = %v, want 39.6", g.TotalCostPH)
	}
}

func TestParseLiumNodesReadsVramPerGPUAndRamPerNode(t *testing.T) {
	byID := parseFixture(t)
	g := byID["b300-node"]

	if g.NumGPUs != 8 {
		t.Errorf("NumGPUs = %d, want 8", g.NumGPUs)
	}
	if g.Vram != 279*1024 {
		t.Errorf("Vram = %d MB, want per-GPU %d MB", g.Vram, 279*1024)
	}
	if g.Ram != 2949*1024 {
		t.Errorf("Ram = %d MB, want node total %d MB", g.Ram, 2949*1024)
	}
	if g.CpuCores != 224 {
		t.Errorf("CpuCores = %v, want 224", g.CpuCores)
	}
	if g.DiskSpace != 28000 {
		t.Errorf("DiskSpace = %v, want 28000", g.DiskSpace)
	}
}

func TestParseLiumNodesSetsSourceLocationAndRentURL(t *testing.T) {
	byID := parseFixture(t)
	g := byID["rtx5090-node"]

	if g.Source != "lium" {
		t.Errorf("Source = %q, want \"lium\"", g.Source)
	}
	if g.Location != "Frankfurt, Germany" {
		t.Errorf("Location = %q, want \"Frankfurt, Germany\"", g.Location)
	}
	if g.Url != "https://lium.io/browse-pods/rtx5090-node" {
		t.Errorf("Url = %q, want the rent_url", g.Url)
	}
}

func TestParseLiumNodesReportsNodeTotalFlops(t *testing.T) {
	byID := parseFixture(t)
	perGPU, _, _ := gpuSpecs("B300 SXM6 AC")
	if perGPU == 0 {
		t.Fatal("gpuSpecs does not know B300, so node-total FLOPS cannot be checked")
	}
	want := perGPU * 8 / 1e12

	if got := byID["b300-node"].TotalFlops; got != want {
		t.Errorf("TotalFlops = %v, want node total %v", got, want)
	}
}

func TestGpuSpecsIdentifiesB300(t *testing.T) {
	flops, membw, name := gpuSpecs("B300 SXM6 AC")

	if name != "B300" {
		t.Errorf("name = %q, want \"B300\"", name)
	}
	if flops == 0 || membw == 0 {
		t.Errorf("flops = %v, membw = %v, want both non-zero", flops, membw)
	}
}

func TestGpuSpecsSeparatesBlackwellPro6000FromAda(t *testing.T) {
	_, _, server := gpuSpecs("RTX PRO 6000 Blackwell Server Edition")
	_, _, workstation := gpuSpecs("RTX PRO 6000 Blackwell Workstation Edition")
	_, _, ada := gpuSpecs("RTX 6000 Ada Generation")

	if server != "RTX PRO 6000 Blackwell" {
		t.Errorf("server edition = %q, want \"RTX PRO 6000 Blackwell\"", server)
	}
	if workstation != "RTX PRO 6000 Blackwell" {
		t.Errorf("workstation edition = %q, want \"RTX PRO 6000 Blackwell\"", workstation)
	}
	if ada != "RTX 6000 ada pro" {
		t.Errorf("ada generation = %q, want it left on the Ada entry", ada)
	}
}
