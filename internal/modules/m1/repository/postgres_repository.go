package repository

import (
	"context"
	"errors"
	"strings"

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
	q := `INSERT INTO users (email, password_hash, name, phone, division_id, is_active, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) RETURNING id, created_at, updated_at`
	err := p.pool.QueryRow(ctx, q, u.Email, u.PasswordHash, u.Name, u.Phone, u.DivisionID, u.IsActive).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return domain.ErrDuplicateEmail
	}
	if err != nil {
		return err
	}
	if u.Role != "" {
		assignQ := `INSERT INTO user_role_assignments (user_id, role_id, unit_usaha_id, created_at)
			SELECT $1, r.id, $2, NOW() FROM roles r WHERE r.name = $3
			ON CONFLICT (user_id, role_id, unit_usaha_id) DO NOTHING`
		_, _ = p.pool.Exec(ctx, assignQ, u.ID, u.UnitUsahaID, u.Role)
	}
	return nil
}

func (p *PostgresUserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	q := `SELECT u.id, u.email, u.password_hash, u.name, COALESCE(u.phone, ''), u.division_id,
	             COALESCE(d.name, ''), COALESCE(r.name, ''), ura.unit_usaha_id, u.is_active, u.created_at, u.updated_at
	      FROM users u
	      LEFT JOIN divisions d ON u.division_id = d.id
	      LEFT JOIN user_role_assignments ura ON u.id = ura.user_id
	      LEFT JOIN roles r ON ura.role_id = r.id
	      WHERE u.id = $1`
	return p.scanUser(p.pool.QueryRow(ctx, q, id))
}

func (p *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	q := `SELECT u.id, u.email, u.password_hash, u.name, COALESCE(u.phone, ''), u.division_id,
	             COALESCE(d.name, ''), COALESCE(r.name, ''), ura.unit_usaha_id, u.is_active, u.created_at, u.updated_at
	      FROM users u
	      LEFT JOIN divisions d ON u.division_id = d.id
	      LEFT JOIN user_role_assignments ura ON u.id = ura.user_id
	      LEFT JOIN roles r ON ura.role_id = r.id
	      WHERE LOWER(u.email) = LOWER($1)`
	return p.scanUser(p.pool.QueryRow(ctx, q, email))
}

func (p *PostgresUserRepository) scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.DivisionID,
		&u.DivisionName, &u.Role, &u.UnitUsahaID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (p *PostgresUserRepository) Update(ctx context.Context, u *domain.User) error {
	q := `UPDATE users SET name = $1, phone = $2, division_id = $3, is_active = $4, updated_at = NOW() WHERE id = $5`
	ct, err := p.pool.Exec(ctx, q, u.Name, u.Phone, u.DivisionID, u.IsActive, u.ID)
	if err != nil || ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	if u.Role != "" {
		_, _ = p.pool.Exec(ctx, `DELETE FROM user_role_assignments WHERE user_id = $1`, u.ID)
		assignQ := `INSERT INTO user_role_assignments (user_id, role_id, unit_usaha_id, created_at)
			SELECT $1, r.id, $2, NOW() FROM roles r WHERE r.name = $3
			ON CONFLICT (user_id, role_id, unit_usaha_id) DO NOTHING`
		_, _ = p.pool.Exec(ctx, assignQ, u.ID, u.UnitUsahaID, u.Role)
	}
	return nil
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
	q := `UPDATE users SET is_active = FALSE, updated_at = NOW() WHERE id = $1`
	ct, err := p.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (p *PostgresUserRepository) List(ctx context.Context, search, division string, limit, offset int) ([]domain.User, int, error) {
	pattern := "%" + strings.TrimSpace(search) + "%"
	divPattern := "%" + strings.TrimSpace(division) + "%"
	countQ := `SELECT count(*) FROM users u LEFT JOIN divisions d ON u.division_id = d.id
	           WHERE ($1 = '%%' OR u.name ILIKE $1 OR u.email ILIKE $1)
	             AND ($2 = '%%' OR d.name ILIKE $2)`
	var total int
	if err := p.pool.QueryRow(ctx, countQ, pattern, divPattern).Scan(&total); err != nil {
		return nil, 0, err
	}
	users, err := p.queryUserList(ctx, pattern, divPattern, limit, offset)
	return users, total, err
}

func (p *PostgresUserRepository) queryUserList(ctx context.Context, s, d string, limit, offset int) ([]domain.User, error) {
	q := `SELECT u.id, u.email, u.password_hash, u.name, COALESCE(u.phone, ''), u.division_id,
	             COALESCE(d.name, ''), COALESCE(r.name, ''), ura.unit_usaha_id, u.is_active, u.created_at, u.updated_at
	      FROM users u
	      LEFT JOIN divisions d ON u.division_id = d.id
	      LEFT JOIN user_role_assignments ura ON u.id = ura.user_id
	      LEFT JOIN roles r ON ura.role_id = r.id
	      WHERE ($1 = '%%' OR u.name ILIKE $1 OR u.email ILIKE $1)
	        AND ($2 = '%%' OR d.name ILIKE $2)
	      ORDER BY u.id ASC LIMIT $3 OFFSET $4`
	rows, err := p.pool.Query(ctx, q, s, d, limit, offset)
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
			&u.DivisionName, &u.Role, &u.UnitUsahaID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
