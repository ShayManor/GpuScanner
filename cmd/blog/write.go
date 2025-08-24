package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v2" // imported as openai
)

func getWritePrompt(topic string) string {
	return fmt.Sprintf("You are an expert blog writer who is creative, entertaining, and writes intruiging blog posts which viewers care about. You are focusing on the ML landscape with a focus on cheap GPUs. A few rules: Don't use em dashes. Don't writeArticle fluff. Ensure you hit important keywords revolving the topic. Everything is in the context of ML/AI and GPUs. Write it well and ensure it is interesting to read with accurate facts. Only return the blog in markdown form without a getTitle.\n\nTopic: %s", topic)
}

func getTitlePrompt(data string) string {
	return fmt.Sprintf("You are an article labeler and you will write a short, catchy getTitle for the given article. Don't use markdown and ONLY return the getTitle, nothing else. Ensure you encapsulate all details and it is catchy. \n\nArticle: %s", data)
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
