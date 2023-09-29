package report

import (
	"encoding/json"
	"fmt"
	"os"
)

// Storage is a storage for reports
type Storage struct{}

// Write writes a report
func (s Storage) Write(id ID, data []Entry) error {
	file, err := os.Create("/tmp/" + string(id))
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	_, err = file.Write(dataBytes)
	if err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}

func (s Storage) ReadRaw(id ID) ([]byte, error) {
	data, err := os.ReadFile("/tmp/" + string(id))
	if err != nil {
		return nil, fmt.Errorf("failed to read report: %w", err)
	}
	return data, nil
}
