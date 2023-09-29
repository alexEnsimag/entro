package api

import (
	"alex/entro/server/pkg/report"
	"alex/entro/server/pkg/sources"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"go.uber.org/zap"
)

// API is an API
type API struct {
	logger                 *zap.Logger
	reportDBStatus         report.DBStatus
	reportStorage          report.Storage
	reportCreationRequests chan reportCreationRequest
}

// NewAPI creates an API
func NewAPI(reportDBStatus report.DBStatus, reportStorage report.Storage, requestBufferSize int) API {
	api := API{
		reportDBStatus:         reportDBStatus,
		reportStorage:          reportStorage,
		reportCreationRequests: make(chan reportCreationRequest, requestBufferSize),
	}
	go api.createReport()
	return api
}

type reportCreationRequest struct {
	id                report.ID
	secretManager     sources.SecretsManager
	auditTrailManager sources.AuditTrailManager
}

// CreateReport initiates the creation of a report
// FIXME (alex): endpoint name should explicitly contain the sources (AWS secrets manager and cloud trail)
func (a API) CreateReport(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AWSAccessKeyID     string `json:"awsAccessKeyID"`
		AWSSecretAccessKey string `json:"awsSecretAccessKey"`
		AWSSessionToken    string `json:"awsSessionToken"`
		AWSRegion          string `json:"awsRegion"`
	}

	// decode body
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&body)
	if err != nil {
		a.logger.Debug("failed to decode body", zap.Error(err))
		// FIXME (alex): send back explicit error
		w.WriteHeader(http.StatusBadRequest)
	}

	logger := a.logger.With(zap.String("awsAccessKeyID", body.AWSAccessKeyID), zap.String("awsRegion", body.AWSRegion))
	// FIXME (alex): verify region is valid

	// creation AWS session
	s, err := session.NewSession(&aws.Config{
		Region:      aws.String(body.AWSRegion),
		Credentials: credentials.NewStaticCredentials(body.AWSAccessKeyID, body.AWSSecretAccessKey, body.AWSSessionToken),
	})
	if err != nil {
		logger.Debug("failed to create AWS session", zap.Error(err))
		// FIXME (alex): send back explicit error
		w.WriteHeader(http.StatusInternalServerError)
	}

	// add report request to queue
	id := report.GenerateID()
	select {
	case a.reportCreationRequests <- reportCreationRequest{
		id: id,
		secretManager: sources.AWSSecretsManager{
			AWSSession: *s,
			Region:     body.AWSRegion,
		},
		auditTrailManager: sources.AWSCloudTrail{
			AWSSession: *s,
			Region:     body.AWSRegion,
		},
	}:
		resp := struct {
			ReportID report.ID `json:"reportID"`
		}{
			ReportID: id,
		}
		respBytes, err := json.Marshal(resp)
		if err != nil {
			logger.Debug("failed to marshal response", zap.Error(err))
			// FIXME (alex): send back explicit error
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(respBytes); err != nil {
			logger.Debug("failed to write response", zap.Error(err))
			// FIXME (alex): send back explicit error
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Info("Successfully created report request", zap.Any("reportID", id))
		w.WriteHeader(http.StatusAccepted)
		a.reportDBStatus.WriteStatus(id, report.StatusCreating)
	default:
		logger.Debug("failed to create report request, channel is full")
		// FIXME (alex): send back explicit error
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// GetReportStatus returns the status of a report
func (a API) GetReportStatus(w http.ResponseWriter, r *http.Request) {
	reportID := r.URL.Query().Get("reportID")
	logger := a.logger.With(zap.String("reportID", reportID))

	if !report.IsValidID(reportID) {
		logger.Debug("wrong report ID")
		// FIXME (alex): send back explicit error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	status, found := a.reportDBStatus.ReadStatus(report.ID(reportID))
	if !found {
		logger.Debug("report not found")
		// FIXME (alex): send back explicit error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp := struct {
		ReportStatus report.Status `json:"reportStatus"`
	}{
		ReportStatus: status,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		logger.Debug("failed to marshal response", zap.Error(err))
		// FIXME (alex): send back explicit error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(respBytes); err != nil {
		logger.Debug("failed to write response", zap.Error(err))
		// FIXME (alex): send back explicit error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	logger.Info("Successfully returned report status", zap.Any("reportStatus", status))
	w.WriteHeader(http.StatusOK)
}

// DownloadReport downloads a report
func (a API) DownloadReport(w http.ResponseWriter, r *http.Request) {
	reportID := r.URL.Query().Get("reportID")
	logger := a.logger.With(zap.String("reportID", reportID))

	if !report.IsValidID(reportID) {
		logger.Debug("wrong report ID")
		// FIXME (alex): send back explicit error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data, err := a.reportStorage.ReadRaw(report.ID(reportID))
	if err != nil {
		logger.Debug("failed to read report", zap.Error(err))
	}

	if _, err := w.Write(data); err != nil {
		logger.Debug("failed to write response", zap.Error(err))
		// FIXME (alex): send back explicit error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	logger.Info("Successfully downloaded report")
	w.WriteHeader(http.StatusOK)
}

func (a API) createReport() {
	var reportData []report.Entry
	for r := range a.reportCreationRequests {
		logger := a.logger.With(zap.Any("reportID", r.id))

		secrets, err := r.secretManager.ListSecrets()
		if err != nil {
			logger.Debug("failed to list secrets", zap.Error(err))
			a.reportDBStatus.WriteStatus(r.id, report.StatusFailed)
			continue
		}

		var auditTrails []report.AuditTrail
		for _, s := range secrets {
			// FIXME (alex): retry
			auditTrails, err = r.auditTrailManager.ListAuditTrails(s.Name)
			if err != nil {
				logger.Debug("failed to list logs", zap.String("secretName", s.Name), zap.Error(err))
				a.reportDBStatus.WriteStatus(r.id, report.StatusFailed)
				break
			}
			reportData = append(reportData, report.Entry{
				SecretMetadata: s,
				Logs:           auditTrails,
			})
		}
		if err != nil {
			continue
		}

		if err := a.reportStorage.Write(r.id, reportData); err != nil {
			logger.Debug("failed to create report", zap.Error(err))
			a.reportDBStatus.WriteStatus(r.id, report.StatusFailed)
		} else {
			a.reportDBStatus.WriteStatus(r.id, report.StatusCreated)
		}
	}
}
