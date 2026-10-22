package models

import (
	"errors"
	"fmt"
)

const (
	// MaxDepth defines the maximum allowable nesting level (1 = Department, 2 = Group, 3..5 = Subgroups)
	MaxDepth = 5

	// RoleGlobalAdmin allows full organization-wide access
	RoleGlobalAdmin = "Global Admin"

	// RoleDepartmentViewer / DepartmentUser allows access strictly to assigned departments
	RoleDepartmentViewer = "Department Viewer"

	// EntryTypeDebit represents Outflow / Udhar / Debit
	EntryTypeDebit = "DEBIT"

	// EntryTypeCredit represents Inflow / Jama / Credit
	EntryTypeCredit = "CREDIT"
)

var (
	ErrNotFound                   = errors.New("resource not found")
	ErrUnauthorized               = errors.New("unauthorized department access")
	ErrInvalidInput               = errors.New("invalid input data")
	ErrMaxDepthExceeded           = fmt.Errorf("exceeds maximum allowed hierarchy depth of %d", MaxDepth)
	ErrCannotMoveToDescendant      = errors.New("cannot move a node into itself or its own descendant")
	ErrMoveExceedsMaxDepth        = fmt.Errorf("moving node would cause its descendants to exceed maximum depth of %d", MaxDepth)
	ErrDepartmentCannotHaveParent = errors.New("a department root node cannot have a parent")
	ErrNonDepartmentNeedsParent   = errors.New("a non-department node must specify a parent node")
)

// APIError represents structured error responses
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
