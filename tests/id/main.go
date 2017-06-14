package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("id called incorrectly\n")
		os.Exit(1)
	}
	fmt.Printf("id called for user %s\n", os.Args[1])
}
