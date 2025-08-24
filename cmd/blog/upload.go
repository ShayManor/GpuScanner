package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func upload(article Article) {
	base := os.Getenv("SUPABASE_URL") + "/rest/v1/blogs"
	key := os.Getenv("SUPABASE_SERVICE_KEY")

	body, _ := json.Marshal(article)
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
	fmt.Println("post OK")
}
