package member

import (
	"context"
	"database/sql"
	"errors"

	"anmo/server/internal/shared"
)

// tags.go — simple tag management (plan §13: no rule engine).

// ListTags returns all tags.
func (p *Provider) ListTags(ctx context.Context) ([]*Tag, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id, name FROM member_tag ORDER BY created_at`)
	if err != nil {
		return nil, shared.Server("TAG_LIST", err)
	}
	defer rows.Close()
	var out []*Tag
	for rows.Next() {
		t := &Tag{}
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, shared.Server("TAG_SCAN", err)
		}
		out = append(out, t)
	}
	return out, nil
}

// CreateTag adds a new tag (name unique).
func (p *Provider) CreateTag(ctx context.Context, name string) (*Tag, error) {
	if name == "" {
		return nil, shared.BadRequest("TAG_EMPTY", "标签名不能为空")
	}
	t := &Tag{ID: shared.NewID(), Name: name}
	_, err := p.db.ExecContext(ctx, `INSERT INTO member_tag (id, name) VALUES (?, ?)`, t.ID, t.Name)
	if err != nil {
		return nil, shared.Conflict("TAG_EXISTS", "标签已存在")
	}
	return t, nil
}

// RenameTag renames a tag (goal §32).
func (p *Provider) RenameTag(ctx context.Context, id, name string) error {
	if name == "" {
		return shared.BadRequest("TAG_EMPTY", "标签名不能为空")
	}
	res, err := p.db.ExecContext(ctx,
		`UPDATE member_tag SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return shared.Conflict("TAG_EXISTS", "标签已存在")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return shared.NotFound("TAG_NOT_FOUND", "标签不存在")
	}
	return nil
}

// DeleteTag removes a tag; refused while any member still uses it (goal §32).
func (p *Provider) DeleteTag(ctx context.Context, id string) error {
	return shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		var n int64
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM member_tag_rel WHERE tag_id = ?`, id).Scan(&n); err != nil {
			return shared.Server("TAG_COUNT", err)
		}
		if n > 0 {
			return shared.Conflict("TAG_IN_USE", "该标签仍被会员使用，请先移除会员标签")
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM member_tag WHERE id = ?`, id)
		if err != nil {
			return shared.Server("TAG_DELETE", err)
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return shared.NotFound("TAG_NOT_FOUND", "标签不存在")
		}
		return nil
	})
}

// TagsOf returns the tags attached to a member.
func (p *Provider) TagsOf(ctx context.Context, memberID string) ([]*Tag, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT t.id, t.name FROM member_tag_rel r JOIN member_tag t ON t.id = r.tag_id
		 WHERE r.member_id = ? ORDER BY t.name`, memberID)
	if err != nil {
		return nil, shared.Server("TAG_QUERY", err)
	}
	defer rows.Close()
	var out []*Tag
	for rows.Next() {
		t := &Tag{}
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, shared.Server("TAG_SCAN", err)
		}
		out = append(out, t)
	}
	return out, nil
}

// SetTags replaces the tag set of a member (idempotent full replace).
func (p *Provider) SetTags(ctx context.Context, memberID string, tagIDs []string) error {
	return shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM member_tag_rel WHERE member_id = ?`, memberID); err != nil {
			return shared.Server("TAG_CLEAR", err)
		}
		for _, tid := range tagIDs {
			if _, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO member_tag_rel (id, member_id, tag_id) VALUES (?, ?, ?)`,
				shared.NewID(), memberID, tid); err != nil {
				return shared.Server("TAG_SET", err)
			}
		}
		return nil
	})
}

var _ = errors.Is
var _ = sql.ErrNoRows
