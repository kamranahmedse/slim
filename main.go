package main

import (
	"os"

	"github.com/kamranahmedse/localname/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
