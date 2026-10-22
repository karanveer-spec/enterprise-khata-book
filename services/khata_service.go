package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"khata-book/models"
	"khata-book/repository"
)

type KhataService struct {
	nodeRepo      *repository.NodeRepository
	entryRepo     *repository.EntryRepository
	analyticsRepo *repository.AnalyticsRepository
	userRepo      *repository.UserRepository
}

func NewKhataService(
	nodeRepo *repository.NodeRepository,
	entryRepo *repository.EntryRepository,
	analyticsRepo *repository.AnalyticsRepository,
	userRepo *repository.UserRepository,
) *KhataService {
	return &KhataService{
		nodeRepo:      nodeRepo,
		entryRepo:     entryRepo,
		analyticsRepo: analyticsRepo,
		userRepo:      userRepo,
	}
}

// CheckDepartmentAccess ensures the authenticated user has permission to access the specified department.
func (s *KhataService) CheckDepartmentAccess(user *models.User, deptID string) error {
	if user == nil {
		return models.ErrUnauthorized
	}
	if user.Role == models.RoleGlobalAdmin || user.Role == "admin" {
		return nil
	}
	for _, id := range user.DepartmentIDs {
		if id == deptID {
			return nil
		}
	}
	return models.ErrUnauthorized
}

// GetUserByID looks up a user by ID.
func (s *KhataService) GetUserByID(id string) (*models.User, error) {
	return s.userRepo.GetUserByID(id)
}

// ListUsers returns all users in the system.
func (s *KhataService) ListUsers() ([]*models.User, error) {
	return s.userRepo.GetAllUsers()
}

// ListDepartments returns departments visible to the user based on RBAC permissions.
func (s *KhataService) ListDepartments(user *models.User) ([]*models.Department, error) {
	if user == nil {
		return nil, models.ErrUnauthorized
	}
	if user.Role == models.RoleGlobalAdmin || user.Role == "admin" {
		return s.nodeRepo.GetAllDepartments()
	}
	return s.nodeRepo.GetDepartmentsByIDs(user.DepartmentIDs)
}

// GetDepartmentTree builds the hierarchical tree for a department with rollup figures for each node.
func (s *KhataService) GetDepartmentTree(user *models.User, deptID string) (*models.Node, error) {
	if err := s.CheckDepartmentAccess(user, deptID); err != nil {
		return nil, err
	}

	nodes, err := s.nodeRepo.GetNodesByDepartment(deptID)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, models.ErrNotFound
	}

	nodeMap := make(map[string]*models.Node)
	var root *models.Node

	for _, n := range nodes {
		totals, err := s.analyticsRepo.GetRollUpTotals(n.ID, n.Path)
		if err == nil {
			n.RollUp = totals
		}
		n.Children = make([]*models.Node, 0)
		nodeMap[n.ID] = n
		if n.Depth == 1 {
			root = n
		}
	}

	for _, n := range nodes {
		if n.ParentID != nil {
			if parent, exists := nodeMap[*n.ParentID]; exists {
				parent.Children = append(parent.Children, n)
			}
		}
	}

	if root == nil {
		return nil, models.ErrNotFound
	}
	return root, nil
}

// GetNodeDetails returns the node, its ancestors path (breadcrumbs), and its financial rollup.
func (s *KhataService) GetNodeDetails(user *models.User, nodeID string) (*models.Node, []*models.Node, *models.RollUpTotals, error) {
	node, err := s.nodeRepo.GetNodeByID(nodeID)
	if err != nil {
		return nil, nil, nil, err
	}

	if err := s.CheckDepartmentAccess(user, node.DepartmentID); err != nil {
		return nil, nil, nil, err
	}

	ancestors, err := s.nodeRepo.GetAncestors(node.Path)
	if err != nil {
		return nil, nil, nil, err
	}

	totals, err := s.analyticsRepo.GetRollUpTotals(node.ID, node.Path)
	if err != nil {
		return nil, nil, nil, err
	}

	return node, ancestors, totals, nil
}

