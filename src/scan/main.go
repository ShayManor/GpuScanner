package main

import "fmt"

func main() {
	fmt.Println("Hello GPUs")
	getter := Getter(tensordockGetter)
	fmt.Println(scan(getter))
}
