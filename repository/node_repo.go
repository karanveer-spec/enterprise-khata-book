package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"khata-book/models"
)

type NodeRepository struct {
	db         *sql.DB
	writeQueue *WriteQueue
}

func NewNodeRepository(db *sql.DB, writeQueue *WriteQueue) *NodeRepository {
	return &NodeRepository{
		db:         db,
		writeQueue: writeQueue,
	}
}

func (r *NodeRepository) GetNodeByID(id string) (*models.Node, error) {
	row := r.db.QueryRow(`
		SELECT id, department_id, parent_id, name, path, depth, created_at, updated_at
		FROM ledger_nodes WHERE id = ?
	`, id)

	var n models.Node
	var parentID sql.NullString
	err := row.Scan(&n.ID, &n.DepartmentID, &parentID, &n.Name, &n.Path, &n.Depth, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	if parentID.Valid {
		n.ParentID = &parentID.String
	}
	return &n, nil
}

func (r *NodeRepository) GetAllDepartments() ([]*models.Department, error) {
	rows, err := r.db.Query(`
		SELECT id, name, created_at, updated_at
		FROM departments
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var depts []*models.Department
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		depts = append(depts, &d)
	}
	return depts, nil
}

func (r *NodeRepository) GetDepartmentByID(id string) (*models.Department, error) {
	row := r.db.QueryRow(`SELECT id, name, created_at, updated_at FROM departments WHERE id = ?`, id)
	var d models.Department
	err := row.Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *NodeRepository) GetDepartmentsByIDs(ids []string) ([]*models.Department, error) {
	if len(ids) == 0 {
		return []*models.Department{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, name, created_at, updated_at
		FROM departments
		WHERE id IN (%s)
		ORDER BY name ASC
	`, strings.Join(placeholders, ","))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var depts []*models.Department
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		depts = append(depts, &d)
	}
	return depts, nil
}

func (r *NodeRepository) GetNodesByDepartment(deptID string) ([]*models.Node, error) {
	rows, err := r.db.Query(`
		SELECT id, department_id, parent_id, name, path, depth, created_at, updated_at
		FROM ledger_nodes
		WHERE department_id = ?
		ORDER BY depth ASC, name ASC
	`, deptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*models.Node
	for rows.Next() {
		var n models.Node
		var parentID sql.NullString
		if err := rows.Scan(&n.ID, &n.DepartmentID, &parentID, &n.Name, &n.Path, &n.Depth, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			n.ParentID = &parentID.String
		}
		nodes = append(nodes, &n)
	}
	return nodes, nil
}

// CreateDepartment creates both the department entity and its level-1 root ledger node inside the write worker.
func (r *NodeRepository) CreateDepartment(dept *models.Department, rootNode *models.Node) error {
	return r.writeQueue.ExecWrite(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO departments (id, name)
			VALUES (?, ?)
		`, dept.ID, dept.Name)
		if err != nil {
			return fmt.Errorf("failed to insert department: %w", err)
		}

		_, err = tx.Exec(`
			INSERT INTO ledger_nodes (id, department_id, parent_id, name, path, depth)
			VALUES (?, ?, NULL, ?, ?, 1)
		`, rootNode.ID, dept.ID, rootNode.Name, rootNode.Path)
		if err != nil {
			return fmt.Errorf("failed to insert root node: %w", err)
		}
		return nil
	})
}

// CreateNode adds a group or subgroup node through the serialized write queue.
func (r *NodeRepository) CreateNode(n *models.Node) error {
	return r.writeQueue.ExecWrite(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO ledger_nodes (id, department_id, parent_id, name, path, depth)
			VALUES (?, ?, ?, ?, ?, ?)
		`, n.ID, n.DepartmentID, n.ParentID, n.Name, n.Path, n.Depth)
		return err
	})
}

