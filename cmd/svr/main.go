package main

import (
	"os"

	"github.com/horonlee/servora/cmd/svr/internal/root"
)

func main() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
