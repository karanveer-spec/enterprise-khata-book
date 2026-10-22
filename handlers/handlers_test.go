package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"khata-book/handlers"
	"khata-book/models"
	"khata-book/repository"
	"khata-book/services"
	_ "modernc.org/sqlite"
)

func setupTestApp(t *testing.T) (*sql.DB, *repository.WriteQueue, *services.KhataService, http.Handler) {
	dbPath := filepath.Join(t.TempDir(), "test_api.db")
	db, err := repository.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	wq := repository.NewWriteQueue(db, 64)
	nodeRepo := repository.NewNodeRepository(db, wq)
	entryRepo := repository.NewEntryRepository(db, wq)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	userRepo := repository.NewUserRepository(db)

	svc := services.NewKhataService(nodeRepo, entryRepo, analyticsRepo, userRepo)
	router := handlers.NewRouter(svc)
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	app := handlers.AuthMiddleware(svc)(mux)
	return db, wq, svc, app
}

// Test HTTP Endpoints: Create Node, MaxDepth Violation, Record Entry, Pagination & CSV Export
func TestAPI_Workflow(t *testing.T) {
	db, wq, _, app := setupTestApp(t)
	defer db.Close()
	defer wq.Close()

	// 1. Create a Department (Depth 1)
	createDeptBody := `{"name": "Supply Chain & Logistics"}`
	req := httptest.NewRequest("POST", "/api/nodes", bytes.NewBufferString(createDeptBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin") // Global Admin
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for department, got %d: %s", rec.Code, rec.Body.String())
	}

	var deptNode models.Node
	if err := json.NewDecoder(rec.Body).Decode(&deptNode); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if deptNode.Depth != 1 {
		t.Fatalf("expected depth 1, got %d", deptNode.Depth)
	}

	// 2. Add Group (Depth 2)
	createGroupBody := `{"name": "Warehouse Depot North", "parent_id": "` + deptNode.ID + `"}`
	req = httptest.NewRequest("POST", "/api/nodes", bytes.NewBufferString(createGroupBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for group, got %d: %s", rec.Code, rec.Body.String())
	}
	var groupNode models.Node
	_ = json.NewDecoder(rec.Body).Decode(&groupNode)
	if groupNode.Depth != 2 {
		t.Fatalf("expected depth 2, got %d", groupNode.Depth)
	}

	// 3. Record Khata Entry (CREDIT ₹ 75,000.00)
	createEntryBody := `{
		"type": "CREDIT",
		"amount": 75000.00,
		"entry_date": "2026-10-22",
		"description": "Advance delivery retainer received",
		"reference": "VOUCH-ADV-01"
	}`
	req = httptest.NewRequest("POST", "/api/nodes/"+groupNode.ID+"/entries", bytes.NewBufferString(createEntryBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for entry, got %d: %s", rec.Code, rec.Body.String())
	}
	var entry models.Entry
	_ = json.NewDecoder(rec.Body).Decode(&entry)
	if entry.AmountPaise != 7500000 {
		t.Fatalf("expected 7,500,000 paise, got %d", entry.AmountPaise)
	}

	// 4. Test SQL Pagination & Search Endpoint
	req = httptest.NewRequest("GET", "/api/nodes/"+deptNode.ID+"/entries?include_descendants=true&page=1&limit=10&q=retainer", nil)
	req.Header.Set("X-User-Id", "admin")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for entry list, got %d: %s", rec.Code, rec.Body.String())
	}
	var listResp models.EntryListResponse
	_ = json.NewDecoder(rec.Body).Decode(&listResp)
	if listResp.TotalCount != 1 {
		t.Fatalf("expected 1 search result, got %d", listResp.TotalCount)
	}

	// 5. Test Streaming CSV Export Endpoint
	req = httptest.NewRequest("GET", "/api/nodes/"+deptNode.ID+"/export", nil)
	req.Header.Set("X-User-Id", "admin")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for CSV export, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Fatalf("expected text/csv Content-Type, got: %s", contentType)
	}
	bodyStr := rec.Body.String()
	if !strings.Contains(bodyStr, "Transaction ID") || !strings.Contains(bodyStr, "Advance delivery retainer received") {
		t.Fatalf("CSV content missing expected headers or transaction row: %s", bodyStr)
	}
}
