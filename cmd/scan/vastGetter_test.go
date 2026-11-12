package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Vast rejects the scalar form with
// 400 {"error":"bad_request","msg":"q.rentable: Input should be a valid dictionary"},
// so every filter field has to be an operator dictionary.
func TestVastSearchBodyFiltersWithOperatorDict(t *testing.T) {
	var body struct {
		Q struct {
			Limit    int             `json:"limit"`
			Rentable json.RawMessage `json:"rentable"`
			Verified json.RawMessage `json:"verified"`
		} `json:"q"`
	}
	if err := json.Unmarshal([]byte(vastSearchBody(4096, -1)), &body); err != nil {
		t.Fatalf("search body is not valid JSON: %v", err)
	}

	if body.Q.Limit != 4096 {
		t.Errorf("limit = %d, want 4096", body.Q.Limit)
	}
	if got := strings.TrimSpace(string(body.Q.Rentable)); got != `{"eq":true}` {
		t.Errorf("rentable = %s, want {\"eq\":true}", got)
	}
	if got := strings.TrimSpace(string(body.Q.Verified)); got != `{"eq":true}` {
		t.Errorf("verified = %s, want {\"eq\":true}", got)
	}
}

func TestFetchVastOffersReportsNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"bad_request","msg":"q.rentable: Input should be a valid dictionary"}`))
	}))
	defer srv.Close()

	old := vastSearchURL
	vastSearchURL = srv.URL
	defer func() { vastSearchURL = old }()

	offers, err := fetchVastOffers(10)

	if err == nil {
		t.Fatalf("got %d offers and no error, want an error: a rejected query must not look like an empty marketplace", len(offers))
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %q, want it to carry the status", err)
	}
}

// vastPageOffers renders n offers priced from lo upward, as the search endpoint would.
func vastPageOffers(startID int, n int, lo float64) string {
	rows := make([]string, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, fmt.Sprintf(
			`{"id":%d,"machine_id":%d,"rentable":true,"dph_total":%.4f,"discounted_dph_total":%.4f,"gpu_name":"RTX 4090"}`,
			startID+i, 900+((startID+i)%7), lo+float64(i)/1000, lo+float64(i)/1000))
	}
	return `{"offers":[` + strings.Join(rows, ",") + `]}`
}

// vastCursorOf reads the dph_total lower bound out of a request body, or -1 when absent.
func vastCursorOf(t *testing.T, r *http.Request) float64 {
	t.Helper()
	var body struct {
		Q struct {
			Dph *struct {
				Gte float64 `json:"gte"`
			} `json:"dph_total"`
		} `json:"q"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if body.Q.Dph == nil {
		return -1
	}
	return body.Q.Dph.Gte
}

func TestVastSearchBodyOrdersByPriceAndAddsCursor(t *testing.T) {
	first := vastSearchBody(64, -1)
	if !strings.Contains(first, `"order":[["dph_total","asc"]]`) {
		t.Errorf("first page body lacks the ascending price order: %s", first)
	}
	if strings.Contains(first, `"gte"`) {
		t.Errorf("first page must not carry a cursor: %s", first)
	}

	next := vastSearchBody(64, 1.25)
	if !strings.Contains(next, `"dph_total":{"gte":1.25}`) {
		t.Errorf("cursor page body lacks the price cursor: %s", next)
	}
	var probe map[string]any
	if err := json.Unmarshal([]byte(next), &probe); err != nil {
		t.Errorf("cursor page body is not valid JSON: %v", err)
	}
}

func TestFetchVastOffersWalksPagesUntilShortPage(t *testing.T) {
	var reqs int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs++
		switch cur := vastCursorOf(t, r); {
		case cur < 0:
			w.Write([]byte(vastPageOffers(0, vastPageSize, 0.10)))
		case reqs == 2:
			w.Write([]byte(vastPageOffers(100, vastPageSize, 1.00)))
		default:
			w.Write([]byte(vastPageOffers(200, 5, 2.00))) // short page ends the walk
		}
	}))
	defer srv.Close()
	old := vastSearchURL
	vastSearchURL = srv.URL
	defer func() { vastSearchURL = old }()

	offers, err := fetchVastOffers(40)
	if err != nil {
		t.Fatalf("fetchVastOffers: %v", err)
	}

	if want := vastPageSize*2 + 5; len(offers) != want {
		t.Errorf("got %d offers, want %d across three pages", len(offers), want)
	}
	if reqs != 3 {
		t.Errorf("made %d requests, want 3: the short page ends the walk", reqs)
	}
}

func TestFetchVastOffersDedupesOffersRepeatedAcrossPages(t *testing.T) {
	var reqs int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs++
		if reqs == 1 {
			w.Write([]byte(vastPageOffers(0, vastPageSize, 0.10)))
			return
		}
		w.Write([]byte(vastPageOffers(vastPageSize-10, 12, 0.154))) // 10 repeats, 2 new
	}))
	defer srv.Close()
	old := vastSearchURL
	vastSearchURL = srv.URL
	defer func() { vastSearchURL = old }()

	offers, err := fetchVastOffers(40)
	if err != nil {
		t.Fatalf("fetchVastOffers: %v", err)
	}

	seen := map[int]bool{}
	for _, o := range offers {
		if seen[o.AskID] {
			t.Fatalf("offer id %d returned twice", o.AskID)
		}
		seen[o.AskID] = true
	}
	if want := vastPageSize + 2; len(offers) != want {
		t.Errorf("got %d offers, want %d after dedupe", len(offers), want)
	}
}

func TestFetchVastOffersStopsAtPageBudget(t *testing.T) {
	var reqs int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs++
		w.Write([]byte(vastPageOffers(reqs*1000, vastPageSize, 0.10))) // never a short page
	}))
	defer srv.Close()
	old := vastSearchURL
	vastSearchURL = srv.URL
	defer func() { vastSearchURL = old }()

	if _, err := fetchVastOffers(4); err != nil {
		t.Fatalf("fetchVastOffers: %v", err)
	}
	if reqs != 4 {
		t.Errorf("made %d requests, want the 4 page budget to cap the walk", reqs)
	}
}
