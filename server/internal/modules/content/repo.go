package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"anmo/server/internal/shared"
)

// Banner — content_banner row.
type Banner struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Image  string `json:"image"`
	Link   string `json:"link"`
	Sort   int    `json:"sort"`
	Status string `json:"status"`
}

// Announcement — content_announcement row.
type Announcement struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Sort    int    `json:"sort"`
	Status  string `json:"status"`
}

// PageBlock — one JSON block inside content_page_config (§83). D16: blocks
// reference banner/announcement IDs; the body is not copied here.
type PageBlock struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

// --- home (customer) ---

// Home assembles the customer homepage payload (§105).
func (p *Provider) Home(ctx context.Context) (map[string]any, error) {
	blocks := []PageBlock{}
	var raw []byte
	err := p.db.QueryRowContext(ctx, `SELECT blocks FROM content_page_config WHERE page = 'home'`).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// default layout until the owner customizes it
		blocks = []PageBlock{
			{Type: "banner", Data: map[string]any{}},
			{Type: "announcement", Data: map[string]any{}},
			{Type: "service_list", Data: map[string]any{}},
		}
	case err == nil:
		if err := json.Unmarshal(raw, &blocks); err != nil {
			return nil, shared.Server("CONTENT_PARSE", err)
		}
	default:
		return nil, shared.Server("CONTENT_QUERY", err)
	}

	banners, err := p.listBanners(ctx, true)
	if err != nil {
		return nil, err
	}
	announcements, err := p.listAnnouncements(ctx, true)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"blocks":        blocks,
		"banners":       banners,
		"announcements": announcements,
	}, nil
}

// --- banners ---

func (p *Provider) listBanners(ctx context.Context, activeOnly bool) ([]*Banner, error) {
	q := `SELECT id, title, image, link, sort, status FROM content_banner`
	if activeOnly {
		q += ` WHERE status = 'ACTIVE'`
	}
	q += ` ORDER BY sort`
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, shared.Server("BANNER_LIST", err)
	}
	defer rows.Close()
	var out []*Banner
	for rows.Next() {
		b := &Banner{}
		if err := rows.Scan(&b.ID, &b.Title, &b.Image, &b.Link, &b.Sort, &b.Status); err != nil {
			return nil, shared.Server("BANNER_SCAN", err)
		}
		out = append(out, b)
	}
	return out, nil
}

// SaveBanner upserts one banner (id empty = create).
func (p *Provider) SaveBanner(ctx context.Context, b *Banner) error {
	if b.ID == "" {
		b.ID = shared.NewID()
		b.Status = "ACTIVE"
		if _, err := p.db.ExecContext(ctx,
			`INSERT INTO content_banner (id, title, image, link, sort) VALUES (?,?,?,?,?)`,
			b.ID, b.Title, b.Image, b.Link, b.Sort); err != nil {
			return shared.Server("BANNER_INSERT", err)
		}
		return nil
	}
	if _, err := p.db.ExecContext(ctx,
		`UPDATE content_banner SET title=?, image=?, link=?, sort=? WHERE id=?`,
		b.Title, b.Image, b.Link, b.Sort, b.ID); err != nil {
		return shared.Server("BANNER_UPDATE", err)
	}
	return nil
}

// SetBannerStatus toggles a banner.
func (p *Provider) SetBannerStatus(ctx context.Context, id, status string) error {
	if status != "ACTIVE" && status != "INACTIVE" {
		return shared.BadRequest("BANNER_BAD_STATUS", "状态仅支持 ACTIVE/INACTIVE")
	}
	_, err := p.db.ExecContext(ctx, `UPDATE content_banner SET status=? WHERE id=?`, status, id)
	if err != nil {
		return shared.Server("BANNER_STATUS", err)
	}
	return nil
}

// --- announcements ---

func (p *Provider) listAnnouncements(ctx context.Context, activeOnly bool) ([]*Announcement, error) {
	q := `SELECT id, title, content, sort, status FROM content_announcement`
	if activeOnly {
		q += ` WHERE status = 'ACTIVE'`
	}
	q += ` ORDER BY sort`
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, shared.Server("ANN_LIST", err)
	}
	defer rows.Close()
	var out []*Announcement
	for rows.Next() {
		a := &Announcement{}
		if err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.Sort, &a.Status); err != nil {
			return nil, shared.Server("ANN_SCAN", err)
		}
		out = append(out, a)
	}
	return out, nil
}

