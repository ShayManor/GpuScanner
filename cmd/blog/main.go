package main

import (
	"fmt"
	"math/rand"
)

// When this is called, a new blog post is written and added to the supabase table public.blogs
// Type of blogs:
// 1) GPU compare: Compares two similar GPUs
// 2) Current relevant topic: Write about a relevant topic
// 3) Explain GPU architecture: Goes deep into a specific GPU architecture
// 4) Top 5 GPUs for AI training
// 5) CPU vs. GPU vs. TPU vs. NPU
// Examples:
//
//	topics := []string{
//		"Compare the RTX 4090 and RTX 5070 ti GPUs",
//		"Compare the RTX 4070 and RTX 5060 ti GPUs",
//		"Compare the RTX A6000 and RTX 5090 GPUs",
//		"Compare the A100 and H100 GPUs",
//		"Compare the B200 and H200 GPUs",
//		"Bitnets, how they work and how to efficiently optimize for GPUs",
//		"Explain hopper artchitecture",
//		"Explain blackwell artchitecture",
//		"Explain hopper vs blackwell artchitecture",
//		"Explain ampere vs ada artchitecture",
//		"What are the top 5 GPUs for AI training and why?",
//		"Explain CPU vs. GPU vs. TPU vs. NPU",
//	}
func main() {
	var topics []string
	for _ = range 5 {
		topics = append(topics, getTopic())
	}

	topic := topics[rand.Intn(len(topics))]

	data := writeArticle(topic)
	fmt.Println("Wrote article")

	title := getTitle(data)
	fmt.Println("Got title")

	article := Article{title, data}
	upload(article)
	fmt.Println("Uploaded article")
}