// CreateNode creates either a new Department (root depth 1) or a Group / Subgroup up to MAX_DEPTH = 5.
func (s *KhataService) CreateNode(user *models.User, req *models.CreateNodeRequest) (*models.Node, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name cannot be empty", models.ErrInvalidInput)
	}

	// Case 1: Creating a root Department (Depth = 1)
	if req.ParentID == nil || *req.ParentID == "" {
		// Only Global Admins can create top-level departments
		if user != nil && user.Role != models.RoleGlobalAdmin && user.Role != "admin" {
			return nil, models.ErrUnauthorized
		}

		deptID := "dept-" + uuid.New().String()[:8]
		dept := &models.Department{
			ID:   deptID,
			Name: name,
		}
		rootNode := &models.Node{
			ID:           deptID,
			DepartmentID: deptID,
			Name:         name,
			Path:         fmt.Sprintf("/%s/", deptID),
			Depth:        1,
		}

		if err := s.nodeRepo.CreateDepartment(dept, rootNode); err != nil {
			return nil, err
		}
		return rootNode, nil
	}

	// Case 2: Creating a Group / Subgroup under an existing parent node
	parent, err := s.nodeRepo.GetNodeByID(*req.ParentID)
	if err != nil {
		return nil, fmt.Errorf("parent node: %w", err)
	}

	if err := s.CheckDepartmentAccess(user, parent.DepartmentID); err != nil {
		return nil, err
	}

	// Enforce MAX_DEPTH = 5
	if parent.Depth >= models.MaxDepth {
		return nil, models.ErrMaxDepthExceeded
	}

	nodeID := "node-" + uuid.New().String()[:8]
	newNode := &models.Node{
		ID:           nodeID,
		DepartmentID: parent.DepartmentID,
		ParentID:     &parent.ID,
		Name:         name,
		Path:         fmt.Sprintf("%s%s/", parent.Path, nodeID),
		Depth:        parent.Depth + 1,
	}

	if err := s.nodeRepo.CreateNode(newNode); err != nil {
		return nil, err
	}
	return newNode, nil
}

// RenameNode updates the display name of a node.
func (s *KhataService) RenameNode(user *models.User, nodeID string, newName string) error {
	name := strings.TrimSpace(newName)
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", models.ErrInvalidInput)
	}

	node, err := s.nodeRepo.GetNodeByID(nodeID)
	if err != nil {
		return err
	}

	if err := s.CheckDepartmentAccess(user, node.DepartmentID); err != nil {
		return err
	}

	return s.nodeRepo.RenameNode(nodeID, name)
}

// MoveNode relocates a node and its entire subtree to a new parent with cycle detection and MAX_DEPTH validation.
func (s *KhataService) MoveNode(user *models.User, nodeID string, newParentID string) error {
	if nodeID == newParentID {
		return models.ErrCannotMoveToDescendant
	}

	node, err := s.nodeRepo.GetNodeByID(nodeID)
	if err != nil {
		return err
	}
	if node.Depth == 1 {
		return fmt.Errorf("%w: cannot move a top-level department", models.ErrInvalidInput)
	}

	newParent, err := s.nodeRepo.GetNodeByID(newParentID)
	if err != nil {
		return fmt.Errorf("target parent: %w", err)
	}

	// RBAC: User must have access to both origin department and target department
	if err := s.CheckDepartmentAccess(user, node.DepartmentID); err != nil {
		return err
	}
	if err := s.CheckDepartmentAccess(user, newParent.DepartmentID); err != nil {
		return err
	}

	// Cycle Prevention: target parent cannot be the node itself or any descendant of the node
	if s.nodeRepo.IsDescendant(node.Path, newParent.Path) {
		return models.ErrCannotMoveToDescendant
	}

	// Subtree Depth Limit: Calculate maximum depth among all descendants in the moved subtree
	maxSubtreeDepth, err := s.nodeRepo.GetSubtreeMaxDepth(node.Path)
	if err != nil {
		return err
	}

	subtreeHeight := maxSubtreeDepth - node.Depth
	newSubtreeMaxDepth := (newParent.Depth + 1) + subtreeHeight

	if newSubtreeMaxDepth > models.MaxDepth {
		return fmt.Errorf(
			"%w: target parent depth (%d) + subtree height (%d) = %d (limit is %d)",
			models.ErrMoveExceedsMaxDepth,
			newParent.Depth,
			subtreeHeight,
			newSubtreeMaxDepth,
			models.MaxDepth,
		)
	}

	// Atomic path, depth, and department update
	return s.nodeRepo.MoveNode(
		node.ID,
		node.Path,
		node.Depth,
		newParent.ID,
		newParent.DepartmentID,
		newParent.Path,
		newParent.Depth+1,
	)
}

