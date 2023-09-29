package report

import (
	"sync"
)

// Status is the status of a report
type Status string

const (
	StatusCreating Status = "creating"
	StatusCreated  Status = "created"
	StatusFailed   Status = "failed"
)

// DBStatus is a database where the reports' status is saved
type DBStatus struct {
	// FIXME (alex): save the details of the error in case an error happened
	db map[ID]Status // FIXME (alex): use a persistent storage
	mu *sync.RWMutex
}

// NewReportStatusDB creates a db for the report status
func NewReportStatusDB() DBStatus {
	return DBStatus{
		db: map[ID]Status{},
		mu: &sync.RWMutex{},
	}
}

// WriteStatus saves the status of the report
func (db DBStatus) WriteStatus(id ID, status Status) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.db[id] = status
}

// ReadStatus returns the status of a report if found
func (db DBStatus) ReadStatus(id ID) (status Status, found bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	status, found = db.db[id]
	return status, found
}
