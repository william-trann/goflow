package model

type QueueStats struct {
	Name         string `json:"name"`
	Ready        int    `json:"ready"`
	Scheduled    int    `json:"scheduled"`
	Retrying     int    `json:"retrying"`
	Processing   int    `json:"processing"`
	Succeeded    int    `json:"succeeded"`
	DeadLettered int    `json:"dead_lettered"`
	Total        int    `json:"total"`
}

type SystemStats struct {
	Queues       int `json:"queues"`
	Ready        int `json:"ready"`
	Scheduled    int `json:"scheduled"`
	Retrying     int `json:"retrying"`
	Processing   int `json:"processing"`
	Succeeded    int `json:"succeeded"`
	DeadLettered int `json:"dead_lettered"`
	Total        int `json:"total"`
}
