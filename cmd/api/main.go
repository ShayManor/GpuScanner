// cmd/api/main.go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
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
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
	})

	// Serve swagger.json
	r.Get("/docs/doc.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/swagger.json")
	})

	// Serve Swagger UI
	r.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"), // Point to the swagger.json endpoint
	))

	log.Println("Setting up SPA handler...")

	h, err := spaHandler()
	if err != nil {
		log.Printf("Failed to create SPA handler: %v", err)
	} else {
		r.Mount("/", h)
	}
	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	addr := "0.0.0.0:" + port
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
