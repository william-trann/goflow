package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	API       APIConfig
	Worker    WorkerConfig
	Scheduler SchedulerConfig
	Retry     RetryConfig
}

type APIConfig struct {
	Addr string
}

type WorkerConfig struct {
	Queues       []string
	Concurrency  int
	PollInterval time.Duration
}

type SchedulerConfig struct {
	Interval time.Duration
}

type RetryConfig struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

func Load() Config {
	return Config{
		API: APIConfig{
			Addr: stringEnv("FORGEQ_HTTP_ADDR", ":8080"),
		},
		Worker: WorkerConfig{
			Queues:       listEnv("FORGEQ_WORKER_QUEUES", []string{"default"}),
			Concurrency:  intEnv("FORGEQ_WORKER_CONCURRENCY", 4),
			PollInterval: durationEnv("FORGEQ_WORKER_POLL_INTERVAL", 100*time.Millisecond),
		},
		Scheduler: SchedulerConfig{
			Interval: durationEnv("FORGEQ_SCHEDULER_INTERVAL", 250*time.Millisecond),
		},
		Retry: RetryConfig{
			BaseDelay: durationEnv("FORGEQ_RETRY_BASE_DELAY", time.Second),
			MaxDelay:  durationEnv("FORGEQ_RETRY_MAX_DELAY", 30*time.Second),
		},
	}
}

func stringEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func listEnv(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]string(nil), fallback...)
	}

	parts := strings.Split(value, ",")
	queues := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			queues = append(queues, part)
		}
	}
	if len(queues) == 0 {
		return append([]string(nil), fallback...)
	}
	return queues
}
