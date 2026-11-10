package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"araya-business-hub/be/internal/modules/m1/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (p *PostgresUserRepository) Create(ctx context.Context, u *domain.User) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := `INSERT INTO users (email, password_hash, name, phone, division_id, partner_id, is_active,
	              activation_token_hash, activation_expires_at, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()) RETURNING id, created_at, updated_at`
	err = tx.QueryRow(ctx, q, u.Email, u.PasswordHash, u.Name, u.Phone, u.DivisionID, u.PartnerID, u.IsActive,
		u.ActivationTokenHash, u.ActivationExpiresAt).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return domain.ErrDuplicateEmail
	}
	if err != nil {
		return err
	}
	if err := replaceRoleAssignment(ctx, tx, u); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ErrRoleNotAssignable is returned when the requested role does not exist or
// is inactive, so the user would silently end up without any access.
var ErrRoleNotAssignable = errors.New("role tidak ditemukan atau tidak aktif")

// replaceRoleAssignment stores the user's single effective role/scope.
func replaceRoleAssignment(ctx context.Context, tx pgx.Tx, u *domain.User) error {
	if u.Role == "" {
		return nil
	}
	var roleID int
	err := tx.QueryRow(ctx, `SELECT id FROM roles WHERE name = $1 AND is_active = TRUE`, u.Role).Scan(&roleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRoleNotAssignable
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_role_assignments WHERE user_id = $1`, u.ID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO user_role_assignments (user_id, role_id, unit_usaha_id, created_at)
		VALUES ($1, $2, $3, NOW())`, u.ID, roleID, u.UnitUsahaID)
	return err
}

func (p *PostgresUserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	q := `SELECT u.id, u.email, u.password_hash, u.name, COALESCE(u.phone, ''), u.division_id,
	             COALESCE(d.name, ''), COALESCE(r.name, ''), ura.unit_usaha_id, u.partner_id, u.is_active,
	             COALESCE(u.activation_token_hash, ''), u.activation_expires_at, COALESCE(u.is_manually_disabled, FALSE), u.created_at, u.updated_at
	      FROM users u
	      LEFT JOIN divisions d ON u.division_id = d.id
	      LEFT JOIN LATERAL (
	          SELECT a.role_id, a.unit_usaha_id FROM user_role_assignments a
	          WHERE a.user_id = u.id ORDER BY a.created_at DESC, a.id DESC LIMIT 1
	      ) ura ON TRUE
	      LEFT JOIN roles r ON ura.role_id = r.id
	      WHERE u.id = $1`
	return p.scanUser(p.pool.QueryRow(ctx, q, id))
}

func (p *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	q := `SELECT u.id, u.email, u.password_hash, u.name, COALESCE(u.phone, ''), u.division_id,
	             COALESCE(d.name, ''), COALESCE(r.name, ''), ura.unit_usaha_id, u.partner_id, u.is_active,
	             COALESCE(u.activation_token_hash, ''), u.activation_expires_at, COALESCE(u.is_manually_disabled, FALSE), u.created_at, u.updated_at
	      FROM users u
	      LEFT JOIN divisions d ON u.division_id = d.id
	      LEFT JOIN LATERAL (
	          SELECT a.role_id, a.unit_usaha_id FROM user_role_assignments a
	          WHERE a.user_id = u.id ORDER BY a.created_at DESC, a.id DESC LIMIT 1
	      ) ura ON TRUE
	      LEFT JOIN roles r ON ura.role_id = r.id
	      WHERE LOWER(u.email) = LOWER($1)`
	return p.scanUser(p.pool.QueryRow(ctx, q, email))
}

func (p *PostgresUserRepository) scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.DivisionID,
		&u.DivisionName, &u.Role, &u.UnitUsahaID, &u.PartnerID, &u.IsActive,
		&u.ActivationTokenHash, &u.ActivationExpiresAt, &u.IsManuallyDisabled, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (p *PostgresUserRepository) Update(ctx context.Context, u *domain.User) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := `UPDATE users SET name = $1, phone = $2, division_id = $3, partner_id = $4, is_active = $5, updated_at = NOW() WHERE id = $6`
	ct, err := tx.Exec(ctx, q, u.Name, u.Phone, u.DivisionID, u.PartnerID, u.IsActive, u.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	if err := replaceRoleAssignment(ctx, tx, u); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *PostgresUserRepository) UpdatePassword(ctx context.Context, id int, hash string) error {
	q := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	ct, err := p.pool.Exec(ctx, q, hash, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (p *PostgresUserRepository) Deactivate(ctx context.Context, id int) error {
	q := `UPDATE users SET is_active = FALSE, is_manually_disabled = TRUE, updated_at = NOW() WHERE id = $1`
	ct, err := p.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (p *PostgresUserRepository) GetByActivationTokenHash(ctx context.Context, tokenHash string) (*domain.User, error) {
	q := `SELECT u.id, u.email, u.password_hash, u.name, COALESCE(u.phone, ''), u.division_id,
	             COALESCE(d.name, ''), COALESCE(r.name, ''), ura.unit_usaha_id, u.partner_id, u.is_active,
	             COALESCE(u.activation_token_hash, ''), u.activation_expires_at, COALESCE(u.is_manually_disabled, FALSE), u.created_at, u.updated_at
	      FROM users u
	      LEFT JOIN divisions d ON u.division_id = d.id
	      LEFT JOIN LATERAL (
	          SELECT a.role_id, a.unit_usaha_id FROM user_role_assignments a
	          WHERE a.user_id = u.id ORDER BY a.created_at DESC, a.id DESC LIMIT 1
	      ) ura ON TRUE
	      LEFT JOIN roles r ON ura.role_id = r.id
	      WHERE u.activation_token_hash = $1 AND u.activation_token_hash <> ''`
	return p.scanUser(p.pool.QueryRow(ctx, q, tokenHash))
}

func (p *PostgresUserRepository) IssueActivationToken(ctx context.Context, id int, tokenHash string, expiresAt time.Time) error {
	q := `UPDATE users SET activation_token_hash = $1, activation_expires_at = $2, updated_at = NOW() WHERE id = $3`
	ct, err := p.pool.Exec(ctx, q, tokenHash, expiresAt, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (p *PostgresUserRepository) ActivateAccount(ctx context.Context, id int, passwordHash string) error {
	q := `UPDATE users SET password_hash = $1, activation_token_hash = '', activation_expires_at = NULL,
	             is_active = TRUE, is_manually_disabled = FALSE, updated_at = NOW()
	      WHERE id = $2`
	ct, err := p.pool.Exec(ctx, q, passwordHash, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (p *PostgresUserRepository) List(ctx context.Context, search, division string, partnerID *int, limit, offset int) ([]domain.User, int, error) {
	pattern := "%" + strings.TrimSpace(search) + "%"
	divPattern := "%" + strings.TrimSpace(division) + "%"
	countQ := `SELECT count(*) FROM users u LEFT JOIN divisions d ON u.division_id = d.id
	           WHERE ($1 = '%%' OR u.name ILIKE $1 OR u.email ILIKE $1)
	             AND ($2 = '%%' OR d.name ILIKE $2)
	             AND ($3::int IS NULL OR u.partner_id = $3)`
	var total int
	if err := p.pool.QueryRow(ctx, countQ, pattern, divPattern, partnerID).Scan(&total); err != nil {
		return nil, 0, err
	}
	users, err := p.queryUserList(ctx, pattern, divPattern, partnerID, limit, offset)
	return users, total, err
}

func (p *PostgresUserRepository) queryUserList(ctx context.Context, s, d string, partnerID *int, limit, offset int) ([]domain.User, error) {
	q := `SELECT u.id, u.email, u.password_hash, u.name, COALESCE(u.phone, ''), u.division_id,
	             COALESCE(d.name, ''), COALESCE(r.name, ''), ura.unit_usaha_id, u.partner_id, u.is_active,
	             COALESCE(u.activation_token_hash, ''), u.activation_expires_at, COALESCE(u.is_manually_disabled, FALSE), u.created_at, u.updated_at
	      FROM users u
	      LEFT JOIN divisions d ON u.division_id = d.id
	      LEFT JOIN LATERAL (
	          SELECT a.role_id, a.unit_usaha_id FROM user_role_assignments a
	          WHERE a.user_id = u.id ORDER BY a.created_at DESC, a.id DESC LIMIT 1
	      ) ura ON TRUE
	      LEFT JOIN roles r ON ura.role_id = r.id
	      WHERE ($1 = '%%' OR u.name ILIKE $1 OR u.email ILIKE $1)
	        AND ($2 = '%%' OR d.name ILIKE $2)
	        AND ($3::int IS NULL OR u.partner_id = $3)
	      ORDER BY u.id ASC LIMIT $4 OFFSET $5`
	rows, err := p.pool.Query(ctx, q, s, d, partnerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return p.collectUserRows(rows)
}

func (p *PostgresUserRepository) collectUserRows(rows pgx.Rows) ([]domain.User, error) {
	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.DivisionID,
			&u.DivisionName, &u.Role, &u.UnitUsahaID, &u.PartnerID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
