package member

import (
	"context"
	"testing"

	"anmo/server/internal/config"
	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

// newTestProvider builds a provider against a fresh schema with migrations applied.
func newTestProvider(t *testing.T) (*Provider, shared.DB) {
	t.Helper()
	db := testsupport.NewSchemaDB(t, "anmo_member_")
	return New(db, config.Defaults()), db
}

func TestCreateAndPhoneConflict(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	m, err := p.Create(ctx, NewMember{Name: "张三", Phone: "13911112222", Gender: "女"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if m.MemberNo == "" || len(m.MemberNo) < 9 {
		t.Fatalf("member_no missing: %+v", m)
	}
	if _, err := p.Create(ctx, NewMember{Name: "李四", Phone: "13911112222"}); !shared.Is(err, "MEMBER_PHONE_EXISTS") {
		t.Fatalf("want MEMBER_PHONE_EXISTS, got %v", err)
	}
}

func TestEnsureByPhoneBindsSameMember(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	var id1, id2 string
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		var e error
		id1, _, e = p.EnsureByPhone(ctx, tx, "13911113333", "")
		return e
	})
	if err != nil {
		t.Fatalf("ensure1: %v", err)
	}
	err = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		var e error
		id2, _, e = p.EnsureByPhone(ctx, tx, "13911113333", "")
		return e
	})
	if err != nil {
		t.Fatalf("ensure2: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("second login bound different member: %s vs %s", id1, id2)
	}
}

func TestTagsSetAndReplace(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	t1, err := p.CreateTag(ctx, "测试标签A")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if _, err := p.CreateTag(ctx, "测试标签A"); !shared.Is(err, "TAG_EXISTS") {
		t.Fatalf("want TAG_EXISTS, got %v", err)
	}
	m, _ := p.Create(ctx, NewMember{Name: "王五", Phone: "13911114444"})
	if err := p.SetTags(ctx, m.ID, []string{t1.ID}); err != nil {
		t.Fatalf("set tags: %v", err)
	}
	tags, _ := p.TagsOf(ctx, m.ID)
	if len(tags) != 1 || tags[0].Name != "测试标签A" {
		t.Fatalf("tags = %+v", tags)
	}
	// replace with empty
	if err := p.SetTags(ctx, m.ID, nil); err != nil {
		t.Fatalf("clear tags: %v", err)
	}
	tags, _ = p.TagsOf(ctx, m.ID)
	if len(tags) != 0 {
		t.Fatalf("tags after clear = %+v", tags)
	}
}

func TestUpdateProfileValidation(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()
	m, _ := p.Create(ctx, NewMember{Name: "赵六", Phone: "13911115555"})

	if _, err := p.UpdateProfile(ctx, m.ID, ProfileUpdate{Gender: strPtr("未知")}); !shared.Is(err, "MEMBER_BAD_GENDER") {
		t.Fatalf("want MEMBER_BAD_GENDER, got %v", err)
	}
	if _, err := p.UpdateProfile(ctx, m.ID, ProfileUpdate{Birthday: strPtr("2026/01/01")}); !shared.Is(err, "MEMBER_BAD_BIRTHDAY") {
		t.Fatalf("want MEMBER_BAD_BIRTHDAY, got %v", err)
	}
	got, err := p.UpdateProfile(ctx, m.ID, ProfileUpdate{Name: strPtr("赵六六"), Gender: strPtr("男"), Birthday: strPtr("1990-01-01")})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Name != "赵六六" || got.Gender != "男" || got.Birthday == nil || *got.Birthday != "1990-01-01" {
		t.Fatalf("updated = %+v", got)
	}
}

func strPtr(s string) *string { return &s }

// ---- V2.2 第二批：密码体系（design §2/§4）----

// 迁移 014：phone 可空后两个纯微信会员（phone NULL）并存不触发唯一冲突；
// 非空 phone 重复仍被拒（uk_member_phone 对 NULL 天然放行）。
func TestMigr014NullPhoneMembersCoexist(t *testing.T) {
	p, db := newTestProvider(t)
	ctx := context.Background()

	var id1, id2 string
	err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		var e error
		id1, e = p.CreateByOpenID(ctx, tx, "o-null-1")
		if e != nil {
			return e
		}
		id2, e = p.CreateByOpenID(ctx, tx, "o-null-2")
		return e
	})
	if err != nil {
		t.Fatalf("two null-phone members: %v", err)
	}
	if id1 == id2 || id1 == "" || id2 == "" {
		t.Fatalf("ids = %s / %s", id1, id2)
	}

	// 旧数据无损断言：建库后插入的 member（含 wx_openid、无密码）迁移后仍在——
	// testsupport 建库即应用全迁移，这里以行字段回读等价验证。
	m, err := p.Get(ctx, id1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if m.Phone != "" || m.WxBound != true || m.HasPassword {
		t.Fatalf("null-phone member = %+v", m)
	}
	if m.MemberNo == "" {
		t.Fatal("member_no missing")
	}
}

