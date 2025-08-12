// api/gpus.go
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// GPU is the response schema (subset shown; keep your full struct if you want).
// swagger:model GPU
type GPU struct {
	// Instance details
	Id          string  `json:"id" bson:"_id"`
	Location    string  `json:"location" bson:"location"`
	Reliability float64 `json:"reliability" bson:"reliability"`
	Duration    float64 `json:"duration_hours" bson:"duration_hours"`
	Source      string  `json:"source" bson:"source"` // e.g., "tensordock", "vast", etc.
	// GPU details
	Name              string  `json:"name" bson:"name"`
	Vram              int     `json:"vram_mb" bson:"vram_mb"`
	TotalFlops        float64 `json:"total_flops" bson:"total_flops"`
	GpuMemoryBandwith float64 `json:"gpu_mem_bw_gbps" bson:"gpu_mem_bw_gbps"`
	NumGPUs           int     `json:"num_gpus" bson:"num_gpus"`
	// CPU specs
	CpuCores float64 `json:"cpu_cores" bson:"cpu_cores"`
	CpuName  string  `json:"cpu_name" bson:"cpu_name"`
	CpuGhz   float64 `json:"cpu_ghz" bson:"cpu_ghz"`
	CpuArch  string  `json:"cpu_arch" bson:"cpu_arch"`
	// Ram
	Ram int `json:"ram_mb" bson:"ram_mb"`
	// SSD
	DiskSpace float64 `json:"disk_space_gb" bson:"disk_space_gb"`
	DiskBW    float64 `json:"disk_bw_gbps" bson:"disk_bw_gbps"`
	DiskName  string  `json:"disk_name" bson:"disk_name"`
	// Internet
	UploadSpeed   float64 `json:"upload_mbps" bson:"upload_mbps"`
	DownloadSpeed float64 `json:"download_mbps" bson:"download_mbps"`
	// Cost
	TotalCostPH      float64 `json:"total_cost_ph" bson:"total_cost_ph"` // PH = per hour
	GpuCostPH        float64 `json:"gpu_cost_ph" bson:"gpu_cost_ph"`
	DiskCostPH       float64 `json:"disk_cost_ph" bson:"disk_cost_ph"`
	UploadCostPH     float64 `json:"upload_cost_ph" bson:"upload_cost_ph"`
	DownloadCostPH   float64 `json:"download_cost_ph" bson:"download_cost_ph"`
	FlopsPerDollarPH float64 `json:"flops_per_dollar_ph" bson:"flops_per_dollar_ph"`

	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

var (
	supabaseURL = "https://eteavfeiumodbjywzqfa.supabase.co/"
	anonKey     = mustEnv("SUPABASE_ANON_KEY")
	httpc       = &http.Client{Timeout: 10 * time.Second}
)

func mustEnv(k string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		panic("missing env var: " + k)
	}
	return v
}

// getHandler godoc
// @Summary     List GPUs
// @Description Returns a JSON array of GPU offers (read-only).
// @Tags        gpus
// @Produce     json
// @Param       source      query  string  false  "Provider (e.g., vast, tensordock, runpod)"
// @Param       location    query  string  false  "Case-insensitive substring match"
// @Param       max_price   query  number  false  "Max total_cost_ph"
// @Param       min_flopsd  query  number  false  "Min flops_per_dollar_ph"
// @Param       sort        query  string  false  "Column.direction (e.g., updated_at.desc)" default(updated_at.desc)
// @Param       limit       query  int     false  "Limit (1-1000)"                           default(200) minimum(1) maximum(1000)
// @Param       offset      query  int     false  "Offset for pagination"                     minimum(0)
// @Success     200         {array}  GPU
// @Failure     400         {string} string  "Bad request"
// @Failure     502         {string} string  "Upstream error"
// @Router      /gpus [get]
func getHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	v := url.Values{}
	v.Set("select", "*")

	if s := q.Get("source"); s != "" {
		v.Set("source", "eq."+s)
	}
	if loc := q.Get("location"); loc != "" {
		v.Set("location", "ilike.*"+loc+"*")
	}
	if mp := q.Get("max_price"); mp != "" {
		v.Set("total_cost_ph", "lt."+mp)
	}
	if mfd := q.Get("min_flopsd"); mfd != "" {
		v.Set("flops_per_dollar_ph", "gte."+mfd)
	}
	sort := q.Get("sort")
	if sort == "" {
		sort = "updated_at.desc"
	}
	v.Set("order", sort)
	limit := q.Get("limit")
	if limit == "" {
		limit = "200"
	}
	v.Set("limit", limit)
	if off := q.Get("offset"); off != "" {
		v.Set("offset", off)
	}

	endpoint := supabaseURL + "/rest/v1/gpus?" + v.Encode()
	req, _ := http.NewRequestWithContext(r.Context(), "GET", endpoint, nil)
	req.Header.Set("apikey", anonKey)
	req.Header.Set("Authorization", "Bearer "+anonKey)
	req.Header.Set("Accept", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("upstream %s: %s", resp.Status, string(b)), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = io.Copy(w, resp.Body)
}
