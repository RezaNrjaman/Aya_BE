package repository

import (
	"context"
	"errors"
	"strings"

	"araya-business-hub/be/internal/modules/m3/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPartnerRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPartnerRepository(pool *pgxpool.Pool) *PostgresPartnerRepository {
	return &PostgresPartnerRepository{pool: pool}
}

func (p *PostgresPartnerRepository) Create(ctx context.Context, part *domain.Partner) error {
	q := `INSERT INTO partners (company_name, contact_person, email, phone, address, status, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	      RETURNING id, created_at, updated_at`
	err := p.pool.QueryRow(ctx, q, part.CompanyName, part.ContactPerson, part.Email, part.Phone, part.Address, part.Status).
		Scan(&part.ID, &part.CreatedAt, &part.UpdatedAt)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return domain.ErrEmailAlreadyRegistered
	}
	return err
}

func (p *PostgresPartnerRepository) GetByID(ctx context.Context, id int) (*domain.Partner, error) {
	q := `SELECT p.id, p.user_id, p.company_name, p.contact_person, p.email, p.phone, COALESCE(p.address, ''),
	             p.status, COALESCE(p.rejection_reason, ''), COALESCE(p.suspension_reason, ''),
	             p.verified_at, COALESCE(u.name, ''), p.created_at, p.updated_at
	      FROM partners p LEFT JOIN users u ON p.verified_by = u.id WHERE p.id = $1`
	return p.scanPartner(p.pool.QueryRow(ctx, q, id))
}

func (p *PostgresPartnerRepository) GetByEmail(ctx context.Context, email string) (*domain.Partner, error) {
	q := `SELECT p.id, p.user_id, p.company_name, p.contact_person, p.email, p.phone, COALESCE(p.address, ''),
	             p.status, COALESCE(p.rejection_reason, ''), COALESCE(p.suspension_reason, ''),
	             p.verified_at, COALESCE(u.name, ''), p.created_at, p.updated_at
	      FROM partners p LEFT JOIN users u ON p.verified_by = u.id WHERE LOWER(p.email) = LOWER($1)`
	return p.scanPartner(p.pool.QueryRow(ctx, q, email))
}

func (p *PostgresPartnerRepository) GetByUserID(ctx context.Context, userID int) (*domain.Partner, error) {
	q := `SELECT p.id, p.user_id, p.company_name, p.contact_person, p.email, p.phone, COALESCE(p.address, ''),
	             p.status, COALESCE(p.rejection_reason, ''), COALESCE(p.suspension_reason, ''),
	             p.verified_at, COALESCE(v.name, ''), p.created_at, p.updated_at
	      FROM partners p
	      LEFT JOIN users v ON p.verified_by = v.id
	      WHERE p.user_id = $1
	         OR EXISTS (SELECT 1 FROM users u WHERE u.id = $1 AND u.partner_id = p.id)
	      LIMIT 1`
	return p.scanPartner(p.pool.QueryRow(ctx, q, userID))
}

func (p *PostgresPartnerRepository) scanPartner(row pgx.Row) (*domain.Partner, error) {
	var part domain.Partner
	err := row.Scan(&part.ID, &part.UserID, &part.CompanyName, &part.ContactPerson, &part.Email, &part.Phone,
		&part.Address, &part.Status, &part.RejectionReason, &part.SuspensionReason, &part.VerifiedAt,
		&part.VerifiedBy, &part.CreatedAt, &part.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPartnerNotFound
		}
		return nil, err
	}
	return &part, nil
}

func (p *PostgresPartnerRepository) Update(ctx context.Context, part *domain.Partner) error {
	q := `UPDATE partners SET company_name = $1, contact_person = $2, phone = $3, address = $4,
	             status = $5, rejection_reason = $6, suspension_reason = $7, verified_at = $8,
	             user_id = $9, updated_at = NOW()
	      WHERE id = $10`
	ct, err := p.pool.Exec(ctx, q, part.CompanyName, part.ContactPerson, part.Phone, part.Address,
		part.Status, part.RejectionReason, part.SuspensionReason, part.VerifiedAt, part.UserID, part.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrPartnerNotFound
	}
	return nil
}