func TestCreateByOpenIDGeneratesMemberNo(t *testing.T) {
	p, db := newTestProvider(t)
	ctx := context.Background()
	var id string
	err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		var e error
		id, e = p.CreateByOpenID(ctx, tx, "o-abc")
		return e
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	m, _ := p.Get(ctx, id)
	if m.MemberNo == "" || m.Name != "" || m.Phone != "" {
		t.Fatalf("by-openid member = %+v", m)
	}
	if !m.WxBound {
		t.Fatal("wx_bound should be true")
	}
	// 空 openid 拒绝
	if err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		_, e := p.CreateByOpenID(ctx, tx, "  ")
		return e
	}); !shared.Is(err, "MEMBER_BAD_OPENID") {
		t.Fatalf("want MEMBER_BAD_OPENID, got %v", err)
	}
}

func TestCreateWithPhoneAndSetPassword(t *testing.T) {
	p, db := newTestProvider(t)
	ctx := context.Background()

	var id string
	err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		var e error
		id, e = p.CreateWithPhone(ctx, tx, "13922220001", "secret66")
		return e
	})
	if err != nil {
		t.Fatalf("create with phone: %v", err)
	}
	m, _ := p.Get(ctx, id)
	if !m.HasPassword || m.Phone != "13922220001" || m.PasswordHash == "" {
		t.Fatalf("member = %+v", m)
	}
	if m.WxBound {
		t.Fatal("wx_bound should be false")
	}

	// 坏手机号 / 弱密码
	if err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		_, e := p.CreateWithPhone(ctx, tx, "12345", "secret66")
		return e
	}); !shared.Is(err, "MEMBER_BAD_PHONE") {
		t.Fatalf("want MEMBER_BAD_PHONE, got %v", err)
	}
	if err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		_, e := p.CreateWithPhone(ctx, tx, "13922220002", "12345")
		return e
	}); !shared.Is(err, "MEMBER_WEAK_PASSWORD") {
		t.Fatalf("want MEMBER_WEAK_PASSWORD, got %v", err)
	}

	// SetPassword：幂等覆盖；空手机号会员被拒（MEMBER_PHONE_REQUIRED）
	err = shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		return p.SetPassword(ctx, tx, id, "newpass88")
	})
	if err != nil {
		t.Fatalf("set password: %v", err)
	}
	after, _ := p.Get(ctx, id)
	if !after.HasPassword || after.PasswordHash == m.PasswordHash {
		t.Fatal("password not overwritten")
	}

	var shellID string
	_ = shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		var e error
		shellID, e = p.CreateByOpenID(ctx, tx, "o-shell")
		return e
	})
	if err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		return p.SetPassword(ctx, tx, shellID, "newpass88")
	}); !shared.Is(err, "MEMBER_PHONE_REQUIRED") {
		t.Fatalf("want MEMBER_PHONE_REQUIRED, got %v", err)
	}
}

func TestSetPhoneOnce(t *testing.T) {
	p, db := newTestProvider(t)
	ctx := context.Background()

	var shellID string
	_ = shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		var e error
		shellID, e = p.CreateByOpenID(ctx, tx, "o-claim-1")
		if e != nil {
			return e
		}
		_, e = p.CreateWithPhone(ctx, tx, "13933330001", "oldpass1")
		return e
	})

	// 空 → 成功
	if err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		return p.SetPhoneOnce(ctx, tx, shellID, "13933330002")
	}); err != nil {
		t.Fatalf("set phone: %v", err)
	}
	m, _ := p.Get(ctx, shellID)
	if m.Phone != "13933330002" {
		t.Fatalf("phone = %s", m.Phone)
	}

	// 已有 → 拒（一次性）
	if err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		return p.SetPhoneOnce(ctx, tx, shellID, "13933330003")
	}); !shared.Is(err, "MEMBER_PHONE_SET") {
		t.Fatalf("want MEMBER_PHONE_SET, got %v", err)
	}

	// 撞号 → 409 MEMBER_PHONE_TAKEN
	var otherID string
	_ = shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		var e error
		otherID, e = p.CreateByOpenID(ctx, tx, "o-claim-2")
		return e
	})
	if err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		return p.SetPhoneOnce(ctx, tx, otherID, "13933330001")
	}); !shared.Is(err, "MEMBER_PHONE_TAKEN") {
		t.Fatalf("want MEMBER_PHONE_TAKEN, got %v", err)
	}
}

