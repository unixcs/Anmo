package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"anmo/server/internal/shared"
)

// tags.go — 服务标签（D28，plan §五）：调理部位 BODY_PART / 服务方式 METHOD 两组。
// 仅商家内部数据，用户端不可见。历史 service_record 存名称快照，标签增删/停用
// 不影响历史；删除仅限从未被任何记录引用（REVERSED 也算使用过，历史可查）。

// Tag groups (frozen by ck_tag_group).
const (
	GroupBodyPart = "BODY_PART"
	GroupMethod   = "METHOD"
)

// ServiceTag — service_tag row; Used is a computed decoration (D28 删除守卫).
type ServiceTag struct {
	ID        string `json:"id"`
	TagGroup  string `json:"tag_group"`
	Name      string `json:"name"`
	Sort      int    `json:"sort"`
	Status    string `json:"status"` // ACTIVE | DISABLED
	Used      bool   `json:"used"`
	CreatedAt string `json:"created_at"`
}

const tagColumns = `id, tag_group, name, sort, status, created_at`

func scanTag(row interface{ Scan(...any) error }) (*ServiceTag, error) {
	t := &ServiceTag{}
	if err := row.Scan(&t.ID, &t.TagGroup, &t.Name, &t.Sort, &t.Status, &t.CreatedAt); err != nil {
		return nil, err
	}
	return t, nil
}

func validTagGroup(group string) bool {
	return group == GroupBodyPart || group == GroupMethod
}

// validateTagName — trim 后 1..12 字符；禁止 % _ " '（保证 body_parts LIKE 定位
// 安全与快照整洁，D28）。
func validateTagName(name string) (string, error) {
	name = strings.TrimSpace(name)
	n := utf8.RuneCountInString(name)
	if n < 1 || n > 12 {
		return "", shared.BadRequest("TAG_BAD_NAME", "标签名需 1~12 个字")
	}
	if strings.ContainsAny(name, `%_"'`) {
		return "", shared.BadRequest("TAG_BAD_NAME", "标签名不能包含 % _ 引号等字符")
	}
	return name, nil
}

// ListServiceTags — 全量（含 DISABLED，admin 标签管理页用），按 sort, created_at
// 排序，每项带 used（是否被 service_record 引用）。
func (p *Provider) ListServiceTags(ctx context.Context, group string) ([]*ServiceTag, error) {
	if group != "" && !validTagGroup(group) {
		return nil, shared.BadRequest("TAG_BAD_GROUP", "标签分组仅支持 BODY_PART/METHOD")
	}
	q := `SELECT ` + tagColumns + ` FROM service_tag`
	args := []any{}
	if group != "" {
		q += ` WHERE tag_group = ?`
		args = append(args, group)
	}
	q += ` ORDER BY sort, created_at`
	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, shared.Server("TAG_LIST", err)
	}
	defer rows.Close()
	var out []*ServiceTag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, shared.Server("TAG_SCAN", err)
		}
		out = append(out, t)
	}
	for _, t := range out {
		t.Used = p.tagUsed(ctx, t.TagGroup, t.Name)
	}
	return out, nil
}

// tagUsed — used 判定（D28）：METHOD 组按 service_method 等值；BODY_PART 组按
// body_parts LIKE '%"name"%'（名称已禁引号/通配符，安全）。REVERSED 记录也算使用过。
func (p *Provider) tagUsed(ctx context.Context, group, name string) bool {
	var used bool
	var err error
	if group == GroupMethod {
		err = p.db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM service_record WHERE service_method = ?)`, name).Scan(&used)
	} else {
		err = p.db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM service_record WHERE body_parts LIKE ?)`,
			`%"`+name+`"%`).Scan(&used)
	}
	if err != nil {
		return false
	}
	return used
}