func (p *PostgresPartnerRepository) List(ctx context.Context, status, search string, limit, offset int) ([]domain.Partner, int, error) {
	pattern := "%" + strings.TrimSpace(search) + "%"
	countQ := `SELECT count(*) FROM partners WHERE ($1 = '' OR status = $1)
	           AND ($2 = '%%' OR company_name ILIKE $2 OR contact_person ILIKE $2 OR email ILIKE $2)`
	var total int
	if err := p.pool.QueryRow(ctx, countQ, status, pattern).Scan(&total); err != nil {
		return nil, 0, err
	}
	partners, err := p.queryPartnerList(ctx, status, pattern, limit, offset)
	return partners, total, err
}

func (p *PostgresPartnerRepository) queryPartnerList(ctx context.Context, status, s string, limit, offset int) ([]domain.Partner, error) {
	q := `SELECT p.id, p.user_id, p.company_name, p.contact_person, p.email, p.phone, COALESCE(p.address, ''),
	             p.status, COALESCE(p.rejection_reason, ''), COALESCE(p.suspension_reason, ''),
	             p.verified_at, COALESCE(u.name, ''), p.created_at, p.updated_at
	      FROM partners p LEFT JOIN users u ON p.verified_by = u.id
	      WHERE ($1 = '' OR p.status = $1) AND ($2 = '%%' OR p.company_name ILIKE $2 OR p.contact_person ILIKE $2 OR p.email ILIKE $2)
	      ORDER BY p.id ASC LIMIT $3 OFFSET $4`
	rows, err := p.pool.Query(ctx, q, status, s, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return p.collectRows(rows)
}

func (p *PostgresPartnerRepository) ListVerified(ctx context.Context) ([]domain.Partner, error) {
	q := `SELECT p.id, p.user_id, p.company_name, p.contact_person, p.email, p.phone, COALESCE(p.address, ''),
	             p.status, COALESCE(p.rejection_reason, ''), COALESCE(p.suspension_reason, ''),
	             p.verified_at, COALESCE(u.name, ''), p.created_at, p.updated_at
	      FROM partners p LEFT JOIN users u ON p.verified_by = u.id
	      WHERE p.status = 'terverifikasi' ORDER BY p.company_name ASC`
	rows, err := p.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return p.collectRows(rows)
}

func (p *PostgresPartnerRepository) collectRows(rows pgx.Rows) ([]domain.Partner, error) {
	var list []domain.Partner
	for rows.Next() {
		var part domain.Partner
		if err := rows.Scan(&part.ID, &part.UserID, &part.CompanyName, &part.ContactPerson, &part.Email, &part.Phone,
			&part.Address, &part.Status, &part.RejectionReason, &part.SuspensionReason, &part.VerifiedAt,
			&part.VerifiedBy, &part.CreatedAt, &part.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, part)
	}
	return list, nil
}

func (p *PostgresPartnerRepository) AddHistory(ctx context.Context, h *domain.StatusHistory) error {
	q := `INSERT INTO partner_status_histories (partner_id, from_status, to_status, reason, changed_by_name, created_at)
	      VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING id, created_at`
	return p.pool.QueryRow(ctx, q, h.PartnerID, h.FromStatus, h.ToStatus, h.Reason, h.ChangedBy).Scan(&h.ID, &h.CreatedAt)
}

func (p *PostgresPartnerRepository) GetHistories(ctx context.Context, partnerID int) ([]domain.StatusHistory, error) {
	q := `SELECT id, partner_id, COALESCE(from_status, ''), to_status, COALESCE(reason, ''),
	             COALESCE(changed_by_name, ''), created_at
	      FROM partner_status_histories WHERE partner_id = $1 ORDER BY created_at ASC`
	rows, err := p.pool.Query(ctx, q, partnerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.StatusHistory
	for rows.Next() {
		var h domain.StatusHistory
		if err := rows.Scan(&h.ID, &h.PartnerID, &h.FromStatus, &h.ToStatus, &h.Reason, &h.ChangedBy, &h.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, nil
}
