package main

import (
	"alex/entro/server/pkg/report"
	"alex/entro/server/pkg/server"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	logger := zap.Must(zap.NewProduction())
	logger.Info("Starting API server")

	apiImpl := server.NewAPI(server.NewReportStatusDB(), report.Storage{}, 1000)

	http.HandleFunc("/create", apiImpl.CreateReport)
	http.HandleFunc("/status", apiImpl.GetReportStatus)
	http.HandleFunc("/filePath", apiImpl.GetReportFilePath)

	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		logger.Info("Error while running API server", zap.Error(err))
		return
	}
	logger.Info("API server terminated")
}