// CreateServiceTag — 添加标签；sort = 组内 max+1；同组同名 → TAG_EXISTS 409。
func (p *Provider) CreateServiceTag(ctx context.Context, group, name string) (*ServiceTag, error) {
	if !validTagGroup(group) {
		return nil, shared.BadRequest("TAG_BAD_GROUP", "标签分组仅支持 BODY_PART/METHOD")
	}
	name, err := validateTagName(name)
	if err != nil {
		return nil, err
	}
	var maxSort sql.NullInt64
	if err := p.db.QueryRowContext(ctx,
		`SELECT MAX(sort) FROM service_tag WHERE tag_group = ?`, group).Scan(&maxSort); err != nil {
		return nil, shared.Server("TAG_QUERY", err)
	}
	t := &ServiceTag{ID: shared.NewID(), TagGroup: group, Name: name, Sort: int(maxSort.Int64) + 1, Status: "ACTIVE"}
	if _, err := p.db.ExecContext(ctx,
		`INSERT INTO service_tag (id, tag_group, name, sort) VALUES (?,?,?,?)`,
		t.ID, t.TagGroup, t.Name, t.Sort); err != nil {
		if shared.IsDupKey(err) {
			return nil, shared.Conflict("TAG_EXISTS", "该分组下已有同名标签")
		}
		return nil, shared.Server("TAG_INSERT", err)
	}
	return t, nil
}

// UpdateServiceTag — 改名 / 排序 / 停用启用（指针为 nil 的字段不更新）。
func (p *Provider) UpdateServiceTag(ctx context.Context, id string, name *string, sort *int, status *string) (*ServiceTag, error) {
	t, err := scanTag(p.db.QueryRowContext(ctx, `SELECT `+tagColumns+` FROM service_tag WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.NotFound("TAG_NOT_FOUND", "标签不存在")
	}
	if err != nil {
		return nil, shared.Server("TAG_QUERY", err)
	}
	newName := t.Name
	if name != nil {
		newName, err = validateTagName(*name)
		if err != nil {
			return nil, err
		}
	}
	newSort := t.Sort
	if sort != nil {
		newSort = *sort
	}
	newStatus := t.Status
	if status != nil {
		if *status != "ACTIVE" && *status != "DISABLED" {
			return nil, shared.BadRequest("TAG_BAD_STATUS", "状态仅支持 ACTIVE/DISABLED")
		}
		newStatus = *status
	}
	if _, err := p.db.ExecContext(ctx,
		`UPDATE service_tag SET name = ?, sort = ?, status = ? WHERE id = ?`,
		newName, newSort, newStatus, id); err != nil {
		if shared.IsDupKey(err) {
			return nil, shared.Conflict("TAG_EXISTS", "该分组下已有同名标签")
		}
		return nil, shared.Server("TAG_UPDATE", err)
	}
	return scanTag(p.db.QueryRowContext(ctx, `SELECT `+tagColumns+` FROM service_tag WHERE id = ?`, id))
}

// DeleteServiceTag — 仅未使用可删；used → TAG_IN_USE 409（停用替代删除，D28）。
func (p *Provider) DeleteServiceTag(ctx context.Context, id string) error {
	t, err := scanTag(p.db.QueryRowContext(ctx, `SELECT `+tagColumns+` FROM service_tag WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return shared.NotFound("TAG_NOT_FOUND", "标签不存在")
	}
	if err != nil {
		return shared.Server("TAG_QUERY", err)
	}
	if p.tagUsed(ctx, t.TagGroup, t.Name) {
		return shared.Conflict("TAG_IN_USE", "该标签已被服务记录使用，仅可停用")
	}
	if _, err := p.db.ExecContext(ctx, `DELETE FROM service_tag WHERE id = ?`, id); err != nil {
		return shared.Server("TAG_DELETE", err)
	}
	return nil
}

// ActiveTagNames — 录入页 chips 数据源（仅 ACTIVE，按 sort）；group 为空返回两组。
func (p *Provider) ActiveTagNames(ctx context.Context, group string) (parts, methods []string, err error) {
	q := `SELECT tag_group, name FROM service_tag WHERE status = 'ACTIVE'`
	args := []any{}
	if group != "" {
		if !validTagGroup(group) {
			return nil, nil, shared.BadRequest("TAG_BAD_GROUP", "标签分组仅支持 BODY_PART/METHOD")
		}
		q += ` AND tag_group = ?`
		args = append(args, group)
	}
	q += ` ORDER BY sort, created_at`
	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, nil, shared.Server("TAG_LIST", err)
	}
	defer rows.Close()
	for rows.Next() {
		var g, name string
		if err := rows.Scan(&g, &name); err != nil {
			return nil, nil, shared.Server("TAG_SCAN", err)
		}
		if g == GroupBodyPart {
			parts = append(parts, name)
		} else {
			methods = append(methods, name)
		}
	}
	return parts, methods, nil
}
