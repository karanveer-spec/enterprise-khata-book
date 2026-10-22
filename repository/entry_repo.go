package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"khata-book/models"
)

type EntryRepository struct {
	db         *sql.DB
	writeQueue *WriteQueue
}

func NewEntryRepository(db *sql.DB, writeQueue *WriteQueue) *EntryRepository {
	return &EntryRepository{
		db:         db,
		writeQueue: writeQueue,
	}
}

// CreateEntry inserts a new financial entry via the serialized write queue.
func (r *EntryRepository) CreateEntry(e *models.Entry) error {
	return r.writeQueue.ExecWrite(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO entries (id, node_id, department_id, type, amount_paise, entry_date, description, reference)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, e.ID, e.NodeID, e.DepartmentID, e.Type, e.AmountPaise, e.EntryDate, e.Description, e.Reference)
		return err
	})
}

// GetEntryByID retrieves an entry by its ID.
func (r *EntryRepository) GetEntryByID(id string) (*models.Entry, error) {
	row := r.db.QueryRow(`
		SELECT e.id, e.node_id, e.department_id, e.type, e.amount_paise, e.entry_date, e.description, COALESCE(e.reference, ''), n.name, e.created_at
		FROM entries e
		JOIN ledger_nodes n ON e.node_id = n.id
		WHERE e.id = ?
	`, id)

	var e models.Entry
	err := row.Scan(&e.ID, &e.NodeID, &e.DepartmentID, &e.Type, &e.AmountPaise, &e.EntryDate, &e.Description, &e.Reference, &e.NodeName, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	e.AmountRupees = models.PaiseToRupees(e.AmountPaise)
	e.Amount = e.AmountRupees
	e.FormattedAmount = models.FormatIndianRupees(e.AmountPaise)
	return &e, nil
}

// DeleteEntry deletes an entry via the write queue.
func (r *EntryRepository) DeleteEntry(id string) error {
	return r.writeQueue.ExecWrite(func(tx *sql.Tx) error {
		res, err := tx.Exec(`DELETE FROM entries WHERE id = ?`, id)
		if err != nil {
			return err
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return models.ErrNotFound
		}
		return nil
	})
}

// BulkDeleteEntries deletes multiple entries via the write queue.
func (r *EntryRepository) BulkDeleteEntries(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.writeQueue.ExecWrite(func(tx *sql.Tx) error {
		placeholders := make([]string, len(ids))
		args := make([]interface{}, len(ids))
		for i, id := range ids {
			placeholders[i] = "?"
			args[i] = id
		}
		query := fmt.Sprintf("DELETE FROM entries WHERE id IN (%s)", strings.Join(placeholders, ","))
		_, err := tx.Exec(query, args...)
		return err
	})
}

// BulkMoveEntries reassigns entries to a target node and department via the write queue.
func (r *EntryRepository) BulkMoveEntries(entryIDs []string, targetNodeID, targetDeptID string) error {
	if len(entryIDs) == 0 {
		return nil
	}
	return r.writeQueue.ExecWrite(func(tx *sql.Tx) error {
		placeholders := make([]string, len(entryIDs))
		args := make([]interface{}, len(entryIDs)+2)
		args[0] = targetNodeID
		args[1] = targetDeptID
		for i, id := range entryIDs {
			placeholders[i] = "?"
			args[i+2] = id
		}
		query := fmt.Sprintf("UPDATE entries SET node_id = ?, department_id = ? WHERE id IN (%s)", strings.Join(placeholders, ","))
		_, err := tx.Exec(query, args...)
		return err
	})
}

// ListEntries returns a paginated, filtered, and searched list of entries using SQL.
func (r *EntryRepository) ListEntries(filter models.EntryFilter, nodePath string) (*models.EntryListResponse, error) {
	var whereConditions []string
	var args []interface{}

	// 1. Node Scope: direct node vs all descendants using materialized path
	if filter.IncludeDescendants && nodePath != "" {
		whereConditions = append(whereConditions, "e.node_id IN (SELECT id FROM ledger_nodes WHERE path LIKE ? || '%')")
		args = append(args, nodePath)
	} else if filter.NodeID != "" {
		whereConditions = append(whereConditions, "e.node_id = ?")
		args = append(args, filter.NodeID)
	}

	// 2. Type Filter (CREDIT / DEBIT)
	if filter.Type != "" {
		whereConditions = append(whereConditions, "e.type = ?")
		args = append(args, filter.Type)
	}

	// 3. Search Filter (Voucher/Ref or Description)
	if strings.TrimSpace(filter.Search) != "" {
		whereConditions = append(whereConditions, "(e.description LIKE ? OR e.reference LIKE ?)")
		searchTerm := "%" + strings.TrimSpace(filter.Search) + "%"
		args = append(args, searchTerm, searchTerm)
	}

	// 4. Date Filtering
	now := time.Now()
	switch filter.Period {
	case "today":
		todayStr := now.Format("2006-01-02")
		whereConditions = append(whereConditions, "e.entry_date = ?")
		args = append(args, todayStr)
	case "this_month":
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		nextMonthStart := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		whereConditions = append(whereConditions, "e.entry_date >= ? AND e.entry_date < ?")
		args = append(args, monthStart, nextMonthStart)
	case "this_quarter":
		currentQuarter := (int(now.Month()) - 1) / 3
		quarterStartMonth := time.Month(currentQuarter*3 + 1)
		quarterStart := time.Date(now.Year(), quarterStartMonth, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		nextQuarterStart := time.Date(now.Year(), quarterStartMonth+3, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		whereConditions = append(whereConditions, "e.entry_date >= ? AND e.entry_date < ?")
		args = append(args, quarterStart, nextQuarterStart)
	case "custom":
		if filter.FromDate != "" {
			whereConditions = append(whereConditions, "e.entry_date >= ?")
			args = append(args, filter.FromDate)
		}
		if filter.ToDate != "" {
			whereConditions = append(whereConditions, "e.entry_date <= ?")
			args = append(args, filter.ToDate)
		}
	default:
		// Also respect explicit FromDate / ToDate if passed without period
		if filter.FromDate != "" {
			whereConditions = append(whereConditions, "e.entry_date >= ?")
			args = append(args, filter.FromDate)
		}
		if filter.ToDate != "" {
			whereConditions = append(whereConditions, "e.entry_date <= ?")
			args = append(args, filter.ToDate)
		}
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// 5. Total Count Query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM entries e %s`, whereClause)
	var totalCount int
	if err := r.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count entries: %w", err)
	}

	// 6. Pagination limits & offsets
	limit := filter.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 200 {
		limit = 200
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// 7. Sorting
	orderByCol := "e.entry_date"
	switch filter.SortBy {
	case "amount":
		orderByCol = "e.amount_paise"
	case "created_at":
		orderByCol = "e.created_at"
	case "date":
		orderByCol = "e.entry_date"
	}

	orderDir := "DESC"
	if strings.ToUpper(filter.SortDir) == "ASC" {
		orderDir = "ASC"
	}

	selectQuery := fmt.Sprintf(`
		SELECT e.id, e.node_id, e.department_id, e.type, e.amount_paise, e.entry_date, e.description, COALESCE(e.reference, ''), n.name, e.created_at
		FROM entries e
		JOIN ledger_nodes n ON e.node_id = n.id
		%s
		ORDER BY %s %s, e.created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause, orderByCol, orderDir)

	queryArgs := append(args, limit, offset)
	rows, err := r.db.Query(selectQuery, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}
	defer rows.Close()

	var items []*models.Entry
	for rows.Next() {
		var e models.Entry
		if err := rows.Scan(&e.ID, &e.NodeID, &e.DepartmentID, &e.Type, &e.AmountPaise, &e.EntryDate, &e.Description, &e.Reference, &e.NodeName, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.AmountRupees = models.PaiseToRupees(e.AmountPaise)
		e.Amount = e.AmountRupees
		e.FormattedAmount = models.FormatIndianRupees(e.AmountPaise)
		items = append(items, &e)
	}

	totalPages := (totalCount + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.EntryListResponse{
		Items:      items,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}