// DeleteNode deletes a node and its entire subtree.
func (s *KhataService) DeleteNode(user *models.User, nodeID string) error {
	node, err := s.nodeRepo.GetNodeByID(nodeID)
	if err != nil {
		return err
	}

	if err := s.CheckDepartmentAccess(user, node.DepartmentID); err != nil {
		return err
	}

	return s.nodeRepo.DeleteNode(nodeID)
}

// CreateEntry validates and records a new financial entry in integer paise.
func (s *KhataService) CreateEntry(user *models.User, nodeID string, req *models.CreateEntryRequest) (*models.Entry, error) {
	node, err := s.nodeRepo.GetNodeByID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("target node: %w", err)
	}

	if err := s.CheckDepartmentAccess(user, node.DepartmentID); err != nil {
		return nil, err
	}

	entryType := strings.ToUpper(strings.TrimSpace(req.Type))
	if entryType != models.EntryTypeDebit && entryType != models.EntryTypeCredit {
		return nil, fmt.Errorf("%w: type must be 'CREDIT' or 'DEBIT'", models.ErrInvalidInput)
	}

	// Calculate integer paise
	var amountPaise int64
	if req.AmountPaise != nil && *req.AmountPaise > 0 {
		amountPaise = *req.AmountPaise
	} else if req.Amount > 0 {
		amountPaise = models.RupeesToPaise(req.Amount)
	}

	if amountPaise <= 0 {
		return nil, fmt.Errorf("%w: amount must be strictly greater than 0", models.ErrInvalidInput)
	}

	entryDate := strings.TrimSpace(req.EntryDate)
	if entryDate == "" {
		entryDate = time.Now().Format("2006-01-02")
	} else {
		if _, err := time.Parse("2006-01-02", entryDate); err != nil {
			return nil, fmt.Errorf("%w: invalid date format (expected YYYY-MM-DD)", models.ErrInvalidInput)
		}
	}

	desc := strings.TrimSpace(req.Description)
	if desc == "" {
		return nil, fmt.Errorf("%w: description is required", models.ErrInvalidInput)
	}

	entry := &models.Entry{
		ID:           "ent-" + uuid.New().String()[:8],
		NodeID:       node.ID,
		DepartmentID: node.DepartmentID,
		Type:         entryType,
		AmountPaise:  amountPaise,
		EntryDate:    entryDate,
		Description:  desc,
		Reference:    strings.TrimSpace(req.Reference),
	}

	if err := s.entryRepo.CreateEntry(entry); err != nil {
		return nil, err
	}

	entry.AmountRupees = models.PaiseToRupees(entry.AmountPaise)
	entry.Amount = entry.AmountRupees
	entry.FormattedAmount = models.FormatIndianRupees(entry.AmountPaise)
	entry.NodeName = node.Name
	return entry, nil
}

// ListEntries returns a paginated and filtered list of entries for a node.
func (s *KhataService) ListEntries(user *models.User, filter models.EntryFilter) (*models.EntryListResponse, error) {
	node, err := s.nodeRepo.GetNodeByID(filter.NodeID)
	if err != nil {
		return nil, err
	}

	if err := s.CheckDepartmentAccess(user, node.DepartmentID); err != nil {
		return nil, err
	}

	res, err := s.entryRepo.ListEntries(filter, node.Path)
	if err != nil {
		return nil, err
	}

	// Attach consolidated roll-up to response
	totals, err := s.analyticsRepo.GetRollUpTotals(node.ID, node.Path)
	if err == nil {
		res.RollUp = totals
	}

	return res, nil
}

