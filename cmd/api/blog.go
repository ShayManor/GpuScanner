package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type data struct {
	body  string
	title string
}

// Gets all blog posts and returns them
func blogHandler(w http.ResponseWriter, r *http.Request) {
	base := os.Getenv("SUPABASE_URL")
	key := os.Getenv("SUPABASE_ANON_KEY")
	if base == "" || key == "" {
		http.Error(w, "server not configured: SUPABASE_URL/SUPABASE_ANON_KEY missing", http.StatusInternalServerError)
		return
	}

	v := url.Values{}
	v.Set("select", "title,data")
	// Adjust to your schema; if you track publish dates:
	v.Set("order", "created_at.desc")

	// If a specific post is requested:
	if slug := r.URL.Query().Get("slug"); slug != "" {
		v.Set("slug", "eq."+slug)
		v.Set("limit", "1")
	} else {
		// Otherwise list posts with pagination
		limit := r.URL.Query().Get("limit")
		if limit == "" {
			limit = "100"
		}
		v.Set("limit", limit)
		if off := r.URL.Query().Get("offset"); off != "" {
			v.Set("offset", off)
		}
		// If you have a boolean `published` column and don’t want drafts:
		// v.Set("published", "eq.true")
	}

	endpoint := fmt.Sprintf("%s/rest/v1/blogs?%s", strings.TrimRight(base, "/"), v.Encode())

	req, _ := http.NewRequestWithContext(r.Context(), "GET", endpoint, nil)
	req.Header.Set("apikey", key)
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
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
