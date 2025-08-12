// Pings supabase dataset and returns values
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	//supabaseURL = strings.TrimRight("SUPABASE_URL", "/")
	supabaseURL = "https://eteavfeiumodbjywzqfa.supabase.co"
	anonKey     = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImV0ZWF2ZmVpdW1vZGJqeXd6cWZhIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NTQ5NTY1MDIsImV4cCI6MjA3MDUzMjUwMn0.OoB4llKDEtXq0CGgPavqMYEAsZtRF8St_C-YPU-MNPw"
	httpc       = &http.Client{Timeout: 10 * time.Second}
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// For /health, returns healthy
	// For /health/{something}, reflects
	path := strings.TrimPrefix(r.URL.Path, "/health/")
	if path == "" {
		fmt.Fprint(w, "Healthy!")
		return
	}
	fmt.Fprint(w, path)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	// Pingpong :)
	fmt.Fprint(w, "pong")
}

// Gets all gpus with given queries
// Ex: GET /gpus?source=vast&location=us&max_price=0.7&min_flopsd=1000&sort=updated_at.desc&limit=200&offset=0
func getHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Build query
	v := url.Values{}
	v.Set("select", "*")

	if s := q.Get("source"); s != "" {
		v.Set("source", "eq."+s)
	}
	if loc := q.Get("location"); loc != "" {
		// case-insensitive substring
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

	// return total count in Content-Range header
	req.Header.Set("Prefer", "count=planned")

	resp, err := httpc.Do(req)
	if err != nil {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("upstream %s: %s", resp.Status, string(b)), resp.StatusCode)
		return
	}

	// Stream through
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.Copy(w, resp.Body)
}

var openapiYAML []byte

func DocsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8"/>
  <title>GPU API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
</head>
<body>
<div id="swagger"></div>
<script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
<script>
  window.ui = SwaggerUIBundle({
    url: '/openapi.yaml',
    dom_id: '#swagger',
    presets: [SwaggerUIBundle.presets.apis],
    layout: "BaseLayout"
  });
</script>
</body>
</html>`)
}

func OpenAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Write(openapiYAML)
}

func main() {
	http.HandleFunc("/health/", healthHandler)
	http.HandleFunc("/", pingHandler)
	http.HandleFunc("/get", getHandler)
	http.HandleFunc("/docs", DocsHandler)
	http.HandleFunc("/openapi.yaml", OpenAPIHandler)

	fmt.Println("Server listening on port 8080...")
	http.ListenAndServe(":8080", nil) // Start the server
}
