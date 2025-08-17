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

func fetchCatalogue() ([]GPU, error) {
	resp, err := http.Get("https://gpufindr.com/gpus")
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
	region := req.GetString("region", "")
	maxP := req.GetFloat("max_price", 0)

	gpus, err := fetchCatalogue()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to reach gpufindr", err), nil
	}

	var hits []GPU
	for _, g := range gpus {
		if q != "" && !strings.Contains(strings.ToLower(g.Name), q) {
			continue
		}
		if region != "" && region != g.Location {
			continue
		}
		if maxP > 0 && g.TotalCostPH > maxP {
			continue
		}
		hits = append(hits, g)
	}

	fmt.Printf("Found %d hits\n", len(hits))

	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false) // Disable HTML escaping
	if err := encoder.Encode(hits); err != nil {
		return mcp.NewToolResultError("failed to marshal results"), nil
	}
	js := buf.Bytes()
	js = bytes.TrimSuffix(js, []byte("\n"))

	return mcp.NewToolResultStructured(json.RawMessage(js), ""), nil
}

func fetchHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	gpus, err := fetchCatalogue()
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
