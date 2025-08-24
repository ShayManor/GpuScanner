package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/openai/openai-go/v2" // imported as openai
	"github.com/openai/openai-go/v2/packages/param"
)

func getWritePrompt(topic string) string {
	return fmt.Sprintf("You are an expert blog writer who is creative, entertaining, and writes intruiging blog posts which viewers care about. You are focusing on the ML landscape with a focus on cheap GPUs. A few rules: Don't use em dashes. Don't writeArticle fluff. Ensure you hit important keywords revolving the topic. Everything is in the context of ML/AI and GPUs. Write it well and ensure it is interesting to read with accurate facts. Only return the blog in markdown form without a getTitle.\n\nTopic: %s", topic)
}

func getTitlePrompt(data string) string {
	return fmt.Sprintf("You are an article labeler and you will write a short, catchy getTitle for the given article. Don't use markdown and ONLY return the getTitle, nothing else. Ensure you encapsulate all details and it is catchy. \n\nArticle: %s", data)
}

func getTopicPrompt() string {
	return `You are getting topics for unique and interesting topics for blog posts regarding gpus and AI/ML. Especially focus on GPUs with ML. Be creative and unique. Focus on new-gen and hot topics. Some examples are: Compare the RTX 4090 and RTX 5070 ti GPUs, Explain blackwell artchitecture, What are the top 5 GPUs for AI training and why... Write another unique and different topic to these that still matches the theme of an interesting blog post. It should be less than a sentence and be intruiging to read about. Be unique and creative. Only return the topic, nothing else. Do NOT mention: liquid cooling, data centers, "reshaping the future", "top X", comparisons ("vs", "compare"), buyer's guides, benchmarks, 4090, 5070 Ti, "Blackwell architecture".`
}

// Removed gpt formatting
func sanitize(data string) string {
	options := []string{
		"```markdown",
		"```md",
		"```mark",
		"```",
	}
	var start int
	for _, option := range options {

		start = strings.Index(data, option)
		if start != -1 {
			start += len(option)
			break
		}
	}
	if start == -1 {
		return data
	}
	end := strings.Index(data[start:], "```")
	return data[start : end+start]
}

// Call chatgpt with prompt and info, writes blog post and returns it
func writeArticle(topic string) string {
	client := openai.NewClient()
	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(getWritePrompt(topic)),
		},
		Model: openai.ChatModelGPT5ChatLatest,
	})
	if err != nil {
		panic(err.Error())
	}
	res := sanitize(chatCompletion.Choices[0].Message.Content)
	fmt.Println(res)
	return res
}

func getTitle(data string) string {
	client := openai.NewClient()
	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(getTitlePrompt(data)),
		},
		Model: openai.ChatModelGPT5ChatLatest,
	})
	if err != nil {
		panic(err.Error())
	}
	res := sanitize(chatCompletion.Choices[0].Message.Content)
	fmt.Println(res)
	return res
}

func getTopic() string {
	client := openai.NewClient()
	var temp param.Opt[float64]
	temp.Value = 1.5
	var topP param.Opt[float64]
	topP.Value = 0.8
	var N param.Opt[int64]
	N.Value = 2
	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(getTopicPrompt()),
		},
		Model:       openai.ChatModelGPT5ChatLatest,
		Temperature: temp,
		TopP:        topP,
		N:           N,
	})
	if err != nil {
		panic(err.Error())
	}
	res := sanitize(chatCompletion.Choices[rand.Intn(len(chatCompletion.Choices))].Message.Content)
	fmt.Println(res)
	return res
}
