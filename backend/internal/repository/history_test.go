package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryRepo(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "history_test")
	defer os.RemoveAll(tmpDir)
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewHistoryRepo(dbPath)
	if err != nil {
		t.Fatalf("NewHistoryRepo failed: %v", err)
	}
	defer repo.Close()

	// Test Insert
	id, err := repo.InsertRecord("test prompt", []string{"outputs/img.png"}, "test_mode", nil)
	if err != nil {
		t.Fatalf("InsertRecord failed: %v", err)
	}
	t.Logf("Insert OK, id=%d", id)

	// Test FindByTaskId (negative)
	_, err = repo.FindByTaskId(99999)
	if err == nil {
		t.Log("FindByTaskId: nonexistent returns no error (ok)")
	}

	// Test UpdateRecordImages
	err = repo.UpdateRecordImages(id, []string{"outputs/updated.png"})
	if err != nil {
		t.Fatalf("UpdateRecordImages failed: %v", err)
	}
	t.Log("UpdateRecordImages OK")

	// Test Query
	records, err := repo.GetRecords(10)
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	t.Logf("Record: time=%q mode=%q prompt=%q images=%v", records[0].Time, records[0].Mode, records[0].Prompt, records[0].Images)
}
