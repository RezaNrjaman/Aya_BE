package rbac

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (p *PostgresRepository) CreateRole(ctx context.Context, role Role) error {
	q := `INSERT INTO roles (name, display_name, description, partner_id, division_id, is_system, is_active, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) RETURNING id`
	err := p.pool.QueryRow(ctx, q, role.Name, role.DisplayName, role.Description, role.PartnerID, role.DivisionID, role.IsSystem, role.IsActive).Scan(&role.ID)
	if isUniqueViolation(err) {
		return ErrDuplicateRole
	}
	return err
}

func (p *PostgresRepository) GetRole(ctx context.Context, roleName string) (*Role, error) {
	q := `SELECT r.id, r.name, r.display_name, COALESCE(r.description,''), r.partner_id, r.division_id,
	             COALESCE(d.name,''), COALESCE(bu.name,''), COALESCE(c.name,''),
	             r.is_system, r.is_active, r.created_at, r.updated_at
	      FROM roles r
	      LEFT JOIN divisions d ON d.id=r.division_id
	      LEFT JOIN business_units bu ON bu.id=d.unit_usaha_id
	      LEFT JOIN business_categories c ON c.id=d.category_id
	      WHERE r.name = $1`
	var r Role
	err := p.pool.QueryRow(ctx, q, roleName).Scan(&r.ID, &r.Name, &r.DisplayName, &r.Description, &r.PartnerID, &r.DivisionID,
		&r.DivisionName, &r.UnitName, &r.CategoryName, &r.IsSystem, &r.IsActive, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return &r, nil
}

func (p *PostgresRepository) listRolePermissions(ctx context.Context, roleID int) ([]Permission, error) {
	q := `SELECT pe.id, pe.key, COALESCE(pe.module,''), COALESCE(pe.name,''), COALESCE(pe.description,''), pe.is_active
	      FROM permissions pe
	      JOIN role_permissions rp ON pe.id = rp.permission_id
	      WHERE rp.role_id = $1 AND pe.is_active = TRUE`
	rows, err := p.pool.Query(ctx, q, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var perms []Permission
	for rows.Next() {
		var pm Permission
		if err := rows.Scan(&pm.ID, &pm.Key, &pm.Module, &pm.Name, &pm.Description, &pm.IsActive); err != nil {
			return nil, err
		}
		perms = append(perms, pm)
	}
	return perms, nil
}

func (p *PostgresRepository) ListRoles(ctx context.Context, partnerID *int) ([]Role, error) {
	q := `SELECT r.id, r.name, r.display_name, COALESCE(r.description,''), r.partner_id, r.division_id,
	             COALESCE(d.name,''), COALESCE(bu.name,''), COALESCE(c.name,''),
	             r.is_system, r.is_active, r.created_at, r.updated_at
	      FROM roles r
	      LEFT JOIN divisions d ON d.id=r.division_id
	      LEFT JOIN business_units bu ON bu.id=d.unit_usaha_id
	      LEFT JOIN business_categories c ON c.id=d.category_id
	      WHERE ($1::INT IS NULL OR r.partner_id IS NULL OR r.partner_id = $1)
	      ORDER BY r.division_id NULLS FIRST, r.id ASC`
	rows, err := p.pool.Query(ctx, q, partnerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.DisplayName, &r.Description, &r.PartnerID, &r.DivisionID, &r.DivisionName, &r.UnitName, &r.CategoryName,
			&r.IsSystem, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	for i, r := range roles {
		perms, _ := p.listRolePermissions(ctx, r.ID)
		if perms == nil {
			perms = []Permission{}
		}
		roles[i].Permissions = perms
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
	role, err := p.GetRole(ctx, roleName)
	if err != nil {
		return err
	}
	if !role.IsActive {
		return ErrRoleInactive
	}
	return insertRolePermission(ctx, p.pool, role.ID, permKey)
}

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// insertRolePermission maps a permission key to a role and reports an unknown
// key instead of silently inserting nothing.
func insertRolePermission(ctx context.Context, db execer, roleID int, permKey string) error {
	var permID int
	err := db.QueryRow(ctx, `SELECT id FROM permissions WHERE key = $1`, permKey).Scan(&permID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPermissionNotFound
	}
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)
	      ON CONFLICT (role_id, permission_id) DO NOTHING`, roleID, permID)
	return err
}

func (p *PostgresRepository) SetRolePermissions(ctx context.Context, roleName string, permKeys []string) error {
	role, err := p.GetRole(ctx, roleName)
	if err != nil {
		return err
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, role.ID); err != nil {
		return err
	}
	for _, key := range permKeys {
		if err := insertRolePermission(ctx, tx, role.ID, key); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
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
	      WHERE r.name = $1 AND r.is_active = TRUE AND p.is_active = TRUE`
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

func (p *PostgresRepository) ListRolePermissionKeys(ctx context.Context, roleName string) ([]string, error) {
	if _, err := p.GetRole(ctx, roleName); err != nil {
		return nil, err
	}
	q := `SELECT p.key FROM permissions p
	      JOIN role_permissions rp ON p.id = rp.permission_id
	      JOIN roles r ON rp.role_id = r.id
	      WHERE r.name = $1 ORDER BY p.key`
	rows, err := p.pool.Query(ctx, q, roleName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (p *PostgresRepository) AssignUserRole(ctx context.Context, a UserRoleAssignment) error {
	role, err := p.GetRole(ctx, a.RoleName)
	if err != nil {
		return err
	}
	if !role.IsActive {
		return ErrRoleInactive
	}
	q := `INSERT INTO user_role_assignments (user_id, role_id, unit_usaha_id, created_at)
	      VALUES ($1, $2, $3, NOW())
	      ON CONFLICT (user_id, role_id, unit_usaha_id) DO NOTHING`
	_, err = p.pool.Exec(ctx, q, a.UserID, role.ID, a.UnitUsahaID)
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
