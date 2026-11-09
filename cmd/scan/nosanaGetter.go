package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---- Nosana public markets list (no key) ----
// GET https://dashboard.k8s.prd.nos.ci/api/markets/
// One entry per market (GPU class). usd_reward_per_hour is what the host earns;
// the renter also pays network_fee_percentage on top of it.
type nosanaMarket struct {
	Address              string   `json:"address"`
	Slug                 string   `json:"slug"`
	Name                 string   `json:"name"`
	Type                 string   `json:"type"` // COMMUNITY, PREMIUM, OTHER
	USDRewardPerHour     *float64 `json:"usd_reward_per_hour"`
	NetworkFeePercentage float64  `json:"network_fee_percentage"`
	Client               bool     `json:"client"` // private, client-owned market
	Metadata             []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"metadata"` // "vram" is the whole market's VRAM, e.g. "576GB"
}

// "NVIDIA Pro 6000 (6x)" rents six GPUs at once; other markets rent one.
var nosanaCountRe = regexp.MustCompile(`\((\d+)x\)`)

func nosanaGPUCount(name string) int {
	if m := nosanaCountRe.FindStringSubmatch(name); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			return n
		}
	}
	return 1
}

// nosanaVramGB returns the whole market's VRAM: from metadata, else from a size
// in the market name ("NVIDIA A100 80GB Community"), else the card's only size.
func nosanaVramGB(m nosanaMarket, num int, name string) int {
	for _, kv := range m.Metadata {
		if kv.Key == "vram" {
			v, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(kv.Value), "GB"))
			return v
		}
	}
	if gb := parseVRAMMB(m.Name) / 1024; gb > 0 {
		return gb * num
	}
	return standardVramGB[name] * num
}

func parseNosana(r io.Reader) ([]GPU, error) {
	var markets []nosanaMarket
	if err := json.NewDecoder(r).Decode(&markets); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]GPU, 0, len(markets))
	for _, m := range markets {
		if m.Client || m.USDRewardPerHour == nil || *m.USDRewardPerHour <= 0 {
			continue
		}
		flops, memBWGBs, name := gpuSpecs(proBlackwellName(m.Name))
		if name == "unknown" {
			continue
		}
		num := nosanaGPUCount(m.Name)
		price := *m.USDRewardPerHour * (1 + m.NetworkFeePercentage/100)
		totalFlops := float64(num) * flops / 1e12

		newGpu := GPU{
			_Id: m.Slug,

			Name:              name,
			Vram:              nosanaVramGB(m, num, name) / num * 1024,
			TotalFlops:        totalFlops,
			GpuMemoryBandwith: memBWGBs,
			NumGPUs:           num,

			TotalCostPH:      price,
			GpuCostPH:        price / float64(num),
			FlopsPerDollarPH: totalFlops / price,
			Source:           "nosana",
			Url:              "https://dashboard.nosana.com/markets/" + m.Address,
		}
		newGpu.Score = calculateScore(newGpu)
		newGpu.ScoreDPH = newGpu.Score / newGpu.TotalCostPH
		out = append(out, newGpu)
	}
	return out, nil
}

func nosanaGetter() ([]GPU, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get("https://dashboard.k8s.prd.nos.ci/api/markets/")
	if err != nil {
		return nil, fmt.Errorf("fetch nosana markets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("nosana API status %s", resp.Status)
	}

	out, err := parseNosana(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d Nosana GPUs\n", len(out))
	return out, nil
}