// SaveAnnouncement upserts one announcement.
func (p *Provider) SaveAnnouncement(ctx context.Context, a *Announcement) error {
	if a.ID == "" {
		a.ID = shared.NewID()
		a.Status = "ACTIVE"
		if _, err := p.db.ExecContext(ctx,
			`INSERT INTO content_announcement (id, title, content, sort) VALUES (?,?,?,?)`,
			a.ID, a.Title, a.Content, a.Sort); err != nil {
			return shared.Server("ANN_INSERT", err)
		}
		return nil
	}
	if _, err := p.db.ExecContext(ctx,
		`UPDATE content_announcement SET title=?, content=?, sort=? WHERE id=?`,
		a.Title, a.Content, a.Sort, a.ID); err != nil {
		return shared.Server("ANN_UPDATE", err)
	}
	return nil
}

// SetAnnouncementStatus toggles an announcement.
func (p *Provider) SetAnnouncementStatus(ctx context.Context, id, status string) error {
	if status != "ACTIVE" && status != "INACTIVE" {
		return shared.BadRequest("ANN_BAD_STATUS", "状态仅支持 ACTIVE/INACTIVE")
	}
	_, err := p.db.ExecContext(ctx, `UPDATE content_announcement SET status=? WHERE id=?`, status, id)
	if err != nil {
		return shared.Server("ANN_STATUS", err)
	}
	return nil
}

// --- page config (admin) ---

// GetPageConfig returns the raw blocks for a page.
func (p *Provider) GetPageConfig(ctx context.Context, page string) ([]PageBlock, error) {
	var raw []byte
	err := p.db.QueryRowContext(ctx, `SELECT blocks FROM content_page_config WHERE page = ?`, page).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return []PageBlock{}, nil
	}
	if err != nil {
		return nil, shared.Server("CONTENT_QUERY", err)
	}
	var blocks []PageBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, shared.Server("CONTENT_PARSE", err)
	}
	return blocks, nil
}

// SavePageConfig validates and stores the block list for a page (D16).
func (p *Provider) SavePageConfig(ctx context.Context, page string, blocks []PageBlock, operatorID string) error {
	if page == "" {
		return shared.BadRequest("CONTENT_BAD_PAGE", "页面标识不能为空")
	}
	for _, b := range blocks {
		switch b.Type {
		case "banner", "announcement", "service_list", "activity", "richtext":
		default:
			return shared.BadRequest("CONTENT_BAD_BLOCK", "未知的区块类型 "+b.Type)
		}
	}
	raw, err := json.Marshal(blocks)
	if err != nil {
		return shared.Server("CONTENT_ENCODE", err)
	}
	_, err = p.db.ExecContext(ctx,
		`INSERT INTO content_page_config (id, page, blocks, updated_by) VALUES (?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE blocks = VALUES(blocks), updated_by = VALUES(updated_by)`,
		shared.NewID(), page, raw, operatorID)
	if err != nil {
		return shared.Server("CONTENT_SAVE", err)
	}
	return nil
}

// --- system settings (§84) ---

// Settings returns all settings as a map.
func (p *Provider) Settings(ctx context.Context) (map[string]string, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT setting_key, setting_value FROM content_system_setting`)
	if err != nil {
		return nil, shared.Server("SETTING_LIST", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, shared.Server("SETTING_SCAN", err)
		}
		out[k] = v
	}
	return out, nil
}

// PublicSettings exposes only customer-safe keys with config fallbacks.
func (p *Provider) PublicSettings(ctx context.Context) (map[string]string, error) {
	stored, err := p.Settings(ctx)
	if err != nil {
		return nil, err
	}
	b := p.cfg.Business
	out := map[string]string{
		"open_time":   b.OpenTime,
		"close_time":  b.CloseTime,
		"shop_phone":  stored["shop_phone"],
		"shop_notice": stored["shop_notice"],
	}
	for _, k := range []string{"open_time", "close_time", "shop_phone", "shop_notice"} {
		if v, ok := stored[k]; ok && v != "" {
			out[k] = v
		}
	}
	return out, nil
}

// SaveSetting upserts one setting.
func (p *Provider) SaveSetting(ctx context.Context, key, value, operatorID string) error {
	if key == "" {
		return shared.BadRequest("SETTING_BAD_KEY", "设置项不能为空")
	}
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO content_system_setting (id, setting_key, setting_value, updated_by, updated_at)
		 VALUES (?, ?, ?, ?, NOW())
		 ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value), updated_by = VALUES(updated_by)`,
		shared.NewID(), key, value, operatorID)
	if err != nil {
		return shared.Server("SETTING_SAVE", err)
	}
	return nil
}
