package main

import (
	"context"
	"fmt"
	"log"

	"github.com/william-trann/goflow/internal/config"
	"github.com/william-trann/goflow/internal/core"
	"github.com/william-trann/goflow/internal/model"
	"github.com/william-trann/goflow/internal/worker"
)

func main() {
	cfg := config.Load()
	system := core.NewInMemorySystem(cfg)

	handler := worker.HandlerFunc(func(_ context.Context, job *model.Job) error {
		fmt.Printf("processed queue=%s job=%s payload=%s\n", job.Queue, job.ID, string(job.Payload))
		return nil
	})

	for _, queueName := range cfg.Worker.Queues {
		system.Register(queueName, handler)
	}

	if err := core.RunUntilSignal(core.NewCombinedRuntime(system)); err != nil {
		log.Fatal(err)
	}
}
