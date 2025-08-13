// Scans different GPU sellers, standardizes the data, and aggregates onto supabase db
package main

import (
	"fmt"
	"os"
)

func main() {
	r, _ := vastGetter()
	for _, row := range r {
		fmt.Println(row.toString())
	}
	os.Exit(0)

	//getters := []Getter{
	//	vastGetter,
	//	runpodGetter,
	//	tensordockGetter,
	//}
	//
	//var rows []GPU
	//for _, getter := range getters {
	//	rows = append(rows, scan(getter)...)
	//}
	//
	//body, _ := json.Marshal(rows)
	//req, _ := http.NewRequest("POST",
	//	os.Getenv("SUPABASE_URL")+"/rest/v1/gpus?on_conflict=id",
	//	bytes.NewReader(body),
	//)
	//// Upsert:
	//req.Header.Set("Prefer", "resolution=merge-duplicates")
	//req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("apikey", os.Getenv("SUPABASE_SERVICE_KEY"))
	//req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_KEY"))
	//
	//resp, err := http.DefaultClient.Do(req)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer resp.Body.Close()
	//if resp.StatusCode >= 300 {
	//	b, _ := io.ReadAll(resp.Body)
	//	log.Fatalf("upsert failed: %s %s", resp.Status, string(b))
	//}
}
