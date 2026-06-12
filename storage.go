package main

import (
	"encoding/json"
	"os"
	"time"
)

type State struct {
	LastCommitSHA        string    `json:"last_commit_sha"`
	StartupCommitSHA     string    `json:"startup_commit_sha,omitempty"`
	LastNotificationTime time.Time `json:"last_notification_time,omitempty"`
}

func getStateFile() string {
	if path := os.Getenv("STATE_FILE_PATH"); path != "" {
		return path
	}
	return "state.json"
}

func LoadState() (*State, error) {
	data, err := os.ReadFile(getStateFile())
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, err
	}

	var state State
	err = json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

func SaveState(state *State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(getStateFile(), data, 0644)
}
