package main

type Configuration struct {
	AccessKey    string `json:"key"`
	AccessSecret string `json:"secret"`
	Bucket       string `json:"bucket"`
}

// Task struct declaration
type Task struct {
	TaskID      string `json:"taskid,omitempty"`
	User        string `json:"user"`
	Description string `json:"description"`
	Priority    *int   `json:"priority"`
	Completed   string `json:"completed"`
}

type Taskid struct {
	TaskID string `json:"taskid"`
}
