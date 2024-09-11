package main

import "fmt"

func main() {
	num := 0
	for _, c := range "string" {
		num <<= 8
		num += int(c)
	}

	fmt.Printf("My favourite number is %d.\n", num)
}
