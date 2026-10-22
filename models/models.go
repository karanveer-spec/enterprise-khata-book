package models

import "time"

// Department represents a high-level organizational business unit.
type Department struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// User represents an authenticated identity with assigned department scopes.
type User struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Role          string   `json:"role"` // "Global Admin" or "Department Viewer"
	DepartmentIDs []string `json:"department_ids"`
}

// Node represents a node in the 5-level hierarchy tree (Department -> Group -> Subgroup -> ...).
// Uses a hybrid Materialized Path + Adjacency List pattern.
type Node struct {
	ID           string        `json:"id"`
	DepartmentID string        `json:"department_id"`
	ParentID     *string       `json:"parent_id"`
	Name         string        `json:"name"`
	Path         string        `json:"path"`  // Materialized Path e.g. "/dept-fin/group-1/sub-2/"
	Depth        int           `json:"depth"` // 1 to 5
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Children     []*Node       `json:"children,omitempty"`
	RollUp       *RollUpTotals `json:"rollup,omitempty"`
}

// RollUpTotals represents direct and descendant aggregated financial figures in integer paise & rupees.
type RollUpTotals struct {
	// Direct totals recorded strictly on this node (Integer Paise)
	DirectJamaPaise    int64   `json:"direct_jama_paise"`
	DirectUdharPaise   int64   `json:"direct_udhar_paise"`
	DirectBalancePaise int64   `json:"direct_balance_paise"` // DirectJama - DirectUdhar

	// Direct totals in float Rupees
	DirectJamaRupees    float64 `json:"direct_jama_rupees"`
	DirectUdharRupees   float64 `json:"direct_udhar_rupees"`
	DirectBalanceRupees float64 `json:"direct_balance_rupees"`
	DirectCredits       float64 `json:"direct_credits"`
	DirectDebits        float64 `json:"direct_debits"`
	DirectBalance       float64 `json:"direct_balance"`
	DirectCount         int     `json:"direct_count"`

	// Roll-up totals aggregated across this node and all descendants (Integer Paise)
	TotalJamaPaise    int64 `json:"total_jama_paise"`
	TotalUdharPaise   int64 `json:"total_udhar_paise"`
	TotalBalancePaise int64 `json:"total_balance_paise"` // TotalJama - TotalUdhar

	// Roll-up totals in float Rupees
	TotalJamaRupees    float64 `json:"total_jama_rupees"`
	TotalUdharRupees   float64 `json:"total_udhar_rupees"`
	TotalBalanceRupees float64 `json:"total_balance_rupees"`
	TotalCredits       float64 `json:"total_credits"`
	TotalDebits        float64 `json:"total_debits"`
	TotalBalance       float64 `json:"total_balance"`
	TotalCount         int     `json:"total_count"`
}

// Entry represents a financial transaction row (Jama/Credit or Udhar/Debit).
// Financial amounts are strictly stored as integer paise to guarantee zero precision loss.
type Entry struct {
	ID              string    `json:"id"`
	NodeID          string    `json:"node_id"`
	DepartmentID    string    `json:"department_id"`
	Type            string    `json:"type"` // "CREDIT" (Jama) or "DEBIT" (Udhar)
	AmountPaise     int64     `json:"amount_paise"`
	AmountRupees    float64   `json:"amount_rupees"`
	Amount          float64   `json:"amount"` // Rupee amount alias for UI
	FormattedAmount string    `json:"formatted_amount"` // e.g. ₹ 1,50,000.00
	EntryDate       string    `json:"entry_date"`       // YYYY-MM-DD
	Description     string    `json:"description"`
	Reference       string    `json:"reference"`
	NodeName        string    `json:"node_name,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// MonthlyTrend represents time-series financial inflow vs outflow for a given month.
type MonthlyTrend struct {
	Month       string  `json:"month"` // YYYY-MM
	JamaPaise   int64   `json:"jama_paise"`
	UdharPaise  int64   `json:"udhar_paise"`
	NetPaise    int64   `json:"net_paise"`
	JamaRupees  float64 `json:"jama_rupees"`
	UdharRupees float64 `json:"udhar_rupees"`
	NetRupees   float64 `json:"net_rupees"`
	Credits     float64 `json:"credits"`
	Debits      float64 `json:"debits"`
	EntryCount  int     `json:"entry_count"`
}

// DepartmentOverview provides macro organizational metrics.
type DepartmentOverview struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Department Department    `json:"department"`
	RootNodeID string        `json:"root_node_id"`
	NodesCount int           `json:"nodes_count"`
	RollUp     *RollUpTotals `json:"rollup"`
}

// MacroOverview encapsulates organization-wide statistics.
type MacroOverview struct {
	TotalDepartments int                  `json:"total_departments"`
	TotalNodes       int                  `json:"total_nodes"`
	TotalJamaPaise   int64                `json:"total_jama_paise"`
	TotalUdharPaise  int64                `json:"total_udhar_paise"`
	NetBalancePaise  int64                `json:"net_balance_paise"`
	TotalJamaRupees  float64              `json:"total_jama_rupees"`
	TotalUdharRupees float64              `json:"total_udhar_rupees"`
	NetBalanceRupees float64              `json:"net_balance_rupees"`
	TotalCredits     float64              `json:"total_credits"`
	TotalDebits      float64              `json:"total_debits"`
	NetBalance       float64              `json:"net_balance"`
	Departments      []DepartmentOverview `json:"departments"`
}

// EntryFilter contains parameters for querying entries with SQL-level filtering & pagination.
type EntryFilter struct {
	NodeID             string
	IncludeDescendants bool
	Search             string
	Period             string // "all", "today", "this_month", "this_quarter", "custom"
	FromDate           string // YYYY-MM-DD
	ToDate             string // YYYY-MM-DD
	Type               string // "CREDIT", "DEBIT", or ""
	SortBy             string // "date", "amount", "created_at"
	SortDir            string // "asc", "desc"
	Page               int
	Limit              int
}

// EntryListResponse represents paginated entries.
type EntryListResponse struct {
	Items      []*Entry      `json:"items"`
	TotalCount int           `json:"total_count"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalPages int           `json:"total_pages"`
	RollUp     *RollUpTotals `json:"rollup,omitempty"`
}

// Request Payloads

type CreateNodeRequest struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"` // null when creating a top-level department node
}

type RenameNodeRequest struct {
	Name string `json:"name"`
}

type MoveNodeRequest struct {
	NewParentID string `json:"new_parent_id"`
}

type CreateEntryRequest struct {
	Type        string  `json:"type"` // "CREDIT" or "DEBIT"
	Amount      float64 `json:"amount"` // in Rupees from UI, converted to Paise internally
	AmountPaise *int64  `json:"amount_paise,omitempty"`
	EntryDate   string  `json:"entry_date"`
	Description string  `json:"description"`
	Reference   string  `json:"reference"`
}

type BulkDeleteRequest struct {
	EntryIDs []string `json:"entry_ids"`
}

type BulkMoveRequest struct {
	EntryIDs     []string `json:"entry_ids"`
	TargetNodeID string   `json:"target_node_id"`
}
