package gorm

import (
	"testing"
	"time"

	"github.com/agnes-image-tool/backend/internal/model"
)

func TestAssetRepository(t *testing.T) {
	db := openDBMemory(t)
	if err := db.AutoMigrate(&model.Asset{}); err != nil {
		t.Fatalf("AutoMigrate Asset failed: %v", err)
	}
	repo := NewAssetRepository(db)
	now := time.Now().Format("2006-01-02 15:04:05")

	// Insert
	asset := &model.Asset{
		Mode:        "text2image",
		Prompt:      "测试提示词",
		Type:        "image",
		Time:        now,
		Favorite:    false,
		OriginalURL: "/outputs/test.png",
		LocalPath:   "outputs/test.png",
	}
	id, err := repo.Insert(asset)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if id == 0 {
		t.Fatal("Insert returned zero id")
	}

	// List
	_, total, err := repo.List(1, 20, "", "", false)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total < 1 {
		t.Fatal("List returned zero total")
	}

	// ToggleFavorite
	if err := repo.ToggleFavorite(id, true); err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}

	// GetByIDs
	assets, err := repo.GetByIDs([]int64{id})
	if err != nil {
		t.Fatalf("GetByIDs failed: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("GetByIDs returned unexpected count: %d", len(assets))
	}
	if !assets[0].Favorite {
		t.Fatal("Expected favorite=true")
	}

	// Delete
	if err := repo.Delete([]int64{id}); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	after, _, _ := repo.List(1, 20, "", "", false)
	if len(after) != 0 {
		t.Fatal("Expected empty list after delete")
	}
}
