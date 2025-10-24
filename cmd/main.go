package main

import (
	"os"

	"github.com/soundprediction/go-graphiti/cmd/graphiti"
)

func main() {
	if err := graphiti.Execute(); err != nil {
		os.Exit(1)
	}
}
