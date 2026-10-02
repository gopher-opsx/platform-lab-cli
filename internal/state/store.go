package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const labDirectory = ".lab"
const stateFile = "state.json"

func Save(root string, session Session) error {
	dir := filepath.Join(root, labDirectory)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create lab directory: %w", err)
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode lab state: %w", err)
	}

	path := filepath.Join(dir, stateFile)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("save lab state: %w", err)
	}

	return nil
}

func Load(root string) (Session, error) {
	path := filepath.Join(root, labDirectory, stateFile)

	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}, fmt.Errorf("read lab state: %w", err)
	}

	var session Session

	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, fmt.Errorf("decode lab state: %w", err)
	}

	return session, nil
}

func Exists(root string) bool {
	path := filepath.Join(root, labDirectory, stateFile)

	_, err := os.Stat(path)

	return err == nil
}

func Clear(root string) error {
	path := filepath.Join(root, labDirectory, stateFile)

	err := os.Remove(path)

	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove lab state: %w", err)
	}

	return nil
}
