package main

import (
	"fmt"
	"os"
	"os/user"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("id called incorrectly\n")
		os.Exit(1)
	}
	fmt.Printf("id called for user %s\n", os.Args[1])

	// id accepts both usernames and UIDs, so attempt a lookup for both. If
	// either lookup doesn't return an error, exit cleanly.

	_, err := user.Lookup(os.Args[1])
	if err == nil {
		os.Exit(0)
	}

	_, err = user.LookupId(os.Args[1])
	if err == nil {
		os.Exit(0)
	}

	os.Exit(1)
}
