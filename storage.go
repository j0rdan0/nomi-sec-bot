package main

import (
	"encoding/json"
	"os"
)

type State struct {
	LastCommitSHA string `json:"last_commit_sha"`
}

const stateFile = "state.json"

func LoadState() (*State, error) {
	data, err := os.ReadFile(stateFile)
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

	return os.WriteFile(stateFile, data, 0644)
}
