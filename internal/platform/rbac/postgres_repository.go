package rbac

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (p *PostgresRepository) CreateRole(ctx context.Context, role Role) error {
	q := `INSERT INTO roles (name, display_name, description, is_active, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id`
	return p.pool.QueryRow(ctx, q, role.Name, role.DisplayName, role.Description, role.IsActive).Scan(&role.ID)
}

func (p *PostgresRepository) GetRole(ctx context.Context, roleName string) (*Role, error) {
	q := `SELECT id, name, display_name, description, is_active, created_at, updated_at
	      FROM roles WHERE name = $1`
	var r Role
	err := p.pool.QueryRow(ctx, q, roleName).Scan(&r.ID, &r.Name, &r.DisplayName, &r.Description, &r.IsActive, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return &r, nil
}

func (p *PostgresRepository) ListRoles(ctx context.Context) ([]Role, error) {
	q := `SELECT id, name, display_name, description, is_active, created_at, updated_at FROM roles ORDER BY id ASC`
	rows, err := p.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.DisplayName, &r.Description, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, nil
}

func (p *PostgresRepository) UpdateRole(ctx context.Context, role Role) error {
	q := `UPDATE roles SET display_name = $1, description = $2, is_active = $3, updated_at = NOW() WHERE name = $4`
	ct, err := p.pool.Exec(ctx, q, role.DisplayName, role.Description, role.IsActive, role.Name)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrRoleNotFound
	}
	return nil
}

func (p *PostgresRepository) DeactivateRole(ctx context.Context, roleName string) error {
	q := `UPDATE roles SET is_active = FALSE, updated_at = NOW() WHERE name = $1`
	ct, err := p.pool.Exec(ctx, q, roleName)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrRoleNotFound
	}
	return nil
}

func (p *PostgresRepository) AssignPermission(ctx context.Context, roleName, permKey string) error {
	q := `INSERT INTO role_permissions (role_id, permission_id)
	      SELECT r.id, p.id FROM roles r, permissions p
	      WHERE r.name = $1 AND p.key = $2
	      ON CONFLICT (role_id, permission_id) DO NOTHING`
	_, err := p.pool.Exec(ctx, q, roleName, permKey)
	return err
}

func (p *PostgresRepository) RevokePermission(ctx context.Context, roleName, permKey string) error {
	q := `DELETE FROM role_permissions
	      WHERE role_id = (SELECT id FROM roles WHERE name = $1)
	        AND permission_id = (SELECT id FROM permissions WHERE key = $2)`
	_, err := p.pool.Exec(ctx, q, roleName, permKey)
	return err
}

func (p *PostgresRepository) GetRolePermissions(ctx context.Context, roleName string) ([]string, error) {
	q := `SELECT p.key FROM permissions p
	      JOIN role_permissions rp ON p.id = rp.permission_id
	      JOIN roles r ON rp.role_id = r.id
	      WHERE r.name = $1 AND p.is_active = TRUE`
	rows, err := p.pool.Query(ctx, q, roleName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		perms = append(perms, k)
	}
	return perms, nil
}

func (p *PostgresRepository) AssignUserRole(ctx context.Context, a UserRoleAssignment) error {
	q := `INSERT INTO user_role_assignments (user_id, role_id, unit_usaha_id, created_at)
	      SELECT $1, r.id, $2, NOW() FROM roles r WHERE r.name = $3
	      ON CONFLICT (user_id, role_id, unit_usaha_id) DO NOTHING`
	_, err := p.pool.Exec(ctx, q, a.UserID, a.UnitUsahaID, a.RoleName)
	return err
}

func (p *PostgresRepository) GetUserRoleAssignments(ctx context.Context, userID int) ([]UserRoleAssignment, error) {
	q := `SELECT ura.id, ura.user_id, ura.role_id, r.name, ura.unit_usaha_id, ura.created_at
	      FROM user_role_assignments ura
	      JOIN roles r ON ura.role_id = r.id
	      WHERE ura.user_id = $1`
	rows, err := p.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []UserRoleAssignment
	for rows.Next() {
		var a UserRoleAssignment
		if err := rows.Scan(&a.ID, &a.UserID, &a.RoleID, &a.RoleName, &a.UnitUsahaID, &a.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, a)
	}
	return res, nil
}