// RenameNode updates the name of an existing node through the write queue.
func (r *NodeRepository) RenameNode(id string, name string) error {
	return r.writeQueue.ExecWrite(func(tx *sql.Tx) error {
		res, err := tx.Exec(`
			UPDATE ledger_nodes
			SET name = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, name, id)
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

// DeleteNode removes a node and cascades to its subtree and entries through the write queue.
func (r *NodeRepository) DeleteNode(id string) error {
	return r.writeQueue.ExecWrite(func(tx *sql.Tx) error {
		// First verify node exists and get path
		var path string
		err := tx.QueryRow(`SELECT path FROM ledger_nodes WHERE id = ?`, id).Scan(&path)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return models.ErrNotFound
			}
			return err
		}

		// Delete all entries of the node and all descendants
		_, err = tx.Exec(`
			DELETE FROM entries
			WHERE node_id IN (SELECT id FROM ledger_nodes WHERE path LIKE ? || '%')
		`, path)
		if err != nil {
			return fmt.Errorf("failed to delete entries for node subtree: %w", err)
		}

		// Delete the subtree nodes
		_, err = tx.Exec(`DELETE FROM ledger_nodes WHERE path LIKE ? || '%'`, path)
		if err != nil {
			return fmt.Errorf("failed to delete node subtree: %w", err)
		}

		return nil
	})
}

// GetSubtreeMaxDepth finds the maximum depth among nodeID and all its descendants using the indexed path.
func (r *NodeRepository) GetSubtreeMaxDepth(path string) (int, error) {
	row := r.db.QueryRow(`
		SELECT COALESCE(MAX(depth), 0)
		FROM ledger_nodes
		WHERE path LIKE ? || '%'
	`, path)

	var maxDepth int
	if err := row.Scan(&maxDepth); err != nil {
		return 0, err
	}
	return maxDepth, nil
}

// IsDescendant checks if candidatePath starts with ancestorPath.
func (r *NodeRepository) IsDescendant(ancestorPath, candidatePath string) bool {
	return strings.HasPrefix(candidatePath, ancestorPath)
}

// MoveNode moves a node and its entire subtree under a new parent, updating paths, depths, and department_id.
func (r *NodeRepository) MoveNode(nodeID string, oldPath string, oldDepth int, newParentID string, newDeptID string, newParentPath string, newDepth int) error {
	return r.writeQueue.ExecWrite(func(tx *sql.Tx) error {
		// Construct new path for the moved node: newParentPath + nodeID + "/"
		newPath := fmt.Sprintf("%s%s/", newParentPath, nodeID)
		depthDelta := newDepth - oldDepth

		// 1. Update paths, depths, and department_id for the entire subtree
		// SQLite: SUBSTR(path, LENGTH(oldPath) + 1) extracts the descendant suffix
		updateQuery := `
			UPDATE ledger_nodes
			SET path = ? || SUBSTR(path, LENGTH(?) + 1),
			    depth = depth + ?,
			    department_id = ?,
			    updated_at = CURRENT_TIMESTAMP
			WHERE path LIKE ? || '%';
		`
		_, err := tx.Exec(updateQuery, newPath, oldPath, depthDelta, newDeptID, oldPath)
		if err != nil {
			return fmt.Errorf("failed to update node subtree paths: %w", err)
		}

		// 2. Update the parent_id of the moved root node
		_, err = tx.Exec(`UPDATE ledger_nodes SET parent_id = ? WHERE id = ?`, newParentID, nodeID)
		if err != nil {
			return fmt.Errorf("failed to update parent_id: %w", err)
		}

		// 3. Update department_id for all entries under this subtree
		_, err = tx.Exec(`
			UPDATE entries
			SET department_id = ?
			WHERE node_id IN (SELECT id FROM ledger_nodes WHERE path LIKE ? || '%')
		`, newDeptID, newPath)
		if err != nil {
			return fmt.Errorf("failed to update entry department mapping: %w", err)
		}

		return nil
	})
}

// GetDepartmentNodeCount returns total count of nodes under a department.
func (r *NodeRepository) GetDepartmentNodeCount(deptID string) (int, error) {
	row := r.db.QueryRow(`SELECT COUNT(*) FROM ledger_nodes WHERE department_id = ?`, deptID)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetAncestors returns breadcrumbs ordered from root down to the given node using path prefix match.
func (r *NodeRepository) GetAncestors(targetPath string) ([]*models.Node, error) {
	rows, err := r.db.Query(`
		SELECT id, department_id, parent_id, name, path, depth, created_at, updated_at
		FROM ledger_nodes
		WHERE ? LIKE path || '%'
		ORDER BY depth ASC
	`, targetPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ancestors []*models.Node
	for rows.Next() {
		var n models.Node
		var parentID sql.NullString
		if err := rows.Scan(&n.ID, &n.DepartmentID, &parentID, &n.Name, &n.Path, &n.Depth, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			n.ParentID = &parentID.String
		}
		ancestors = append(ancestors, &n)
	}
	return ancestors, nil
}
