package main

import "fmt"

func scan(getter Getter) []GPU {
	resp, err := getter()
	if err != nil {
		fmt.Println("Error fetching GPU data:", err)
	}
	return resp
}
