package state

import "time"

type Session struct {
	Scenario  string    `json:"scenario"`
	StartedAt time.Time `json:"started_at"`
	Challenge bool      `json:"challenge,omitempty"`
	Changes   []Change  `json:"changes"`
}

type Change struct {
	Type          string `json:"type"`
	Target        string `json:"target"`
	OriginalState string `json:"original_state,omitempty"`
	File          string `json:"file,omitempty"`
}
