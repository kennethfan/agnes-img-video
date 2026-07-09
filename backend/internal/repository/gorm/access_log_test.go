package gorm

import (
	"testing"

	"github.com/agnes-image-tool/backend/internal/repository"
)

func TestAccessLogInsert(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&AccessLog{})
	repo := NewAccessLogRepository(db)

	err := repo.Insert(&repository.AccessLogRecord{
		Method:     "GET",
		Path:       "/api/v1/history",
		Status:     200,
		DurationMs: 42,
		ClientIP:   "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	result, err := repo.Query(repository.AccessLogQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if result.Items[0].Path != "/api/v1/history" {
		t.Fatalf("expected '/api/v1/history', got %q", result.Items[0].Path)
	}
}

func TestAccessLogQueryWithFilters(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&AccessLog{})
	repo := NewAccessLogRepository(db)

	records := []repository.AccessLogRecord{
		{Method: "GET", Path: "/api/v1/history", Status: 200},
		{Method: "POST", Path: "/api/v1/images/text-to-image", Status: 200},
		{Method: "GET", Path: "/api/v1/config", Status: 404},
		{Method: "DELETE", Path: "/api/v1/history", Status: 500},
	}
	for _, r := range records {
		repo.Insert(&r)
	}

	// Filter by method
	result, _ := repo.Query(repository.AccessLogQuery{
		Method:   "GET",
		Page:     1,
		PageSize: 10,
	})
	if result.Total != 2 {
		t.Fatalf("expected 2 GET logs, got %d", result.Total)
	}

	// Filter by path
	result, _ = repo.Query(repository.AccessLogQuery{
		Path:     "history",
		Page:     1,
		PageSize: 10,
	})
	if result.Total != 2 {
		t.Fatalf("expected 2 history logs, got %d", result.Total)
	}

	// Filter by status range
	result, _ = repo.Query(repository.AccessLogQuery{
		StatusMin: 400,
		StatusMax: 599,
		Page:      1,
		PageSize:  10,
	})
	if result.Total != 2 {
		t.Fatalf("expected 2 error logs, got %d", result.Total)
	}
}

func TestAccessLogPagination(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&AccessLog{})
	repo := NewAccessLogRepository(db)

	for range 25 {
		repo.Insert(&repository.AccessLogRecord{Method: "GET", Path: "/", Status: 200})
	}

	result, _ := repo.Query(repository.AccessLogQuery{Page: 1, PageSize: 10})
	if len(result.Items) != 10 {
		t.Fatalf("expected 10 items on page 1, got %d", len(result.Items))
	}
	if result.Total != 25 {
		t.Fatalf("expected total 25, got %d", result.Total)
	}
}

func TestAccessLogDelete(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&AccessLog{})
	repo := NewAccessLogRepository(db)

	repo.Insert(&repository.AccessLogRecord{Method: "GET", Path: "/delete-me", Status: 200})

	result, _ := repo.Query(repository.AccessLogQuery{Page: 1, PageSize: 10})
	if result.Total != 1 {
		t.Fatalf("expected 1 record before delete, got %d", result.Total)
	}

	if err := repo.Delete(result.Items[0].ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	result, _ = repo.Query(repository.AccessLogQuery{Page: 1, PageSize: 10})
	if result.Total != 0 {
		t.Fatalf("expected 0 records after delete, got %d", result.Total)
	}
}

func TestAccessLogClearAll(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&AccessLog{})
	repo := NewAccessLogRepository(db)

	repo.Insert(&repository.AccessLogRecord{Method: "GET", Path: "/a", Status: 200})
	repo.Insert(&repository.AccessLogRecord{Method: "POST", Path: "/b", Status: 200})

	if err := repo.ClearAll(); err != nil {
		t.Fatalf("ClearAll failed: %v", err)
	}

	result, _ := repo.Query(repository.AccessLogQuery{Page: 1, PageSize: 10})
	if result.Total != 0 {
		t.Fatalf("expected 0 records after clear, got %d", result.Total)
	}
}

func TestAccessLogSortAsc(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&AccessLog{})
	repo := NewAccessLogRepository(db)

	repo.Insert(&repository.AccessLogRecord{Method: "GET", Path: "/first", Status: 200})
	repo.Insert(&repository.AccessLogRecord{Method: "POST", Path: "/second", Status: 200})

	result, _ := repo.Query(repository.AccessLogQuery{
		Sort:     "asc",
		Page:     1,
		PageSize: 10,
	})
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	// First inserted should be first in ASC order
	if result.Items[0].Path != "/first" {
		t.Fatalf("expected '/first' first in ASC sort, got %q", result.Items[0].Path)
	}
}
