package handlers

import (
	"net/http"

	"khata-book/models"
	"khata-book/services"
)

type AnalyticsHandler struct {
	svc *services.KhataService
}

func NewAnalyticsHandler(svc *services.KhataService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

// HandleGetConsolidatedRollUp handles GET /api/nodes/{id}/rollup
// Returns direct and descendant roll-up totals using the materialized path.
func (h *AnalyticsHandler) HandleGetConsolidatedRollUp(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	nodeID := r.PathValue("id")

	rollup, err := h.svc.GetConsolidatedRollUp(user, nodeID)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, rollup)
}

// HandleGetCashFlowTrends handles GET /api/nodes/{id}/trends
// Returns time-series data grouping Inflow (Jama) vs Outflow (Udhar) by month.
func (h *AnalyticsHandler) HandleGetCashFlowTrends(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	nodeID := r.PathValue("id")

	trends, err := h.svc.GetCashFlowTrends(user, nodeID)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	if trends == nil {
		trends = make([]*models.MonthlyTrend, 0)
	}

	WriteJSON(w, http.StatusOK, trends)
}

// HandleGetMacroOverview handles GET /api/overview
// Returns organization-wide totals grouped by department.
func (h *AnalyticsHandler) HandleGetMacroOverview(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)

	overview, err := h.svc.GetMacroOverview(user)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, overview)
}
