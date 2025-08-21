package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func fetchCatalogue(order string) ([]GPU, error) {
	url := fmt.Sprintf("https://gpufindr.com/gpus?sort=%s", order)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list []GPU
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	return list, nil
}

func searchHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	q := strings.ToLower(req.GetString("query", ""))
	region := req.GetString("region", "*")
	maxP := req.GetFloat("max_price", 0)
	minS := req.GetFloat("min_score", 0)
	order := req.GetString("order_by", "updated_at.desc")
	limit := req.GetInt("limit", 50)
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := req.GetInt("offset", 0)
	if offset < 0 {
		offset = 0
	}

	gpus, err := fetchCatalogue(order)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to reach gpufindr", err), nil
	}

	var hits []GPU
	for _, g := range gpus {
		if q != "" && q != "*" && !strings.Contains(strings.ToLower(g.Name), q) {
			continue
		}
		if g.Score < minS {
			continue
		}
		if region != "*" && region != g.Location {
			continue
		}
		if maxP > 0 && g.TotalCostPH > maxP {
			continue
		}
		hits = append(hits, g)
	}

	// Calculate safe slice bounds
	start := offset
	if start > len(hits) {
		start = len(hits)
	}
	end := offset + limit
	if end > len(hits) {
		end = len(hits)
	}

	// Get the actual slice
	var results []GPU
	if start < len(hits) {
		results = hits[start:end]
	} else {
		results = []GPU{} // empty if offset is beyond array
	}

	summary := fmt.Sprintf("Found %d offers; returning %d (offset=%d, limit=%d).",
		len(hits), len(results), offset, limit)

	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(results); err != nil {
		return mcp.NewToolResultError("failed to marshal results"), nil
	}
	js := buf.Bytes()
	js = bytes.TrimSuffix(js, []byte("\n"))

	res := mcp.NewToolResultText(summary + "\n\n" + string(js))

	return res, nil
}

func fetchHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	gpus, err := fetchCatalogue("updated_at.desc")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to reach gpufindr", err), nil
	}
	for _, g := range gpus {
		if g.Id == id {
			js, _ := json.Marshal(g)
			return mcp.NewToolResultStructured(json.RawMessage(js), ""), nil
		}
	}
	return mcp.NewToolResultError(fmt.Sprintf("GPU %s not found", id)), nil
}
