package service

import (
	"context"
	"database/sql"
	"errors"

	"anmo/server/internal/shared"
)

// Category — service_category row.
type Category struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Sort   int    `json:"sort"`
	Status string `json:"status"`
}

// Item — service row (§16). default_price in integer cents.
type Item struct {
	ID          string `json:"id"`
	CategoryID  string `json:"category_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DurationMin int    `json:"duration_minutes"`
	PriceCents  int64  `json:"default_price"`
	CoverImage  string `json:"cover_image"`
	Status      string `json:"status"`
	Sort        int    `json:"sort"`
}

const itemColumns = `id, category_id, name, description, duration_minutes, default_price, cover_image, status, sort`

func scanItem(row interface{ Scan(...any) error }) (*Item, error) {
	s := &Item{}
	err := row.Scan(&s.ID, &s.CategoryID, &s.Name, &s.Description, &s.DurationMin,
		&s.PriceCents, &s.CoverImage, &s.Status, &s.Sort)
	return s, err
}

// --- categories ---

func (p *Provider) CreateCategory(ctx context.Context, name string, sort int) (*Category, error) {
	if name == "" {
		return nil, shared.BadRequest("CATEGORY_EMPTY", "分类名不能为空")
	}
	c := &Category{ID: shared.NewID(), Name: name, Sort: sort, Status: "ACTIVE"}
	if _, err := p.db.ExecContext(ctx,
		`INSERT INTO service_category (id, name, sort) VALUES (?,?,?)`, c.ID, c.Name, c.Sort); err != nil {
		return nil, shared.Server("CATEGORY_INSERT", err)
	}
	return c, nil
}

func (p *Provider) ListCategories(ctx context.Context, activeOnly bool) ([]*Category, error) {
	q := `SELECT id, name, sort, status FROM service_category`
	if activeOnly {
		q += ` WHERE status = 'ACTIVE'`
	}
	q += ` ORDER BY sort, created_at`
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, shared.Server("CATEGORY_LIST", err)
	}
	defer rows.Close()
	var out []*Category
	for rows.Next() {
		c := &Category{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Sort, &c.Status); err != nil {
			return nil, shared.Server("CATEGORY_SCAN", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// SetCategoryStatus enables/disables a category.
func (p *Provider) SetCategoryStatus(ctx context.Context, id, status string) error {
	if status != "ACTIVE" && status != "INACTIVE" {
		return shared.BadRequest("CATEGORY_BAD_STATUS", "状态仅支持 ACTIVE/INACTIVE")
	}
	_, err := p.db.ExecContext(ctx,
		`UPDATE service_category SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return shared.Server("CATEGORY_UPDATE", err)
	}
	return nil
}

// --- service items ---

type NewItem struct {
	CategoryID  string `json:"category_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DurationMin int    `json:"duration_minutes"`
	PriceCents  int64  `json:"default_price"`
	CoverImage  string `json:"cover_image"`
	Sort        int    `json:"sort"`
}

func (p *Provider) CreateItem(ctx context.Context, in NewItem) (*Item, error) {
	if in.Name == "" {
		return nil, shared.BadRequest("SERVICE_NO_NAME", "服务项目名称必填")
	}
	if in.DurationMin <= 0 {
		return nil, shared.BadRequest("SERVICE_BAD_DURATION", "时长必须大于 0 分钟")
	}
	if in.PriceCents < 0 {
		return nil, shared.BadRequest("SERVICE_BAD_PRICE", "价格不能为负")
	}
	it := &Item{
		ID: shared.NewID(), CategoryID: in.CategoryID, Name: in.Name,
		Description: in.Description, DurationMin: in.DurationMin,
		PriceCents: in.PriceCents, CoverImage: in.CoverImage, Status: "ACTIVE", Sort: in.Sort,
	}
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO service (id, category_id, name, description, duration_minutes, default_price, cover_image, sort)
		 VALUES (?,?,?,?,?,?,?,?)`,
		it.ID, it.CategoryID, it.Name, it.Description, it.DurationMin, it.PriceCents, it.CoverImage, it.Sort)
	if err != nil {
		return nil, shared.Server("SERVICE_INSERT", err)
	}
	return it, nil
}

// UpdateItem edits an existing service item.
func (p *Provider) UpdateItem(ctx context.Context, id string, in NewItem) (*Item, error) {
	if in.DurationMin <= 0 {
		return nil, shared.BadRequest("SERVICE_BAD_DURATION", "时长必须大于 0 分钟")
	}
	if in.PriceCents < 0 {
		return nil, shared.BadRequest("SERVICE_BAD_PRICE", "价格不能为负")
	}
	res, err := p.db.ExecContext(ctx,
		`UPDATE service SET category_id=?, name=?, description=?, duration_minutes=?, default_price=?, cover_image=?, sort=?
		 WHERE id = ?`,
		in.CategoryID, in.Name, in.Description, in.DurationMin, in.PriceCents, in.CoverImage, in.Sort, id)
	if err != nil {
		return nil, shared.Server("SERVICE_UPDATE", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var one string
		if err := p.db.QueryRowContext(ctx, `SELECT id FROM service WHERE id = ?`, id).Scan(&one); errors.Is(err, sql.ErrNoRows) {
			return nil, shared.NotFound("SERVICE_NOT_FOUND", "服务项目不存在")
		}
	}
	return p.GetItem(ctx, id)
}

// SetItemStatus enables/disables a service item.
func (p *Provider) SetItemStatus(ctx context.Context, id, status string) error {
	if status != "ACTIVE" && status != "INACTIVE" {
		return shared.BadRequest("SERVICE_BAD_STATUS", "状态仅支持 ACTIVE/INACTIVE")
	}
	_, err := p.db.ExecContext(ctx, `UPDATE service SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return shared.Server("SERVICE_UPDATE", err)
	}
	return nil
}

// GetItem returns one service item.
func (p *Provider) GetItem(ctx context.Context, id string) (*Item, error) {
	it, err := scanItem(p.db.QueryRowContext(ctx,
		`SELECT `+itemColumns+` FROM service WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.NotFound("SERVICE_NOT_FOUND", "服务项目不存在")
	}
	if err != nil {
		return nil, shared.Server("SERVICE_QUERY", err)
	}
	return it, nil
}

// ListItems returns items; activeOnly=true for the customer H5.
func (p *Provider) ListItems(ctx context.Context, activeOnly bool) ([]*Item, error) {
	q := `SELECT ` + itemColumns + ` FROM service`
	if activeOnly {
		q += ` WHERE status = 'ACTIVE'`
	}
	q += ` ORDER BY sort, created_at`
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, shared.Server("SERVICE_LIST", err)
	}
	defer rows.Close()
	var out []*Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, shared.Server("SERVICE_SCAN", err)
		}
		out = append(out, it)
	}
	return out, nil
}
