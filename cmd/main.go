package main

import (
	"os"

	"github.com/soundprediction/go-predicato/cmd/predicato"
)

func main() {
	if err := predicato.Execute(); err != nil {
		os.Exit(1)
	}
}