// DeleteEntry deletes an entry.
func (s *KhataService) DeleteEntry(user *models.User, entryID string) error {
	entry, err := s.entryRepo.GetEntryByID(entryID)
	if err != nil {
		return err
	}

	if err := s.CheckDepartmentAccess(user, entry.DepartmentID); err != nil {
		return err
	}

	return s.entryRepo.DeleteEntry(entryID)
}

// BulkDeleteEntries deletes multiple entries.
func (s *KhataService) BulkDeleteEntries(user *models.User, entryIDs []string) error {
	if len(entryIDs) == 0 {
		return nil
	}
	// Verify user has access to each entry's department
	for _, id := range entryIDs {
		entry, err := s.entryRepo.GetEntryByID(id)
		if err != nil {
			return err
		}
		if err := s.CheckDepartmentAccess(user, entry.DepartmentID); err != nil {
			return err
		}
	}
	return s.entryRepo.BulkDeleteEntries(entryIDs)
}

// BulkMoveEntries reassigns entries to a target node.
func (s *KhataService) BulkMoveEntries(user *models.User, entryIDs []string, targetNodeID string) error {
	if len(entryIDs) == 0 {
		return nil
	}

	targetNode, err := s.nodeRepo.GetNodeByID(targetNodeID)
	if err != nil {
		return fmt.Errorf("target node: %w", err)
	}

	if err := s.CheckDepartmentAccess(user, targetNode.DepartmentID); err != nil {
		return err
	}

	for _, id := range entryIDs {
		entry, err := s.entryRepo.GetEntryByID(id)
		if err != nil {
			return err
		}
		if err := s.CheckDepartmentAccess(user, entry.DepartmentID); err != nil {
			return err
		}
	}

	return s.entryRepo.BulkMoveEntries(entryIDs, targetNode.ID, targetNode.DepartmentID)
}

// GetConsolidatedRollUp returns financial rollups for a node and all descendants using the materialized path.
func (s *KhataService) GetConsolidatedRollUp(user *models.User, nodeID string) (*models.RollUpTotals, error) {
	node, err := s.nodeRepo.GetNodeByID(nodeID)
	if err != nil {
		return nil, err
	}
	if err := s.CheckDepartmentAccess(user, node.DepartmentID); err != nil {
		return nil, err
	}
	return s.analyticsRepo.GetRollUpTotals(node.ID, node.Path)
}

// GetCashFlowTrends returns monthly Inflow vs Outflow time-series.
func (s *KhataService) GetCashFlowTrends(user *models.User, nodeID string) ([]*models.MonthlyTrend, error) {
	node, err := s.nodeRepo.GetNodeByID(nodeID)
	if err != nil {
		return nil, err
	}
	if err := s.CheckDepartmentAccess(user, node.DepartmentID); err != nil {
		return nil, err
	}
	return s.analyticsRepo.GetCashFlowTrends(node.Path)
}

// GetMacroOverview aggregates organization-wide financial statistics.
func (s *KhataService) GetMacroOverview(user *models.User) (*models.MacroOverview, error) {
	depts, err := s.ListDepartments(user)
	if err != nil {
		return nil, err
	}

	overview := &models.MacroOverview{
		TotalDepartments: len(depts),
		Departments:      make([]models.DepartmentOverview, 0, len(depts)),
	}

	for _, d := range depts {
		deptOverview, err := s.analyticsRepo.GetDepartmentOverview(*d)
		if err != nil {
			continue
		}
		overview.TotalNodes += deptOverview.NodesCount
		overview.TotalJamaPaise += deptOverview.RollUp.TotalJamaPaise
		overview.TotalUdharPaise += deptOverview.RollUp.TotalUdharPaise
		overview.Departments = append(overview.Departments, *deptOverview)
	}

	overview.NetBalancePaise = overview.TotalJamaPaise - overview.TotalUdharPaise
	overview.TotalJamaRupees = models.PaiseToRupees(overview.TotalJamaPaise)
	overview.TotalUdharRupees = models.PaiseToRupees(overview.TotalUdharPaise)
	overview.NetBalanceRupees = models.PaiseToRupees(overview.NetBalancePaise)
	overview.TotalCredits = overview.TotalJamaRupees
	overview.TotalDebits = overview.TotalUdharRupees
	overview.NetBalance = overview.NetBalanceRupees

	return overview, nil
}
