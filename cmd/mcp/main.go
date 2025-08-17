package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"    // MCP types
	"github.com/mark3labs/mcp-go/server" // transport helpers
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

// --- main: wire up tools & transports ---------------------------
func main() {
	srv := server.NewMCPServer(
		"GPUFinder-MCP", "0.1.0",
		server.WithToolCapabilities(true),
	)

	// search_gpus
	srv.AddTool(
		mcp.NewTool("search_gpus",
			mcp.WithDescription("Search gpufindr catalogue by name/region/price"),
			mcp.WithString("query", mcp.Description("substring to match in GPU name")),
			mcp.WithString("region", mcp.Description("exact region code, e.g. us-south-1")),
			mcp.WithNumber("max_price", mcp.Description("max USD per-hour price")),
		),
		searchHandler,
	)

	// fetch_gpu
	srv.AddTool(
		mcp.NewTool("fetch_gpu",
			mcp.WithDescription("Fetch a single GPU offer by id"),
			mcp.WithString("id", mcp.Required(), mcp.Description("ID returned from search")),
		),
		fetchHandler,
	)

	handler := server.NewStreamableHTTPServer(srv,
		server.WithStateLess(false),
	)
	http.Handle("/mcp/", http.StripPrefix("/mcp", handler))
	http.Handle("/mcp", http.StripPrefix("/mcp", handler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
