package service

import (
	"context"
	"testing"

	"anmo/server/internal/config"
	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

func newTestProvider(t *testing.T) *Provider {
	t.Helper()
	return New(testsupport.NewSchemaDB(t, "anmo_service_"), config.Defaults())
}

func TestCreateItemAndCatalogVisibility(t *testing.T) {
	p := newTestProvider(t)
	ctx := context.Background()

	cat, err := p.CreateCategory(ctx, "按摩", 1)
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	it, err := p.CreateItem(ctx, NewItem{
		CategoryID: cat.ID, Name: "肩颈按摩", DurationMin: 60, PriceCents: 12800,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	if it.Status != "ACTIVE" || it.PriceCents != 12800 || it.DurationMin != 60 {
		t.Fatalf("item = %+v", it)
	}

	items, _ := p.ListItems(ctx, true)
	if len(items) != 1 || items[0].Name != "肩颈按摩" {
		t.Fatalf("active items = %+v", items)
	}

	if err := p.SetItemStatus(ctx, it.ID, "INACTIVE"); err != nil {
		t.Fatalf("disable: %v", err)
	}
	items, _ = p.ListItems(ctx, true)
	if len(items) != 0 {
		t.Fatalf("disabled item still visible: %+v", items)
	}
	all, _ := p.ListItems(ctx, false)
	if len(all) != 1 {
		t.Fatalf("admin items = %+v", all)
	}
}

func TestItemValidation(t *testing.T) {
	p := newTestProvider(t)
	ctx := context.Background()
	cat, _ := p.CreateCategory(ctx, "其它", 2)

	if _, err := p.CreateItem(ctx, NewItem{CategoryID: cat.ID, Name: "坏时长", DurationMin: 0, PriceCents: 100}); !shared.Is(err, "SERVICE_BAD_DURATION") {
		t.Fatalf("want SERVICE_BAD_DURATION, got %v", err)
	}
	if _, err := p.CreateItem(ctx, NewItem{CategoryID: cat.ID, Name: "负价", DurationMin: 30, PriceCents: -1}); !shared.Is(err, "SERVICE_BAD_PRICE") {
		t.Fatalf("want SERVICE_BAD_PRICE, got %v", err)
	}
	if _, err := p.CreateItem(ctx, NewItem{CategoryID: cat.ID, Name: "", DurationMin: 30, PriceCents: 100}); !shared.Is(err, "SERVICE_NO_NAME") {
		t.Fatalf("want SERVICE_NO_NAME, got %v", err)
	}
}
