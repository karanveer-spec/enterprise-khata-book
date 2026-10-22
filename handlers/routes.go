package handlers

import (
	"net/http"

	"khata-book/services"
)

type Router struct {
	svc              *services.KhataService
	nodeHandler      *NodeHandler
	entryHandler     *EntryHandler
	analyticsHandler *AnalyticsHandler
	exportHandler    *ExportHandler
}

func NewRouter(svc *services.KhataService) *Router {
	return &Router{
		svc:              svc,
		nodeHandler:      NewNodeHandler(svc),
		entryHandler:     NewEntryHandler(svc),
		analyticsHandler: NewAnalyticsHandler(svc),
		exportHandler:    NewExportHandler(svc),
	}
}

// RegisterRoutes registers all API routes onto the given ServeMux.
func (ro *Router) RegisterRoutes(mux *http.ServeMux) {
	// Identity & RBAC
	mux.HandleFunc("GET /api/me", ro.handleGetMe)
	mux.HandleFunc("GET /api/users", ro.handleListUsers)

	// Macro Overview
	mux.HandleFunc("GET /api/overview", ro.analyticsHandler.HandleGetMacroOverview)

	// Departments
	mux.HandleFunc("GET /api/departments", ro.nodeHandler.HandleListDepartments)
	mux.HandleFunc("GET /api/departments/{deptID}/tree", ro.nodeHandler.HandleGetDepartmentTree)

	// Hierarchy Nodes
	mux.HandleFunc("GET /api/nodes/{id}", ro.nodeHandler.HandleGetNode)
	mux.HandleFunc("POST /api/nodes", ro.nodeHandler.HandleCreateNode)
	mux.HandleFunc("PUT /api/nodes/{id}/rename", ro.nodeHandler.HandleRenameNode)
	mux.HandleFunc("POST /api/nodes/{id}/move", ro.nodeHandler.HandleMoveNode)
	mux.HandleFunc("DELETE /api/nodes/{id}", ro.nodeHandler.HandleDeleteNode)

	// Entries
	mux.HandleFunc("GET /api/nodes/{id}/entries", ro.entryHandler.HandleGetEntries)
	mux.HandleFunc("POST /api/nodes/{id}/entries", ro.entryHandler.HandleCreateEntry)
	mux.HandleFunc("DELETE /api/entries/{id}", ro.entryHandler.HandleDeleteEntry)
	mux.HandleFunc("POST /api/entries/bulk-delete", ro.entryHandler.HandleBulkDeleteEntries)
	mux.HandleFunc("POST /api/entries/bulk-move", ro.entryHandler.HandleBulkMoveEntries)

	// Analytics & Rollups
	mux.HandleFunc("GET /api/nodes/{id}/rollup", ro.analyticsHandler.HandleGetConsolidatedRollUp)
	mux.HandleFunc("GET /api/nodes/{id}/trends", ro.analyticsHandler.HandleGetCashFlowTrends)

	// Export
	mux.HandleFunc("GET /api/nodes/{id}/export", ro.exportHandler.HandleExportCSV)
}

func (ro *Router) handleGetMe(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	WriteJSON(w, http.StatusOK, user)
}

func (ro *Router) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := ro.svc.ListUsers()
	if err != nil {
		HandleServiceError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, users)
}
