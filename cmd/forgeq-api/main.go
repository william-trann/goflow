package main

import (
	"log"

	"github.com/william-trann/goflow/internal/config"
	"github.com/william-trann/goflow/internal/core"
)

func main() {
	cfg := config.Load()
	system := core.NewInMemorySystem(cfg)

	if err := core.RunUntilSignal(core.NewAPIRuntime(system)); err != nil {
		log.Fatal(err)
	}
}
