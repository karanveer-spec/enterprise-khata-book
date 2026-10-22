package services_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"khata-book/models"
	"khata-book/repository"
	"khata-book/services"
	_ "modernc.org/sqlite"
)

func setupTestService(t *testing.T) (*sql.DB, *repository.WriteQueue, *services.KhataService) {
	dbPath := filepath.Join(t.TempDir(), "test_khata.db")
	db, err := repository.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open isolated test db: %v", err)
	}

	wq := repository.NewWriteQueue(db, 64)
	nodeRepo := repository.NewNodeRepository(db, wq)
	entryRepo := repository.NewEntryRepository(db, wq)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	userRepo := repository.NewUserRepository(db)

	svc := services.NewKhataService(nodeRepo, entryRepo, analyticsRepo, userRepo)
	return db, wq, svc
}

// Test 1: Depth Limit Enforcement (Up to 5 levels, level 6 is strictly rejected)
func TestMaxDepthLimit_Enforcement(t *testing.T) {
	db, wq, svc := setupTestService(t)
	defer db.Close()
	defer wq.Close()

	adminUser := &models.User{ID: "admin", Role: models.RoleGlobalAdmin}

	// 1. Create Level 1 Department
	dept, err := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Engineering & Tech"})
	if err != nil {
		t.Fatalf("failed to create level 1 department: %v", err)
	}
	if dept.Depth != 1 {
		t.Fatalf("expected depth 1, got %d", dept.Depth)
	}

	// 2. Create Level 2 Group
	lvl2, err := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Core Platform", ParentID: &dept.ID})
	if err != nil {
		t.Fatalf("failed to create level 2 node: %v", err)
	}
	if lvl2.Depth != 2 {
		t.Fatalf("expected depth 2, got %d", lvl2.Depth)
	}

	// 3. Create Level 3 Subgroup
	lvl3, err := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Backend Services", ParentID: &lvl2.ID})
	if err != nil {
		t.Fatalf("failed to create level 3 node: %v", err)
	}
	if lvl3.Depth != 3 {
		t.Fatalf("expected depth 3, got %d", lvl3.Depth)
	}

	// 4. Create Level 4 Sub-subgroup
	lvl4, err := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Database Cluster", ParentID: &lvl3.ID})
	if err != nil {
		t.Fatalf("failed to create level 4 node: %v", err)
	}
	if lvl4.Depth != 4 {
		t.Fatalf("expected depth 4, got %d", lvl4.Depth)
	}

	// 5. Create Level 5 Leaf
	lvl5, err := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Sharding Engine", ParentID: &lvl4.ID})
	if err != nil {
		t.Fatalf("failed to create level 5 node: %v", err)
	}
	if lvl5.Depth != 5 {
		t.Fatalf("expected depth 5, got %d", lvl5.Depth)
	}

	// 6. Attempt to create Level 6 - MUST FAIL with ErrMaxDepthExceeded
	_, err = svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Exceeds Maximum Depth", ParentID: &lvl5.ID})
	if err == nil {
		t.Fatalf("expected error when creating node at depth 6, but got success")
	}
	if !errors.Is(err, models.ErrMaxDepthExceeded) {
		t.Fatalf("expected ErrMaxDepthExceeded, got: %v", err)
	}
}

// Test 2: Subtree Move Validation (Cycle Detection & Depth Limit Exceeded)
func TestMoveNode_Validations(t *testing.T) {
	db, wq, svc := setupTestService(t)
	defer db.Close()
	defer wq.Close()

	adminUser := &models.User{ID: "admin", Role: models.RoleGlobalAdmin}

	// Create Hierarchy:
	// Dept 1: L1
	//   -> L2A
	//       -> L3A
	//           -> L4A
	//   -> L2B
	//       -> L3B
	//           -> L4B
	//               -> L5B
	dept, _ := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Logistics Hub"})
	l2A, _ := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "North Zone", ParentID: &dept.ID})
	l3A, _ := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Fleet Maintenance", ParentID: &l2A.ID})
	l4A, _ := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Heavy Trucks", ParentID: &l3A.ID})

	l2B, _ := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "South Zone", ParentID: &dept.ID})
	l3B, _ := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Depot Operations", ParentID: &l2B.ID})
	l4B, _ := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Warehouse 1", ParentID: &l3B.ID})
	_, _ = svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Storage Bay 4", ParentID: &l4B.ID})

	// 1. Cycle Prevention: Moving l2A into its own descendant l4A must fail
	err := svc.MoveNode(adminUser, l2A.ID, l4A.ID)
	if err == nil || !errors.Is(err, models.ErrCannotMoveToDescendant) {
		t.Fatalf("expected ErrCannotMoveToDescendant, got: %v", err)
	}

	// 2. Self Move: Moving l2A into itself must fail
	err = svc.MoveNode(adminUser, l2A.ID, l2A.ID)
	if err == nil || !errors.Is(err, models.ErrCannotMoveToDescendant) {
		t.Fatalf("expected ErrCannotMoveToDescendant, got: %v", err)
	}

	// 3. Subtree Depth Overflow:
	// l3A has subtree height = 1 (depth 3 to depth 4).
	// If moved to l4B (which is at depth 4), new depth for l3A becomes 5, and its child l4A becomes 6!
	// 4 + 1 + 1 = 6 > MAX_DEPTH(5). MUST FAIL with ErrMoveExceedsMaxDepth.
	err = svc.MoveNode(adminUser, l3A.ID, l4B.ID)
	if err == nil || !errors.Is(err, models.ErrMoveExceedsMaxDepth) {
		t.Fatalf("expected ErrMoveExceedsMaxDepth, got: %v", err)
	}

	// 4. Valid Move: Move l3A to l2B (depth 2). New depth for l3A becomes 3, child l4A becomes 4 <= 5.
	err = svc.MoveNode(adminUser, l3A.ID, l2B.ID)
	if err != nil {
		t.Fatalf("expected valid move to succeed, got: %v", err)
	}

	// Verify updated depth and path for moved node and descendant
	nodeDetails, _, _, err := svc.GetNodeDetails(adminUser, l3A.ID)
	if err != nil {
		t.Fatalf("failed to get node details: %v", err)
	}
	if nodeDetails.Depth != 3 {
		t.Fatalf("expected depth 3 after move, got %d", nodeDetails.Depth)
	}

	childDetails, _, _, err := svc.GetNodeDetails(adminUser, l4A.ID)
	if err != nil {
		t.Fatalf("failed to get child details: %v", err)
	}
	if childDetails.Depth != 4 {
		t.Fatalf("expected child depth 4 after move, got %d", childDetails.Depth)
	}
}

