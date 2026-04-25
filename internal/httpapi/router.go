package httpapi

import "net/http"

func NewRouter(handler *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handler.Health)

	mux.HandleFunc("POST /api/v1/devices", handler.CreateDevice)

	mux.HandleFunc("POST /api/v1/measurements", handler.CreateMeasurement)
	mux.HandleFunc("GET /api/v1/measurements/latest", handler.LatestMeasurement)

	return mux
}
