package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
)

func main() {
	getters := []Getter{lambdaGetter, vastGetter, tensordockGetter, runpodGetter}

	var rows []GPU
	for _, getter := range getters {
		rows = append(rows, scan(getter)...)
	}

	for idx, _ := range rows {
		rows[idx].Id = uuid.New().String()
	}

	base := os.Getenv("SUPABASE_URL") + "/rest/v1/gpus"
	key := os.Getenv("SUPABASE_SERVICE_KEY")

	// 1) DELETE only rows we manage: source in (lambda, vast, tensordock, runpod)
	sources := []string{"lambda", "vast", "tensordock", "runpod"}
	q := url.Values{}
	q.Set("source", "in.("+strings.Join(sources, ",")+")") // URL-encodes () as %28 %29
	delURL := base + "?" + q.Encode()

	delReq, _ := http.NewRequest("DELETE", delURL, nil)
	delReq.Header.Set("apikey", key)
	delReq.Header.Set("Authorization", "Bearer "+key)

	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		log.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode >= 300 {
		b, _ := io.ReadAll(delResp.Body)
		log.Fatalf("delete failed: %s %s", delResp.Status, string(b))
	}

	// 2) INSERT fresh rows
	body, _ := json.Marshal(rows)
	insReq, _ := http.NewRequest("POST", base, bytes.NewReader(body))
	insReq.Header.Set("Content-Type", "application/json")
	insReq.Header.Set("apikey", key)
	insReq.Header.Set("Authorization", "Bearer "+key)

	insResp, err := http.DefaultClient.Do(insReq)
	if err != nil {
		log.Fatal(err)
	}
	defer insResp.Body.Close()
	if insResp.StatusCode >= 300 {
		b, _ := io.ReadAll(insResp.Body)
		log.Fatalf("insert failed: %s %s", insResp.Status, string(b))
	}
	fmt.Println("replace OK")
}
