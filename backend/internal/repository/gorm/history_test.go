package gorm

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openDBMemory(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open :memory: failed: %v", err)
	}
	if err := db.AutoMigrate(&History{}, &Favorite{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
	return db
}

func TestHistoryInsertAndGet(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	id, err := repo.InsertRecord("test prompt", []string{"img1.png"}, "text2image", nil)
	if err != nil {
		t.Fatalf("InsertRecord failed: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	records, err := repo.GetRecords(10)
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Prompt != "test prompt" {
		t.Fatalf("expected prompt 'test prompt', got %q", records[0].Prompt)
	}
	if records[0].Mode != "text2image" {
		t.Fatalf("expected mode 'text2image', got %q", records[0].Mode)
	}
}

func TestHistoryInsertWithExtra(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	extra := map[string]any{"taskId": 12345, "source": "batch"}
	_, err := repo.InsertRecord("extra test", []string{"img.png"}, "text2video", extra)
	if err != nil {
		t.Fatalf("InsertRecord failed: %v", err)
	}

	records, err := repo.GetRecords(10)
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Extra == nil {
		t.Fatal("expected extra to be non-nil")
	}
}

func TestHistoryGetRecordsPaginated(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	for i := range 25 {
		_, err := repo.InsertRecord("prompt "+string(rune('A'+i)), []string{"img.png"}, "text2image", nil)
		if err != nil {
			t.Fatalf("InsertRecord %d failed: %v", i, err)
		}
	}

	records, total, err := repo.GetRecordsPaginated(1, 10, "", "", nil)
	if err != nil {
		t.Fatalf("GetRecordsPaginated failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("expected total 25, got %d", total)
	}
	if len(records) != 10 {
		t.Fatalf("expected 10 records on page 1, got %d", len(records))
	}
}

func TestHistoryGetRecordsPaginatedWithFilter(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	repo.InsertRecord("cat image", []string{"c.png"}, "text2image", nil)
	repo.InsertRecord("dog video", []string{"d.png"}, "text2video", nil)
	repo.InsertRecord("car image", []string{"car.png"}, "text2image", nil)

	records, total, err := repo.GetRecordsPaginated(1, 20, "image", "", nil)
	if err != nil {
		t.Fatalf("GetRecordsPaginated failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 for image filter, got %d", total)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records for image filter, got %d", len(records))
	}

	records, total, err = repo.GetRecordsPaginated(1, 20, "video", "", nil)
	if err != nil {
		t.Fatalf("GetRecordsPaginated failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1 for video filter, got %d", total)
	}
}

func TestHistoryGetRecordsByIDs(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	id1, _ := repo.InsertRecord("a", []string{"a.png"}, "text2image", nil)
	id2, _ := repo.InsertRecord("b", []string{"b.png"}, "text2image", nil)
	repo.InsertRecord("c", []string{"c.png"}, "text2image", nil)

	records, err := repo.GetRecordsByIDs([]int64{id1, id2})
	if err != nil {
		t.Fatalf("GetRecordsByIDs failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
}

func TestHistoryDeleteRecord(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	id, _ := repo.InsertRecord("delete me", []string{"d.png"}, "text2image", nil)
	if err := repo.DeleteRecord(id); err != nil {
		t.Fatalf("DeleteRecord failed: %v", err)
	}
	records, _ := repo.GetRecords(10)
	if len(records) != 0 {
		t.Fatal("expected 0 records after delete")
	}
}

func TestHistoryDeleteRecords(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	id1, _ := repo.InsertRecord("a", []string{"a.png"}, "text2image", nil)
	id2, _ := repo.InsertRecord("b", []string{"b.png"}, "text2image", nil)
	repo.InsertRecord("c", []string{"c.png"}, "text2image", nil)

	if err := repo.DeleteRecords([]int64{id1, id2}); err != nil {
		t.Fatalf("DeleteRecords failed: %v", err)
	}
	records, _ := repo.GetRecords(10)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
}

func TestHistoryClearRecords(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	repo.InsertRecord("a", []string{"a.png"}, "text2image", nil)
	repo.InsertRecord("b", []string{"b.png"}, "text2image", nil)

	if err := repo.ClearRecords(); err != nil {
		t.Fatalf("ClearRecords failed: %v", err)
	}
	records, _ := repo.GetRecords(10)
	if len(records) != 0 {
		t.Fatal("expected 0 records after clear")
	}
}

func TestHistoryUpdateRecordImages(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	id, _ := repo.InsertRecord("update images", []string{"old.png"}, "text2image", nil)
	if err := repo.UpdateRecordImages(id, []string{"new.png", "new2.png"}); err != nil {
		t.Fatalf("UpdateRecordImages failed: %v", err)
	}
	records, _ := repo.GetRecords(10)
	if len(records[0].Images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(records[0].Images))
	}
	if records[0].Images[0] != "new.png" {
		t.Fatalf("expected 'new.png', got %q", records[0].Images[0])
	}
}

func TestHistoryFindByTaskId(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	extra := map[string]any{"taskId": 999}
	id, _ := repo.InsertRecord("task test", []string{}, "text2video", extra)

	foundID, err := repo.FindByTaskId(999)
	if err != nil {
		t.Fatalf("FindByTaskId failed: %v", err)
	}
	if foundID != id {
		t.Fatalf("expected id %d, got %d", id, foundID)
	}

	_, err = repo.FindByTaskId(888)
	if err == nil {
		t.Fatal("expected error for nonexistent taskId")
	}
}

func TestHistoryFindPendingVideos(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	// Should match: empty images + video mode + extra with taskId
	extra := map[string]any{"taskId": "task_001"}
	repo.InsertRecord("pending video", []string{}, "text2video", extra)

	// Should NOT match: has images
	repo.InsertRecord("has images", []string{"img.png"}, "text2video", extra)

	// Should NOT match: non-video mode
	repo.InsertRecord("image", []string{}, "text2image", extra)

	pending, err := repo.FindPendingVideos()
	if err != nil {
		t.Fatalf("FindPendingVideos failed: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending video, got %d", len(pending))
	}
	if pending[0].TaskID != "task_001" {
		t.Fatalf("expected task_001, got %q", pending[0].TaskID)
	}
}

func TestHistoryTrimRecords(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	for range 10 {
		repo.InsertRecord("x", []string{"x.png"}, "text2image", nil)
	}

	if err := repo.TrimRecords(3); err != nil {
		t.Fatalf("TrimRecords failed: %v", err)
	}
	records, _ := repo.GetRecords(100)
	if len(records) != 3 {
		t.Fatalf("expected 3 records after trim, got %d", len(records))
	}
}

func TestHistoryToggleFavorite(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	id, _ := repo.InsertRecord("fav test", []string{"f.png"}, "text2image", nil)

	// Favorite
	if err := repo.ToggleFavorite(id, true); err != nil {
		t.Fatalf("ToggleFavorite(true) failed: %v", err)
	}
	favs, _ := repo.GetFavoriteIDs()
	if !favs[id] {
		t.Fatal("expected id to be in favorites")
	}

	// Unfavorite
	if err := repo.ToggleFavorite(id, false); err != nil {
		t.Fatalf("ToggleFavorite(false) failed: %v", err)
	}
	favs, _ = repo.GetFavoriteIDs()
	if favs[id] {
		t.Fatal("expected id to NOT be in favorites")
	}
}

func TestHistoryGetFavoriteIDs(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	id1, _ := repo.InsertRecord("a", []string{"a.png"}, "text2image", nil)
	id2, _ := repo.InsertRecord("b", []string{"b.png"}, "text2image", nil)

	repo.ToggleFavorite(id1, true)
	repo.ToggleFavorite(id2, true)

	favs, err := repo.GetFavoriteIDs()
	if err != nil {
		t.Fatalf("GetFavoriteIDs failed: %v", err)
	}
	if len(favs) != 2 {
		t.Fatalf("expected 2 favorites, got %d", len(favs))
	}
}

func TestHistoryFindByTaskIdWithJSONExtract(t *testing.T) {
	db := openDBMemory(t)
	repo := NewHistoryRepository(db)

	// SQLite json_extract requires the JSON to be valid
	extra := map[string]any{"taskId": 42}
	id, _ := repo.InsertRecord("json extract test", []string{}, "text2video", extra)

	foundID, err := repo.FindByTaskId(42)
	if err != nil {
		t.Fatalf("FindByTaskId failed: %v", err)
	}
	if foundID != id {
		t.Fatalf("expected id %d, got %d", id, foundID)
	}
}

// Ensure HistoryRepository implements repository.HistoryRepository
var _ = (interface {
	InsertRecord(string, []string, string, any) (int64, error)
})((*HistoryRepository)(nil))
