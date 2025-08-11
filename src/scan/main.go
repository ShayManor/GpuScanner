package main

import "fmt"

func main() {
	fmt.Println("Hello GPUs")
	getter := Getter(vastGetter)
	fmt.Println(scan(getter))
}
