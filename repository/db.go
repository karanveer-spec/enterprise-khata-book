package repository

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// OpenDB opens a SQLite database connection with WAL mode and serialized write optimizations.
func OpenDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// SQLite connection pool: allow multiple concurrent readers in WAL mode
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	// Set recommended SQLite pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 10000;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA cache_size = -64000;", // 64MB cache
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to execute pragma %s: %w", pragma, err)
		}
	}

	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// Migrate creates the required schema and initial seed data if not present.
func Migrate(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS departments (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			role TEXT NOT NULL CHECK(role IN ('Global Admin', 'Department Viewer', 'admin', 'department_user'))
		);`,

		`CREATE TABLE IF NOT EXISTS user_departments (
			user_id TEXT NOT NULL,
			department_id TEXT NOT NULL,
			PRIMARY KEY (user_id, department_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS ledger_nodes (
			id TEXT PRIMARY KEY,
			department_id TEXT NOT NULL,
			parent_id TEXT,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			depth INTEGER NOT NULL CHECK(depth >= 1 AND depth <= 5),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE CASCADE,
			FOREIGN KEY (parent_id) REFERENCES ledger_nodes(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS entries (
			id TEXT PRIMARY KEY,
			node_id TEXT NOT NULL,
			department_id TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('CREDIT', 'DEBIT')),
			amount_paise INTEGER NOT NULL CHECK(amount_paise > 0),
			entry_date TEXT NOT NULL,
			description TEXT NOT NULL,
			reference TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (node_id) REFERENCES ledger_nodes(id) ON DELETE CASCADE,
			FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE CASCADE
		);`,

		// Materialized Path & Hierarchy Indexes
		`CREATE INDEX IF NOT EXISTS idx_ledger_nodes_path ON ledger_nodes(path);`,
		`CREATE INDEX IF NOT EXISTS idx_ledger_nodes_parent ON ledger_nodes(parent_id);`,
		`CREATE INDEX IF NOT EXISTS idx_ledger_nodes_dept ON ledger_nodes(department_id);`,
		`CREATE INDEX IF NOT EXISTS idx_ledger_nodes_depth ON ledger_nodes(depth);`,

		// Entry Search, Filtering & Pagination Indexes
		`CREATE INDEX IF NOT EXISTS idx_entries_node ON entries(node_id);`,
		`CREATE INDEX IF NOT EXISTS idx_entries_dept ON entries(department_id);`,
		`CREATE INDEX IF NOT EXISTS idx_entries_date ON entries(entry_date);`,
		`CREATE INDEX IF NOT EXISTS idx_entries_created ON entries(created_at);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("migration failed executing query %s: %w", query, err)
		}
	}

	return seedInitialData(db)
}

func seedInitialData(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM departments").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil // already seeded
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Seed Users with RBAC roles
	users := []struct {
		id, name, role string
	}{
		{"admin", "Alice Sharma (Global Admin)", "Global Admin"},
		{"finance_user", "Bob Verma (Finance Manager)", "Department Viewer"},
		{"ops_user", "Charlie Patel (Operations Lead)", "Department Viewer"},
		{"sales_user", "Diana Rao (Sales & Marketing Lead)", "Department Viewer"},
	}
	for _, u := range users {
		_, err := tx.Exec("INSERT INTO users (id, name, role) VALUES (?, ?, ?)", u.id, u.name, u.role)
		if err != nil {
			return fmt.Errorf("seed users failed: %w", err)
		}
	}

	// 2. Seed Departments
	depts := []struct {
		id, name string
	}{
		{"dept-fin", "Finance & Accounts"},
		{"dept-ops", "Operations & Logistics"},
		{"dept-sales", "Sales & Distribution"},
	}
	for _, d := range depts {
		_, err := tx.Exec("INSERT INTO departments (id, name) VALUES (?, ?)", d.id, d.name)
		if err != nil {
			return fmt.Errorf("seed departments failed: %w", err)
		}
	}

	// 3. Seed Department Scopes for Users
	userDepts := []struct {
		userID, deptID string
	}{
		{"finance_user", "dept-fin"},
		{"ops_user", "dept-ops"},
		{"sales_user", "dept-sales"},
	}
	for _, ud := range userDepts {
		_, err := tx.Exec("INSERT INTO user_departments (user_id, department_id) VALUES (?, ?)", ud.userID, ud.deptID)
		if err != nil {
			return fmt.Errorf("seed user_departments failed: %w", err)
		}
	}

	// 4. Seed Hierarchical Ledger Nodes (Materialized Path + Adjacency List)
	// Depth 1: Root Department Nodes (path = "/<dept-id>/")
	// Depth 2: Groups (path = "/<dept-id>/<group-id>/")
	// Depth 3..5: Subgroups up to 5 levels
	nodes := []struct {
		id, deptID string
		parentID   *string
		name       string
		path       string
		depth      int
	}{
		// Dept 1: Finance (Full 5 levels tree)
		{"dept-fin", "dept-fin", nil, "Finance & Accounts", "/dept-fin/", 1},
		{"fin-g1", "dept-fin", strPtr("dept-fin"), "Operational Expenses (OpEx)", "/dept-fin/fin-g1/", 2},
		{"fin-g1-s1", "dept-fin", strPtr("fin-g1"), "Office Rent & Utilities", "/dept-fin/fin-g1/fin-g1-s1/", 3},
		{"fin-g1-s1-1", "dept-fin", strPtr("fin-g1-s1"), "Electricity & Water Bills", "/dept-fin/fin-g1/fin-g1-s1/fin-g1-s1-1/", 4},
		{"fin-g1-s1-1-leaf", "dept-fin", strPtr("fin-g1-s1-1"), "Meter 04 - Main Warehouse", "/dept-fin/fin-g1/fin-g1-s1/fin-g1-s1-1/fin-g1-s1-1-leaf/", 5},
		{"fin-g2", "dept-fin", strPtr("dept-fin"), "Capital Expenditure (CapEx)", "/dept-fin/fin-g2/", 2},
		{"fin-g2-s1", "dept-fin", strPtr("fin-g2"), "IT Infrastructure & Cloud", "/dept-fin/fin-g2/fin-g2-s1/", 3},
		{"fin-g3", "dept-fin", strPtr("dept-fin"), "Revenue & Client Receipts", "/dept-fin/fin-g3/", 2},

		// Dept 2: Operations
		{"dept-ops", "dept-ops", nil, "Operations & Logistics", "/dept-ops/", 1},
		{"ops-g1", "dept-ops", strPtr("dept-ops"), "Fleet Fuel & Maintenance", "/dept-ops/ops-g1/", 2},
		{"ops-g1-s1", "dept-ops", strPtr("ops-g1"), "Diesel Purchases - North Corridor", "/dept-ops/ops-g1/ops-g1-s1/", 3},
		{"ops-g2", "dept-ops", strPtr("dept-ops"), "Warehouse Packaging & Supplies", "/dept-ops/ops-g2/", 2},

		// Dept 3: Sales
		{"dept-sales", "dept-sales", nil, "Sales & Distribution", "/dept-sales/", 1},
		{"sales-g1", "dept-sales", strPtr("dept-sales"), "Domestic Enterprise Accounts", "/dept-sales/sales-g1/", 2},
		{"sales-g2", "dept-sales", strPtr("dept-sales"), "Digital Marketing & Ad Spend", "/dept-sales/sales-g2/", 2},
	}

	for _, n := range nodes {
		_, err := tx.Exec(`
			INSERT INTO ledger_nodes (id, department_id, parent_id, name, path, depth)
			VALUES (?, ?, ?, ?, ?, ?)
		`, n.id, n.deptID, n.parentID, n.name, n.path, n.depth)
		if err != nil {
			return fmt.Errorf("seed ledger_nodes failed for %s: %w", n.id, err)
		}
	}

	// 5. Seed Initial Entries with integer paise amounts (1 Rupee = 100 Paise)
	entries := []struct {
		id, nodeID, deptID, entryType string
		amountPaise                   int64
		entryDate, description, ref   string
	}{
		// Revenue (Jama / CREDIT)
		{"ent-001", "fin-g3", "dept-fin", "CREDIT", 25000000, "2026-10-15", "Q3 Retainer Inflow - Mumbai Enterprise Client", "INV-2026-101"},
		{"ent-002", "fin-g3", "dept-fin", "CREDIT", 12500000, "2026-10-18", "Annual Enterprise SaaS License Renewal", "INV-2026-102"},
		{"ent-003", "sales-g1", "dept-sales", "CREDIT", 8500000, "2026-10-20", "Advance deposit for Delhi Retail Chain contract", "DEP-DEL-99"},

		// OpEx / Expenses (Udhar / DEBIT)
		{"ent-004", "fin-g1-s1-1-leaf", "dept-fin", "DEBIT", 1850000, "2026-10-12", "Monthly Electricity Bill - Warehouse Meter 04", "VOUCH-ELEC-44"},
		{"ent-005", "fin-g1-s1", "dept-fin", "DEBIT", 9500000, "2026-10-01", "Commercial Office Lease Rent - October 2026", "RENT-OCT-2026"},
		{"ent-006", "fin-g2-s1", "dept-fin", "DEBIT", 3500000, "2026-10-10", "Production Cloud Server Hosting & Backup Nodes", "AWS-INV-7721"},
		{"ent-007", "ops-g1-s1", "dept-ops", "DEBIT", 2850000, "2026-10-14", "Bulk Diesel Tanker Refill - Transport Depot A", "IOCL-BILL-912"},
		{"ent-008", "sales-g2", "dept-sales", "DEBIT", 2200000, "2026-10-08", "Social Media & Search Engine Campaign Retargeting", "META-ADS-334"},
	}

	for _, e := range entries {
		_, err := tx.Exec(`
			INSERT INTO entries (id, node_id, department_id, type, amount_paise, entry_date, description, reference)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, e.id, e.nodeID, e.deptID, e.entryType, e.amountPaise, e.entryDate, e.description, e.ref)
		if err != nil {
			return fmt.Errorf("seed entries failed for %s: %w", e.id, err)
		}
	}

	return tx.Commit()
}

func strPtr(s string) *string {
	return &s
}
