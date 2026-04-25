package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"plants-monitor/internal/models"
	"plants-monitor/internal/storage"
)

type Handler struct {
	store *storage.Store
}

func NewHandler(store *storage.Store) *Handler {
	return &Handler{
		store: store,
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
	})
}

func (h *Handler) CreateMeasurement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method_not_allowed",
		})
		return
	}

	defer r.Body.Close()

	var input models.CreateMeasurementRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "invalid_json",
			"details": err.Error(),
		})
		return
	}

	if err := validateMeasurement(input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "validation_error",
			"details": err.Error(),
		})
		return
	}

	measurement, err := h.store.InsertMeasurement(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "insert_failed",
			"details": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"ok":          true,
		"measurement": measurement,
	})
}

func (h *Handler) LatestMeasurement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method_not_allowed",
		})
		return
	}

	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "device_id_required",
		})
		return
	}

	measurement, err := h.store.LatestMeasurement(r.Context(), deviceID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "measurement_not_found",
		})
		return
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "latest_failed",
			"details": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"measurement": measurement,
	})
}

func validateMeasurement(input models.CreateMeasurementRequest) error {
	if input.DeviceID == "" {
		return errors.New("device_id is required")
	}

	if input.SoilPercent < 0 || input.SoilPercent > 100 {
		return errors.New("soil_percent must be between 0 and 100")
	}

	if input.SoilRaw < 0 {
		return errors.New("soil_raw must be >= 0")
	}

	if input.LightRaw < 0 {
		return errors.New("light_raw must be >= 0")
	}

	if input.SoilVoltage < 0 {
		return errors.New("soil_voltage must be >= 0")
	}

	if input.LightVoltage < 0 {
		return errors.New("light_voltage must be >= 0")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
