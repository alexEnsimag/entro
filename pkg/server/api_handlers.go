package server

import (
	"alex/entro/pkg/connectors"
	"alex/entro/pkg/report"
	"encoding/json"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"go.uber.org/zap"
)

// API is an API
type API struct {
	logger                 *zap.Logger
	reportDBStatus         DBStatus
	reportStorage          report.Storage
	reportCreationRequests chan reportCreationRequest
}

// NewAPI creates an API
func NewAPI(logger *zap.Logger, reportDBStatus DBStatus, reportStorage report.Storage, requestBufferSize int) API {
	api := API{
		logger:                 logger,
		reportDBStatus:         reportDBStatus,
		reportStorage:          reportStorage,
		reportCreationRequests: make(chan reportCreationRequest, requestBufferSize),
	}
	go api.createReport()
	return api
}

type reportCreationRequest struct {
	id                report.ID
	secretManager     connectors.SecretsManager
	auditTrailManager connectors.AuditTrailManager
}

// CreateReportFromAWS initiates the creation of a report from AWS
func (a API) CreateReportFromAWS(w http.ResponseWriter, r *http.Request) {
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
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger := a.logger.With(zap.String("awsAccessKeyID", body.AWSAccessKeyID), zap.String("awsRegion", body.AWSRegion))

	// creation AWS session
	s, err := session.NewSession(&aws.Config{
		Region:      aws.String(body.AWSRegion),
		Credentials: credentials.NewStaticCredentials(body.AWSAccessKeyID, body.AWSSecretAccessKey, body.AWSSessionToken),
	})
	if err != nil {
		logger.Debug("failed to create AWS session", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// add report request to queue
	id := report.GenerateID()
	select {
	case a.reportCreationRequests <- reportCreationRequest{
		id: id,
		secretManager: connectors.AWSSecretsManager{
			AWSSession: *s,
			Region:     body.AWSRegion,
		},
		auditTrailManager: connectors.AWSCloudTrail{
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(respBytes); err != nil {
			logger.Debug("failed to write response", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Info("Successfully created report request", zap.Any("reportID", id))
		w.WriteHeader(http.StatusAccepted)
		a.reportDBStatus.WriteStatus(id, report.StatusCreating)
	default:
		logger.Debug("failed to create report request, channel is full")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// GetReportStatus returns the status of a report
func (a API) GetReportStatus(w http.ResponseWriter, r *http.Request) {
	reportID := r.URL.Query().Get("reportID")
	logger := a.logger.With(zap.String("reportID", reportID))

	if !report.IsValidID(reportID) {
		logger.Debug("wrong report ID")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	status, found := a.reportDBStatus.ReadStatus(report.ID(reportID))
	if !found {
		logger.Debug("report not found")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp := struct {
		ReportStatus report.Status `json:"status"`
	}{
		ReportStatus: status,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		logger.Debug("failed to marshal response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(respBytes); err != nil {
		logger.Debug("failed to write response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	logger.Info("Successfully returned report status", zap.Any("reportStatus", status))
}

func (a API) GetReportFilePath(w http.ResponseWriter, r *http.Request) {
	reportID := r.URL.Query().Get("reportID")
	logger := a.logger.With(zap.String("reportID", reportID))

	if !report.IsValidID(reportID) {
		logger.Debug("wrong report ID")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filePath := reportFilePath(report.ID(reportID))
	if _, err := os.Stat(filePath); err != nil {
		logger.Debug("report not found")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp := struct {
		Path string `json:"path"`
	}{
		Path: filePath,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		logger.Debug("failed to marshal response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(respBytes); err != nil {
		logger.Debug("failed to write response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	logger.Info("Successfully returned report path", zap.Any("reportPath", filePath))
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

		if err := a.reportStorage.Write(reportFilePath(r.id), reportData); err != nil {
			logger.Debug("failed to create report", zap.Error(err))
			a.reportDBStatus.WriteStatus(r.id, report.StatusFailed)
		} else {
			a.reportDBStatus.WriteStatus(r.id, report.StatusCreated)
		}
	}
}

func reportFilePath(reportID report.ID) string {
	return "/tmp/" + string(reportID)
}
