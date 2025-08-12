// cmd/api/main.go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/shaymanor/GpuScanner/docs"
)

// @title           GPU Catalog API
// @version         1.0
// @description     Read-only list of GPU offers. Updated hourly.
// @BasePath        /
// @schemes         https http
func main() {
	r := chi.NewRouter()

	// API
	r.Get("/gpus", getHandler)

	// Swagger UI at /docs
	r.Get("/docs/*", httpSwagger.WrapHandler)

	addr := ":" + coalesce(os.Getenv("PORT"), "8080")
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func coalesce(v, d string) string {
	if v != "" {
		return v
	}
	return d
}
