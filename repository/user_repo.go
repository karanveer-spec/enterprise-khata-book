package repository

import (
	"database/sql"
	"errors"

	"khata-book/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetUserByID(id string) (*models.User, error) {
	row := r.db.QueryRow(`SELECT id, name, role FROM users WHERE id = ?`, id)
	var u models.User
	err := row.Scan(&u.ID, &u.Name, &u.Role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, err
	}

	deptRows, err := r.db.Query(`SELECT department_id FROM user_departments WHERE user_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer deptRows.Close()

	u.DepartmentIDs = []string{}
	for deptRows.Next() {
		var deptID string
		if err := deptRows.Scan(&deptID); err != nil {
			return nil, err
		}
		u.DepartmentIDs = append(u.DepartmentIDs, deptID)
	}

	return &u, nil
}

func (r *UserRepository) GetAllUsers() ([]*models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.name, u.role, COALESCE(ud.department_id, '')
		FROM users u
		LEFT JOIN user_departments ud ON u.id = ud.user_id
		ORDER BY u.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userMap := make(map[string]*models.User)
	var userOrder []string

	for rows.Next() {
		var id, name, role, deptID string
		if err := rows.Scan(&id, &name, &role, &deptID); err != nil {
			return nil, err
		}

		u, exists := userMap[id]
		if !exists {
			u = &models.User{
				ID:            id,
				Name:          name,
				Role:          role,
				DepartmentIDs: []string{},
			}
			userMap[id] = u
			userOrder = append(userOrder, id)
		}

		if deptID != "" {
			u.DepartmentIDs = append(u.DepartmentIDs, deptID)
		}
	}

	var users []*models.User
	for _, id := range userOrder {
		users = append(users, userMap[id])
	}
	return users, nil
}
