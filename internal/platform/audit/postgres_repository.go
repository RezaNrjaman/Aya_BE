package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (p *PostgresRepository) Save(ctx context.Context, r *Record) error {
	var detailsJSON []byte
	if r.Details != nil {
		detailsJSON, _ = json.Marshal(r.Details)
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}

	q := `INSERT INTO audit_logs (partner_id, user_id, user_name, action, entity_type, entity_id, details, ip_address, user_agent, created_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`
	return p.pool.QueryRow(ctx, q, r.PartnerID, r.UserID, r.UserName, r.Action, r.EntityType, r.EntityID,
		detailsJSON, r.IPAddress, r.UserAgent, r.CreatedAt).Scan(&r.ID)
}

func (p *PostgresRepository) List(ctx context.Context, partnerID *int, restricted bool, limit, offset int) ([]Record, int, error) {
	where := `WHERE ($2 = FALSE OR ($1::int IS NOT NULL AND partner_id = $1))`
	var total int
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs `+where, partnerID, restricted).Scan(&total); err != nil {
		return nil, 0, err
	}
	records, err := p.queryRecords(ctx, partnerID, restricted, limit, offset)
	return records, total, err
}

func (p *PostgresRepository) queryRecords(ctx context.Context, partnerID *int, restricted bool, limit, offset int) ([]Record, error) {
	q := `SELECT id, partner_id, user_id, user_name, action, entity_type, entity_id, details, ip_address, user_agent, created_at
	      FROM audit_logs
	      WHERE ($1::int IS NULL AND $2 = FALSE) OR ($2 = TRUE AND $1::int IS NOT NULL AND partner_id = $1)
	      ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	rows, err := p.pool.Query(ctx, q, partnerID, restricted, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		var detailsJSON []byte
		if err := rows.Scan(&r.ID, &r.PartnerID, &r.UserID, &r.UserName, &r.Action, &r.EntityType, &r.EntityID, &detailsJSON, &r.IPAddress, &r.UserAgent, &r.CreatedAt); err != nil {
			return nil, err
		}
		if len(detailsJSON) > 0 {
			_ = json.Unmarshal(detailsJSON, &r.Details)
		}
		records = append(records, r)
	}
	return records, nil
}
