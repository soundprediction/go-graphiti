package main

//go:generate sh -c "cd $(go list -f '{{.Dir}}' -m github.com/LadybugDB/go-ladybug) && go generate ./..."

import (
	"os"

	"github.com/soundprediction/go-predicato/cmd/predicato"
)

func main() {
	if err := predicato.Execute(); err != nil {
		os.Exit(1)
	}
}
