package gorm

import (
	"testing"
)

func TestTaskCreateAndGet(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&TaskRecord{})
	repo := NewTaskRepository(db)

	id, err := repo.CreateTask("image_gen", `{"prompt":"test"}`)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	task, err := repo.GetTask(id)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task == nil {
		t.Fatal("expected non-nil task")
	}
	if task.Type != "image_gen" {
		t.Fatalf("expected type 'image_gen', got %q", task.Type)
	}
	if task.Status != "pending" {
		t.Fatalf("expected status 'pending', got %q", task.Status)
	}
}

func TestTaskGetNotFound(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&TaskRecord{})
	repo := NewTaskRepository(db)

	task, err := repo.GetTask(999)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task != nil {
		t.Fatal("expected nil for not found")
	}
}

func TestTaskUpdateStatus(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&TaskRecord{})
	repo := NewTaskRepository(db)

	id, _ := repo.CreateTask("video_gen", "{}")

	if err := repo.UpdateTaskStatus(id, "processing", 50, "", ""); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}
	task, _ := repo.GetTask(id)
	if task.Status != "processing" {
		t.Fatalf("expected 'processing', got %q", task.Status)
	}
	if task.Progress != 50 {
		t.Fatalf("expected progress 50, got %d", task.Progress)
	}

	// Complete with result
	if err := repo.UpdateTaskStatus(id, "completed", 100, "https://example.com/video.mp4", ""); err != nil {
		t.Fatalf("UpdateTaskStatus(completed) failed: %v", err)
	}
	task, _ = repo.GetTask(id)
	if task.Status != "completed" {
		t.Fatalf("expected 'completed', got %q", task.Status)
	}
	if task.Result != "https://example.com/video.mp4" {
		t.Fatalf("expected result URL, got %q", task.Result)
	}
}

func TestTaskUpdateProgress(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&TaskRecord{})
	repo := NewTaskRepository(db)

	id, _ := repo.CreateTask("test", "{}")
	for _, p := range []int{10, 30, 60, 100} {
		if err := repo.UpdateTaskProgress(id, p); err != nil {
			t.Fatalf("UpdateTaskProgress(%d) failed: %v", p, err)
		}
	}
	task, _ := repo.GetTask(id)
	if task.Progress != 100 {
		t.Fatalf("expected progress 100, got %d", task.Progress)
	}
}

func TestTaskUpdateRetryCount(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&TaskRecord{})
	repo := NewTaskRepository(db)

	id, _ := repo.CreateTask("test", "{}")
	if err := repo.UpdateRetryCount(id, 3); err != nil {
		t.Fatalf("UpdateRetryCount failed: %v", err)
	}
	task, _ := repo.GetTask(id)
	if task.RetryCount != 3 {
		t.Fatalf("expected retry_count 3, got %d", task.RetryCount)
	}
}

func TestTaskCancelAtomic(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&TaskRecord{})
	repo := NewTaskRepository(db)

	id, _ := repo.CreateTask("cancel_test", "{}")

	cancelled, err := repo.CancelTaskAtomic(id)
	if err != nil {
		t.Fatalf("CancelTaskAtomic failed: %v", err)
	}
	if !cancelled {
		t.Fatal("expected cancellation to succeed")
	}

	task, _ := repo.GetTask(id)
	if task.Status != "cancelled" {
		t.Fatalf("expected 'cancelled', got %q", task.Status)
	}

	// Cancelling again should return false
	cancelled, err = repo.CancelTaskAtomic(id)
	if err != nil {
		t.Fatalf("CancelTaskAtomic again failed: %v", err)
	}
	if cancelled {
		t.Fatal("expected second cancel to return false")
	}
}

func TestTaskCancelAtomicOnlyPending(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&TaskRecord{})
	repo := NewTaskRepository(db)

	id, _ := repo.CreateTask("test", "{}")
	repo.UpdateTaskStatus(id, "processing", 0, "", "")

	cancelled, err := repo.CancelTaskAtomic(id)
	if err != nil {
		t.Fatalf("CancelTaskAtomic failed: %v", err)
	}
	if cancelled {
		t.Fatal("expected cancel to fail for non-pending task")
	}
}

func TestTaskFindPending(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&TaskRecord{})
	repo := NewTaskRepository(db)

	repo.CreateTask("a", "{}")                          // pending
	id2, _ := repo.CreateTask("b", "{}")                // pending
	repo.UpdateTaskStatus(id2, "processing", 0, "", "") // processing
	id3, _ := repo.CreateTask("c", "{}")                // pending
	repo.UpdateTaskStatus(id3, "completed", 100, "", "") // completed

	pending, err := repo.FindPendingTasks()
	if err != nil {
		t.Fatalf("FindPendingTasks failed: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending tasks, got %d", len(pending))
	}
}

func TestTaskListTasks(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&TaskRecord{})
	repo := NewTaskRepository(db)

	repo.CreateTask("image", `{"p":"a"}`)
	repo.CreateTask("image", `{"p":"b"}`)
	repo.CreateTask("video", `{"p":"c"}`)

	// Filter by type
	tasks, err := repo.ListTasks("image", "", 10, 0)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 image tasks, got %d", len(tasks))
	}

	// All tasks
	tasks, _ = repo.ListTasks("", "", 10, 0)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 total tasks, got %d", len(tasks))
	}

	// Pagination
	tasks, _ = repo.ListTasks("", "", 1, 0)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task with limit 1, got %d", len(tasks))
	}

	// Negative limit should default to 50
	tasks, _ = repo.ListTasks("", "", 0, 0)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks with default limit, got %d", len(tasks))
	}
}

func TestTaskCleanupOlderThan(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&TaskRecord{})
	repo := NewTaskRepository(db)

	id, _ := repo.CreateTask("test", "{}")

	// SQLite datetime('now') in GORM uses UTC by default.
	// The CleanupOlderThan uses datetime('now', ?) which is SQLite-local.
	// Since these were just created, they shouldn't be cleaned up.
	count, err := repo.CleanupOlderThan(0)
	if err != nil {
		t.Fatalf("CleanupOlderThan failed: %v", err)
	}
	_ = id // task still exists (no completed_at set)
	_ = count

	// Actually the query requires completed_at IS NOT NULL,
	// and since we never called UpdateTaskStatus with completed,
	// the record doesn't have completed_at.
	task, _ := repo.GetTask(id)
	if task == nil {
		t.Fatal("expected task to still exist (no completed_at)")
	}
}

func TestTaskListTasksStatusFilter(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&TaskRecord{})
	repo := NewTaskRepository(db)

	id1, _ := repo.CreateTask("t1", "{}")
	repo.UpdateTaskStatus(id1, "completed", 100, "", "")
	repo.CreateTask("t2", "{}")

	tasks, _ := repo.ListTasks("", "pending", 10, 0)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 pending task, got %d", len(tasks))
	}

	tasks, _ = repo.ListTasks("", "completed", 10, 0)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 completed task, got %d", len(tasks))
	}
}
