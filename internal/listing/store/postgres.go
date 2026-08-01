// Package store contains the Postgres implementation of listing.Repository,
// built on pgx/pgxpool.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LilianMrt/go-listings-service/internal/listing"
)

// Postgres persists listings in PostgreSQL.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres returns a repository backed by the given pool.
func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

// Compile-time check that Postgres satisfies the domain contract.
var _ listing.Repository = (*Postgres)(nil)

const columns = `id, title, description, price_cents, currency, city, postal_code,
	surface_m2, rooms, status, seller_id, created_at, updated_at`

func (s *Postgres) Create(ctx context.Context, l *listing.Listing) error {
	const q = `INSERT INTO listings (` + columns + `)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::listing_status,$11,$12,$13)`
	_, err := s.pool.Exec(ctx, q,
		l.ID, l.Title, l.Description, l.PriceCents, l.Currency, l.City, l.PostalCode,
		l.SurfaceM2, l.Rooms, l.Status, l.SellerID, l.CreatedAt, l.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create listing: %w", err)
	}
	return nil
}

func (s *Postgres) GetByID(ctx context.Context, id uuid.UUID) (*listing.Listing, error) {
	const q = `SELECT ` + columns + ` FROM listings WHERE id = $1`
	l, err := scanListing(s.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, listing.ErrNotFound
		}
		return nil, fmt.Errorf("get listing: %w", err)
	}
	return l, nil
}

func (s *Postgres) List(ctx context.Context, f listing.ListFilter) ([]listing.Listing, error) {
	var (
		conds []string
		args  []any
	)
	add := func(tmpl string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(tmpl, len(args)))
	}
	if f.City != nil {
		add("city = $%d", *f.City)
	}
	if f.Status != nil {
		add("status = $%d::listing_status", *f.Status)
	}
	if f.MinPrice != nil {
		add("price_cents >= $%d", *f.MinPrice)
	}
	if f.MaxPrice != nil {
		add("price_cents <= $%d", *f.MaxPrice)
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	q := fmt.Sprintf(`SELECT %s FROM listings %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		columns, where, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list listings: %w", err)
	}
	defer rows.Close()

	out := make([]listing.Listing, 0, limit)
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil {
			return nil, fmt.Errorf("scan listing: %w", err)
		}
		out = append(out, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate listings: %w", err)
	}
	return out, nil
}

func (s *Postgres) Update(ctx context.Context, l *listing.Listing) error {
	const q = `UPDATE listings SET
		title=$2, description=$3, price_cents=$4, currency=$5, city=$6, postal_code=$7,
		surface_m2=$8, rooms=$9, status=$10::listing_status, seller_id=$11, updated_at=$12
		WHERE id=$1`
	tag, err := s.pool.Exec(ctx, q,
		l.ID, l.Title, l.Description, l.PriceCents, l.Currency, l.City, l.PostalCode,
		l.SurfaceM2, l.Rooms, l.Status, l.SellerID, l.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update listing: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return listing.ErrNotFound
	}
	return nil
}

func (s *Postgres) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM listings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete listing: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return listing.ErrNotFound
	}
	return nil
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanListing(row scanner) (*listing.Listing, error) {
	var l listing.Listing
	err := row.Scan(
		&l.ID, &l.Title, &l.Description, &l.PriceCents, &l.Currency, &l.City,
		&l.PostalCode, &l.SurfaceM2, &l.Rooms, &l.Status, &l.SellerID,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}
