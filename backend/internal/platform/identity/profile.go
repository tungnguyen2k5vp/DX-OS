package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/jackc/pgx/v5"
)

var ErrInactive = errors.New("business user is inactive")

type Profile struct {
	ID             string
	DepartmentID   string
	OrganizationID string
	Active         bool
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func Ensure(
	ctx context.Context,
	database queryRower,
	principal auth.Principal,
	defaultDepartmentCode string,
) (Profile, error) {
	var profile Profile
	err := database.QueryRow(ctx, `
		INSERT INTO users (
			keycloak_subject, username, email, display_name, department_id
		)
		SELECT $1, $2, NULLIF($3, ''), $2, d.id
		FROM departments d
		WHERE d.code = $4 AND d.active
		ORDER BY d.created_at
		LIMIT 1
		ON CONFLICT (keycloak_subject) DO UPDATE
		SET username = EXCLUDED.username,
			email = COALESCE(EXCLUDED.email, users.email),
			updated_at = now()
		RETURNING
			users.id,
			users.department_id,
			(SELECT organization_id FROM departments WHERE id = users.department_id),
			users.active
	`, principal.Subject, principal.Username, principal.Email, defaultDepartmentCode).Scan(
		&profile.ID, &profile.DepartmentID, &profile.OrganizationID, &profile.Active,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, errors.New("default active department is not configured")
	}
	if err != nil {
		return Profile{}, fmt.Errorf("provision business user: %w", err)
	}
	if !profile.Active {
		return Profile{}, ErrInactive
	}
	return profile, nil
}
