package main

import "fmt"

func main() {
	fmt.Println("Hello GPUs")
	getter := Getter(runpodGetter)
	fmt.Println(scan(getter))
}