func TestHasBusinessDataAndDeleteShell(t *testing.T) {
	p, db := newTestProvider(t)
	ctx := context.Background()

	var shellID string
	_ = shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		var e error
		shellID, e = p.CreateByOpenID(ctx, tx, "o-shell-del")
		return e
	})

	has := false
	err := shared.RunInTx(ctx, db, func(tx shared.Tx) (e error) {
		has, e = p.HasBusinessData(ctx, tx, shellID)
		return e
	})
	if err != nil || has {
		t.Fatalf("fresh shell has business data: %v %v", has, err)
	}

	// 打业务标记（直接插最小列集 appointment 行，验证 EXISTS 口径）
	_, _ = db.ExecContext(ctx,
		`INSERT INTO appointment (id, appointment_no, member_id, scheduled_start, scheduled_end)
		 VALUES ('01JAPT0000000000000000T1', 'APT202601010001', ?, '2026-01-01 10:00', '2026-01-01 11:00')`,
		shellID)

	has = false
	err = shared.RunInTx(ctx, db, func(tx shared.Tx) (e error) {
		has, e = p.HasBusinessData(ctx, tx, shellID)
		return e
	})
	if err != nil || !has {
		t.Fatalf("appointment should count as business data: %v %v", has, err)
	}

	// 有数据 → 拒删；清掉后 → 删除成功且行消失
	err = shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		return p.DeleteShell(ctx, tx, shellID)
	})
	if !shared.Is(err, "MEMBER_HAS_DATA") {
		t.Fatalf("want MEMBER_HAS_DATA, got %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM appointment WHERE id = '01JAPT0000000000000000T1'`); err != nil {
		t.Fatal(err)
	}
	if err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		return p.DeleteShell(ctx, tx, shellID)
	}); err != nil {
		t.Fatalf("delete shell: %v", err)
	}
	if _, err := p.Get(ctx, shellID); !shared.Is(err, "MEMBER_NOT_FOUND") {
		t.Fatalf("want MEMBER_NOT_FOUND, got %v", err)
	}
}

func TestAdminSetPassword(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()
	m, err := p.Create(ctx, NewMember{Name: "重置对象", Phone: "13944440001"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if m.HasPassword {
		t.Fatal("fresh member should have no password")
	}
	// 弱密码拒绝
	if err := p.AdminSetPassword(ctx, m.ID, "123"); !shared.Is(err, "MEMBER_WEAK_PASSWORD") {
		t.Fatalf("want MEMBER_WEAK_PASSWORD, got %v", err)
	}
	// 设置成功（无 phone 要求不适用：admin 建号必有 phone；纯微信会员也允许设）
	if err := p.AdminSetPassword(ctx, m.ID, "adminpass1"); err != nil {
		t.Fatalf("admin set: %v", err)
	}
	got, _ := p.Get(ctx, m.ID)
	if !got.HasPassword {
		t.Fatal("password missing after admin reset")
	}
	// 不存在会员
	if err := p.AdminSetPassword(ctx, "01JNOPE0000000000000000NP", "adminpass1"); !shared.Is(err, "MEMBER_NOT_FOUND") {
		t.Fatalf("want MEMBER_NOT_FOUND, got %v", err)
	}
}

// profile 更新入参扩展 phone：空手机号会员可经 UpdateProfile 一次性设置；
// 已有手机号的会员被 MEMBER_PHONE_SET 拒绝。
func TestUpdateProfileSetsPhoneOnce(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	var shellID string
	_ = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		var e error
		shellID, e = p.CreateByOpenID(ctx, tx, "o-profile-phone")
		return e
	})
	got, err := p.UpdateProfile(ctx, shellID, ProfileUpdate{
		Name: strPtr("微信客"), Phone: strPtr("13955550001"),
	})
	if err != nil {
		t.Fatalf("update profile with phone: %v", err)
	}
	if got.Name != "微信客" || got.Phone != "13955550001" {
		t.Fatalf("updated = %+v", got)
	}
	if _, err := p.UpdateProfile(ctx, shellID, ProfileUpdate{Phone: strPtr("13955550002")}); !shared.Is(err, "MEMBER_PHONE_SET") {
		t.Fatalf("want MEMBER_PHONE_SET, got %v", err)
	}
}

// §32 组合搜索：tag_id 过滤 + 列表标签装饰。
func TestListFiltersByTag(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	a, err := p.Create(ctx, NewMember{Name: "张三", Phone: "13911110001"})
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	if _, err := p.Create(ctx, NewMember{Name: "李四", Phone: "13911110002"}); err != nil {
		t.Fatalf("create B: %v", err)
	}

	tags, err := p.ListTags(ctx)
	if err != nil {
		t.Fatalf("tags: %v", err)
	}
	if err := p.SetTags(ctx, a.ID, []string{tags[0].ID}); err != nil {
		t.Fatalf("set tags: %v", err)
	}

	// tag 过滤：只命中 A；列表行带标签装饰
	onlyTagged, total, err := p.List(ctx, ListParams{TagID: tags[0].ID, Page: shared.PageParams{Page: 1, PerPage: 20}})
	if err != nil {
		t.Fatalf("list by tag: %v", err)
	}
	if total != 1 || len(onlyTagged) != 1 || onlyTagged[0].ID != a.ID {
		t.Fatalf("tag filter = %d rows, want only A", total)
	}
	if len(onlyTagged[0].Tags) == 0 {
		t.Fatal("list row should carry decorated tags")
	}

	// tag + keyword 组合：keyword 不匹配 → 空
	_, total, err = p.List(ctx, ListParams{Keyword: "不存在", TagID: tags[0].ID, Page: shared.PageParams{Page: 1, PerPage: 20}})
	if err != nil {
		t.Fatalf("combined list: %v", err)
	}
	if total != 0 {
		t.Fatalf("combined filter total = %d, want 0", total)
	}
}
