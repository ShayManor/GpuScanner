package main

func scan(getter Getter) []GPU {
	resp, _ := getter()
	return resp
}
