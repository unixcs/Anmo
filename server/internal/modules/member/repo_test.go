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
