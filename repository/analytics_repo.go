package repository

import (
	"database/sql"
	"fmt"

	"khata-book/models"
)

type AnalyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// GetRollUpTotals calculates direct totals and consolidated roll-up totals across all descendants using the materialized path.
func (r *AnalyticsRepository) GetRollUpTotals(nodeID, nodePath string) (*models.RollUpTotals, error) {
	// 1. Direct totals recorded strictly on this node
	var directJama, directUdhar int64
	var directCount int
	err := r.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN type = 'CREDIT' THEN amount_paise ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'DEBIT' THEN amount_paise ELSE 0 END), 0),
			COUNT(id)
		FROM entries
		WHERE node_id = ?;
	`, nodeID).Scan(&directJama, &directUdhar, &directCount)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate direct totals: %w", err)
	}

	// 2. Roll-up totals across this node and all its descendants via indexed path
	var totalJama, totalUdhar int64
	var totalCount int
	err = r.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN e.type = 'CREDIT' THEN e.amount_paise ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN e.type = 'DEBIT' THEN e.amount_paise ELSE 0 END), 0),
			COUNT(e.id)
		FROM entries e
		WHERE e.node_id IN (SELECT id FROM ledger_nodes WHERE path LIKE ? || '%');
	`, nodePath).Scan(&totalJama, &totalUdhar, &totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate rollup totals: %w", err)
	}

	directBal := directJama - directUdhar
	totalBal := totalJama - totalUdhar

	directJamaRupees := models.PaiseToRupees(directJama)
	directUdharRupees := models.PaiseToRupees(directUdhar)
	directBalRupees := models.PaiseToRupees(directBal)

	totalJamaRupees := models.PaiseToRupees(totalJama)
	totalUdharRupees := models.PaiseToRupees(totalUdhar)
	totalBalRupees := models.PaiseToRupees(totalBal)

	return &models.RollUpTotals{
		DirectJamaPaise:     directJama,
		DirectUdharPaise:    directUdhar,
		DirectBalancePaise:  directBal,
		DirectJamaRupees:    directJamaRupees,
		DirectUdharRupees:   directUdharRupees,
		DirectBalanceRupees: directBalRupees,
		DirectCredits:       directJamaRupees,
		DirectDebits:        directUdharRupees,
		DirectBalance:       directBalRupees,
		DirectCount:         directCount,

		TotalJamaPaise:     totalJama,
		TotalUdharPaise:    totalUdhar,
		TotalBalancePaise:  totalBal,
		TotalJamaRupees:    totalJamaRupees,
		TotalUdharRupees:   totalUdharRupees,
		TotalBalanceRupees: totalBalRupees,
		TotalCredits:       totalJamaRupees,
		TotalDebits:        totalUdharRupees,
		TotalBalance:       totalBalRupees,
		TotalCount:         totalCount,
	}, nil
}

// GetCashFlowTrends calculates monthly time-series aggregating Jama vs Udhar for sparklines / charts.
func (r *AnalyticsRepository) GetCashFlowTrends(nodePath string) ([]*models.MonthlyTrend, error) {
	query := `
		SELECT
			strftime('%Y-%m', e.entry_date) AS mth,
			COALESCE(SUM(CASE WHEN e.type = 'CREDIT' THEN e.amount_paise ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN e.type = 'DEBIT' THEN e.amount_paise ELSE 0 END), 0),
			COUNT(e.id)
		FROM entries e
		WHERE e.node_id IN (SELECT id FROM ledger_nodes WHERE path LIKE ? || '%')
		GROUP BY mth
		ORDER BY mth ASC
		LIMIT 12;
	`
	rows, err := r.db.Query(query, nodePath)
	if err != nil {
		return nil, fmt.Errorf("failed to query cash flow trends: %w", err)
	}
	defer rows.Close()

	var trends []*models.MonthlyTrend
	for rows.Next() {
		var t models.MonthlyTrend
		if err := rows.Scan(&t.Month, &t.JamaPaise, &t.UdharPaise, &t.EntryCount); err != nil {
			return nil, err
		}
		t.NetPaise = t.JamaPaise - t.UdharPaise
		t.JamaRupees = models.PaiseToRupees(t.JamaPaise)
		t.UdharRupees = models.PaiseToRupees(t.UdharPaise)
		t.NetRupees = models.PaiseToRupees(t.NetPaise)
		trends = append(trends, &t)
	}
	return trends, nil
}

// GetDepartmentOverview calculates macro totals for a specific department.
func (r *AnalyticsRepository) GetDepartmentOverview(dept models.Department) (*models.DepartmentOverview, error) {
	// Look up root node of department
	var rootNodeID, rootNodePath string
	err := r.db.QueryRow(`
		SELECT id, path FROM ledger_nodes WHERE department_id = ? AND depth = 1 LIMIT 1
	`, dept.ID).Scan(&rootNodeID, &rootNodePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find department root node: %w", err)
	}

	var nodesCount int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM ledger_nodes WHERE department_id = ?`, dept.ID).Scan(&nodesCount); err != nil {
		return nil, err
	}

	totals, err := r.GetRollUpTotals(rootNodeID, rootNodePath)
	if err != nil {
		return nil, err
	}

	return &models.DepartmentOverview{
		ID:         dept.ID,
		Name:       dept.Name,
		Department: dept,
		RootNodeID: rootNodeID,
		NodesCount: nodesCount,
		RollUp:     totals,
	}, nil
}
