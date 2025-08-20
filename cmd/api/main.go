// cmd/api/main.go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/shaymanor/GpuScanner/docs"
)

// @title           GPU Catalog API
// @version         1.0
// @description     Read-only list of GPU offers. Updated hourly.
// @BasePath        /
// @schemes         https http
func main() {
	log.Println("Starting API server...")
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	log.Println("Setting up Health handler...")

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Println("Setting up gpus handler...")

	r.Get("/gpus", getHandler)
	r.Get("/gpus/count", countHandler)

	// Swagger UI at /docs
	r.Route("/docs", func(r chi.Router) {
		r.Get("/", httpSwagger.WrapHandler)  // handles /docs and issues the 301
		r.Get("/*", httpSwagger.WrapHandler) // serves /docs/{files}
	})
	r.Get("/docs/*", httpSwagger.WrapHandler)

	log.Println("Setting up SPA handler...")

	h, err := spaHandler()
	if err != nil {
		log.Printf("Failed to create SPA handler: %v", err)
	} else {
		r.Mount("/", h)
	}
	// MCP
	mcpSrv := server.NewMCPServer(
		"GPUFinder-MCP", "0.1.0",
		server.WithToolCapabilities(true),
	)

	// search_gpus
	mcpSrv.AddTool(
		mcp.NewTool("search_gpus",
			mcp.WithDescription("Search gpufindr catalogue by name/region/price"),
			mcp.WithString("query", mcp.Description("substring to match in GPU name. * for any.")),
			mcp.WithString("region", mcp.Description("exact region code, e.g. us-south-1, * for any")),
			mcp.WithNumber("max_price", mcp.Description("max USD per-hour price. -1 for any.")),
		),
		searchHandler,
	)

	// fetch_gpu
	mcpSrv.AddTool(
		mcp.NewTool("fetch_gpu",
			mcp.WithDescription("Fetch a single GPU offer by id"),
			mcp.WithString("id", mcp.Required(), mcp.Description("ID returned from search")),
		),
		fetchHandler,
	)

	mcpHandler := server.NewStreamableHTTPServer(
		mcpSrv,
		server.WithStateLess(false),
	)
	sseSrv := server.NewSSEServer(mcpSrv, server.WithStaticBasePath("/mcp"))

	r.Mount("/mcp", http.StripPrefix("/mcp", mcpHandler))
	r.Mount("/mcp/sse", http.StripPrefix("/mcp/sse", sseSrv))

	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	addr := "0.0.0.0:" + port
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
