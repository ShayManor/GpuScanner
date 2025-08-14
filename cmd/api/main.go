// cmd/api/main.go
package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	_ "github.com/shaymanor/gpuscanner/cmd/api/docs"
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

	r.Get("/docs/*", httpSwagger.WrapHandler)

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
