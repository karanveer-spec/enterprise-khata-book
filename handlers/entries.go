package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"khata-book/models"
	"khata-book/services"
)

type EntryHandler struct {
	svc *services.KhataService
}

func NewEntryHandler(svc *services.KhataService) *EntryHandler {
	return &EntryHandler{svc: svc}
}

// HandleCreateEntry handles POST /api/nodes/{id}/entries
// Records a new entry (Date, Ref/Voucher, Description, Amount) in integer paise.
func (h *EntryHandler) HandleCreateEntry(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	nodeID := r.PathValue("id")

	var req models.CreateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	entry, err := h.svc.CreateEntry(user, nodeID, &req)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, entry)
}

// HandleGetEntries handles GET /api/nodes/{id}/entries
// Lists entries for a specific node with SQL-level Pagination, Date Filtering, and Search.
func (h *EntryHandler) HandleGetEntries(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	nodeID := r.PathValue("id")
	query := r.URL.Query()

	// Parse query params
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 {
		limit = 25
	}

	filter := models.EntryFilter{
		NodeID:             nodeID,
		IncludeDescendants: query.Get("include_descendants") == "true",
		Search:             strings.TrimSpace(query.Get("q")),
		Period:             strings.TrimSpace(query.Get("period")),
		FromDate:           strings.TrimSpace(query.Get("from_date")),
		ToDate:             strings.TrimSpace(query.Get("to_date")),
		Type:               strings.ToUpper(strings.TrimSpace(query.Get("type"))),
		SortBy:             strings.TrimSpace(query.Get("sort_by")),
		SortDir:            strings.TrimSpace(query.Get("sort_dir")),
		Page:               page,
		Limit:              limit,
	}

	resp, err := h.svc.ListEntries(user, filter)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// HandleDeleteEntry handles DELETE /api/entries/{id}
func (h *EntryHandler) HandleDeleteEntry(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	entryID := r.PathValue("id")

	if err := h.svc.DeleteEntry(user, entryID); err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Entry deleted successfully",
	})
}

// HandleBulkDeleteEntries handles POST /api/entries/bulk-delete
func (h *EntryHandler) HandleBulkDeleteEntries(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)

	var req models.BulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if err := h.svc.BulkDeleteEntries(user, req.EntryIDs); err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Entries deleted successfully",
		"count":   len(req.EntryIDs),
	})
}

// HandleBulkMoveEntries handles POST /api/entries/bulk-move
func (h *EntryHandler) HandleBulkMoveEntries(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)

	var req models.BulkMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if strings.TrimSpace(req.TargetNodeID) == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "target_node_id is required")
		return
	}

	if err := h.svc.BulkMoveEntries(user, req.EntryIDs, req.TargetNodeID); err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Entries moved successfully",
		"count":   len(req.EntryIDs),
	})
}
