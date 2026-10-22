package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"khata-book/models"
	"khata-book/services"
)

type NodeHandler struct {
	svc *services.KhataService
}

func NewNodeHandler(svc *services.KhataService) *NodeHandler {
	return &NodeHandler{svc: svc}
}

// HandleListDepartments handles GET /api/departments
func (h *NodeHandler) HandleListDepartments(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	depts, err := h.svc.ListDepartments(user)
	if err != nil {
		HandleServiceError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, depts)
}

// HandleGetDepartmentTree handles GET /api/departments/{deptID}/tree
func (h *NodeHandler) HandleGetDepartmentTree(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	deptID := r.PathValue("deptID")

	tree, err := h.svc.GetDepartmentTree(user, deptID)
	if err != nil {
		HandleServiceError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, tree)
}

// HandleGetNode handles GET /api/nodes/{id}
func (h *NodeHandler) HandleGetNode(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	nodeID := r.PathValue("id")

	node, ancestors, totals, err := h.svc.GetNodeDetails(user, nodeID)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"node":      node,
		"ancestors": ancestors,
		"rollup":    totals,
		"max_depth": models.MaxDepth,
	})
}

// HandleCreateNode handles POST /api/nodes
func (h *NodeHandler) HandleCreateNode(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)

	var req models.CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	node, err := h.svc.CreateNode(user, &req)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, node)
}

// HandleRenameNode handles PUT /api/nodes/{id}/rename
func (h *NodeHandler) HandleRenameNode(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	nodeID := r.PathValue("id")

	var req models.RenameNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if err := h.svc.RenameNode(user, nodeID, req.Name); err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Node renamed successfully",
	})
}

// HandleMoveNode handles POST /api/nodes/{id}/move
// Validates that moving a node does not exceed the depth limit of 5 for any of its descendants.
func (h *NodeHandler) HandleMoveNode(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	nodeID := r.PathValue("id")

	var req models.MoveNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if strings.TrimSpace(req.NewParentID) == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "new_parent_id is required")
		return
	}

	if err := h.svc.MoveNode(user, nodeID, req.NewParentID); err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Node and its subtree moved successfully",
	})
}

// HandleDeleteNode handles DELETE /api/nodes/{id}
func (h *NodeHandler) HandleDeleteNode(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	nodeID := r.PathValue("id")

	if err := h.svc.DeleteNode(user, nodeID); err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Node and its subtree deleted successfully",
	})
}
