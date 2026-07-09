package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/agnes-image-tool/backend/internal/model"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) InitTable() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id          TEXT PRIMARY KEY,
			type        TEXT NOT NULL,
			status      TEXT NOT NULL DEFAULT 'pending',
			params      TEXT NOT NULL,
			result      TEXT,
			progress    INTEGER NOT NULL DEFAULT 0,
			error       TEXT,
			retry_count INTEGER NOT NULL DEFAULT 0,
			created_at  TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			updated_at  TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			completed_at TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 tasks 表失败: %w", err)
	}

	// 创建索引
	_, err = r.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);
		CREATE INDEX IF NOT EXISTS idx_tasks_created ON tasks(created_at);
	`)
	if err != nil {
		return fmt.Errorf("创建 tasks 索引失败: %w", err)
	}

	log.Println("[TaskRepo] 任务表初始化完成")
	return nil
}

// generateTaskID 生成唯一任务 ID: task_{hex12}
func generateTaskID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return fmt.Sprintf("task_%s", hex.EncodeToString(b))
}

func (r *TaskRepository) CreateTask(taskType, params string) (string, error) {
	id := generateTaskID()
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := r.db.Exec(
		"INSERT INTO tasks (id, type, status, params, progress, created_at, updated_at) VALUES (?, ?, 'pending', ?, 0, ?, ?)",
		id, taskType, params, now, now,
	)
	if err != nil {
		return "", fmt.Errorf("创建任务失败: %w", err)
	}
	log.Printf("[TaskRepo] 任务已创建: id=%s type=%s", id, taskType)
	return id, nil
}

func (r *TaskRepository) GetTask(id string) (*model.TaskRecord, error) {
	var rec model.TaskRecord
	var result, errStr, completedAt sql.NullString
	err := r.db.QueryRow(
		"SELECT id, type, status, params, result, progress, error, retry_count, created_at, updated_at, completed_at FROM tasks WHERE id = ?",
		id,
	).Scan(&rec.ID, &rec.Type, &rec.Status, &rec.Params, &result, &rec.Progress, &errStr, &rec.RetryCount, &rec.CreatedAt, &rec.UpdatedAt, &completedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	if result.Valid {
		rec.Result = result.String
	}
	if errStr.Valid {
		rec.Error = errStr.String
	}
	if completedAt.Valid {
		rec.CompletedAt = completedAt.String
	}
	return &rec, nil
}

func (r *TaskRepository) UpdateTaskStatus(id, status string, progress int, result, errMsg string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	completedAt := sql.NullString{Valid: false}
	if status == string(model.TaskStatusCompleted) || status == string(model.TaskStatusFailed) {
		completedAt = sql.NullString{String: now, Valid: true}
	}

	_, err := r.db.Exec(
		"UPDATE tasks SET status = ?, progress = ?, result = ?, error = ?, updated_at = ?, completed_at = COALESCE(?, completed_at) WHERE id = ?",
		status, progress, result, errMsg, now, completedAt, id,
	)
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}
	return nil
}

func (r *TaskRepository) UpdateTaskProgress(id string, progress int) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := r.db.Exec(
		"UPDATE tasks SET progress = ?, updated_at = ? WHERE id = ?",
		progress, now, id,
	)
	return err
}

func (r *TaskRepository) UpdateRetryCount(id string, count int) error {
	_, err := r.db.Exec("UPDATE tasks SET retry_count = ? WHERE id = ?", count, id)
	return err
}

func (r *TaskRepository) CancelTaskAtomic(id string) (bool, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := r.db.Exec(
		"UPDATE tasks SET status = ?, updated_at = ?, completed_at = ? WHERE id = ? AND status = ?",
		string(model.TaskStatusCancelled), now, now, id, string(model.TaskStatusPending),
	)
	if err != nil {
		return false, fmt.Errorf("取消任务失败: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return false, nil
	}
	log.Printf("[TaskRepo] 任务已取消: id=%s", id)
	return true, nil
}

func (r *TaskRepository) FindPendingTasks() ([]*model.TaskRecord, error) {
	rows, err := r.db.Query(
		"SELECT id, type, status, params, result, progress, error, retry_count, created_at, updated_at, completed_at FROM tasks WHERE status IN ('pending', 'processing') ORDER BY created_at ASC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*model.TaskRecord
	for rows.Next() {
		var rec model.TaskRecord
		var result, errStr, completedAt sql.NullString
		if err := rows.Scan(&rec.ID, &rec.Type, &rec.Status, &rec.Params, &result, &rec.Progress, &errStr, &rec.RetryCount, &rec.CreatedAt, &rec.UpdatedAt, &completedAt); err != nil {
			continue
		}
		if result.Valid {
			rec.Result = result.String
		}
		if errStr.Valid {
			rec.Error = errStr.String
		}
		if completedAt.Valid {
			rec.CompletedAt = completedAt.String
		}
		results = append(results, &rec)
	}
	return results, rows.Err()
}

func (r *TaskRepository) ListTasks(taskType, status string, limit, offset int) ([]*model.TaskRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	where := "1=1"
	args := []any{}

	if taskType != "" {
		where += " AND type = ?"
		args = append(args, taskType)
	}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	query := fmt.Sprintf("SELECT id, type, status, params, result, progress, error, retry_count, created_at, updated_at, completed_at FROM tasks WHERE %s ORDER BY created_at DESC LIMIT ? OFFSET ?", where)
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*model.TaskRecord
	for rows.Next() {
		var rec model.TaskRecord
		var result, errStr, completedAt sql.NullString
		if err := rows.Scan(&rec.ID, &rec.Type, &rec.Status, &rec.Params, &result, &rec.Progress, &errStr, &rec.RetryCount, &rec.CreatedAt, &rec.UpdatedAt, &completedAt); err != nil {
			continue
		}
		if result.Valid {
			rec.Result = result.String
		}
		if errStr.Valid {
			rec.Error = errStr.String
		}
		if completedAt.Valid {
			rec.CompletedAt = completedAt.String
		}
		results = append(results, &rec)
	}
	return results, rows.Err()
}

func (r *TaskRepository) CleanupOlderThan(hours int) (int64, error) {
	res, err := r.db.Exec(
		"DELETE FROM tasks WHERE completed_at IS NOT NULL AND completed_at < datetime('now', ?)",
		fmt.Sprintf("-%d hours", hours),
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
