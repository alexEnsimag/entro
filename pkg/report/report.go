package report

import (
	"time"

	"github.com/google/uuid"
)

// ID is the ID of a report
type ID string

// SecretMetadata is the metadata of a secret
type SecretMetadata struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Region string `json:"region"`
}

// AuditTrail is an audit trail
type AuditTrail struct {
	UserName string    `json:"userName"`
	Action   string    `json:"action"`
	Time     time.Time `json:"time"`
}

// Entry is an entry of a report
type Entry struct {
	SecretMetadata
	Logs []AuditTrail `json:"logs"`
}

// Status is the status of a report
type Status string

// Set of status for a report
const (
	StatusCreating Status = "creating"
	StatusCreated  Status = "created"
	StatusFailed   Status = "failed"
)

// GenerateID generates a report ID
func GenerateID() ID {
	return ID(uuid.New().String())
}

func IsValidID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