// Test 3: Roll-up Totals Correctness across all levels using Materialized Path
func TestRollUpTotals_Correctness(t *testing.T) {
	db, wq, svc := setupTestService(t)
	defer db.Close()
	defer wq.Close()

	adminUser := &models.User{ID: "admin", Role: models.RoleGlobalAdmin}

	dept, _ := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Commercial Operations"})
	g1, _ := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Regional Sales", ParentID: &dept.ID})
	s1, _ := svc.CreateNode(adminUser, &models.CreateNodeRequest{Name: "Retail Outlets", ParentID: &g1.ID})

	// Add entries at different depths in Paise:
	// 1. Entry at Subgroup s1 (depth 3): CREDIT ₹ 50,000.00 (5,000,000 paise)
	_, err := svc.CreateEntry(adminUser, s1.ID, &models.CreateEntryRequest{
		Type:        models.EntryTypeCredit,
		Amount:      50000.00,
		Description: "Store Daily Cash Deposit",
	})
	if err != nil {
		t.Fatalf("failed to create entry at subgroup: %v", err)
	}

	// 2. Entry at Group g1 (depth 2): DEBIT ₹ 15,000.00 (1,500,000 paise)
	_, err = svc.CreateEntry(adminUser, g1.ID, &models.CreateEntryRequest{
		Type:        models.EntryTypeDebit,
		Amount:      15000.00,
		Description: "Regional Office Supplies",
	})
	if err != nil {
		t.Fatalf("failed to create entry at group: %v", err)
	}

	// 3. Entry directly at Department Root (depth 1): DEBIT ₹ 10,000.00 (1,000,000 paise)
	_, err = svc.CreateEntry(adminUser, dept.ID, &models.CreateEntryRequest{
		Type:        models.EntryTypeDebit,
		Amount:      10000.00,
		Description: "Department Audit Fees",
	})
	if err != nil {
		t.Fatalf("failed to create entry at root: %v", err)
	}

	// Verify Department Root Rollup:
	// Direct: Jama = 0, Udhar = 10,000 (1,000,000 paise), Balance = -10,000
	// Consolidated Rollup across all descendants:
	// Total Jama = 50,000 (5,000,000 paise)
	// Total Udhar = 15,000 + 10,000 = 25,000 (2,500,000 paise)
	// Total Balance = 25,000 (2,500,000 paise)
	_, _, deptTotals, err := svc.GetNodeDetails(adminUser, dept.ID)
	if err != nil {
		t.Fatalf("failed to get department details: %v", err)
	}

	if deptTotals.DirectUdharPaise != 1000000 {
		t.Fatalf("expected direct udhar 1,000,000 paise, got %d", deptTotals.DirectUdharPaise)
	}
	if deptTotals.TotalJamaPaise != 5000000 {
		t.Fatalf("expected total rollup jama 5,000,000 paise, got %d", deptTotals.TotalJamaPaise)
	}
	if deptTotals.TotalUdharPaise != 2500000 {
		t.Fatalf("expected total rollup udhar 2,500,000 paise, got %d", deptTotals.TotalUdharPaise)
	}
	if deptTotals.TotalBalancePaise != 2500000 {
		t.Fatalf("expected total rollup balance 2,500,000 paise, got %d", deptTotals.TotalBalancePaise)
	}
	if deptTotals.TotalCount != 3 {
		t.Fatalf("expected total count 3, got %d", deptTotals.TotalCount)
	}
}

// Test 4: Department-Scoped Access (RBAC)
func TestDepartmentScopedAccess(t *testing.T) {
	db, wq, svc := setupTestService(t)
	defer db.Close()
	defer wq.Close()

	// Seed user with access strictly to dept-fin
	financeUser := &models.User{
		ID:            "finance_user",
		Role:          models.RoleDepartmentViewer,
		DepartmentIDs: []string{"dept-fin"},
	}

	// 1. Finance user should have access to dept-fin
	err := svc.CheckDepartmentAccess(financeUser, "dept-fin")
	if err != nil {
		t.Fatalf("expected access to assigned department dept-fin, got error: %v", err)
	}

	// 2. Finance user MUST NOT have access to dept-ops
	err = svc.CheckDepartmentAccess(financeUser, "dept-ops")
	if err == nil || !errors.Is(err, models.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for unassigned department dept-ops, got: %v", err)
	}
}
