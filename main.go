package main

import (
	"os"

	"github.com/kamranahmedse/slim/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
