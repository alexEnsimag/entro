package server

import (
	"alex/entro/pkg/report"
	"sync"
)

// DBStatus is a database where the reports' status is saved
type DBStatus struct {
	db map[report.ID]report.Status
	mu *sync.RWMutex
}

// NewReportStatusDB creates a db for the report status
func NewReportStatusDB() DBStatus {
	return DBStatus{
		db: map[report.ID]report.Status{},
		mu: &sync.RWMutex{},
	}
}

// WriteStatus saves the status of the report
func (db DBStatus) WriteStatus(id report.ID, status report.Status) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.db[id] = status
}

// ReadStatus returns the status of a report if found
func (db DBStatus) ReadStatus(id report.ID) (status report.Status, found bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	status, found = db.db[id]
	return status, found
}
