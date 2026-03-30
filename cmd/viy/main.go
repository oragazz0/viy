package main

import (
	"os"

	"github.com/oragazz0/viy/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
