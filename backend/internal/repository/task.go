package repository

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/agnes-image-tool/backend/internal/model"
)

type TaskSQLiteRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskSQLiteRepository {
	return &TaskSQLiteRepository{db: db}
}

func (r *TaskSQLiteRepository) InitTable() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
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

func (r *TaskSQLiteRepository) CreateTask(taskType, params string) (int64, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := r.db.Exec(
		"INSERT INTO tasks (type, status, params, progress, created_at, updated_at) VALUES (?, 'pending', ?, 0, ?, ?)",
		taskType, params, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("创建任务失败: %w", err)
	}
	id, _ := res.LastInsertId()
	log.Printf("[TaskRepo] 任务已创建: id=%d type=%s", id, taskType)
	return id, nil
}

func (r *TaskSQLiteRepository) GetTask(id int64) (*model.TaskRecord, error) {
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

func (r *TaskSQLiteRepository) UpdateTaskStatus(id int64, status string, progress int, result, errMsg string) error {
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

func (r *TaskSQLiteRepository) UpdateTaskProgress(id int64, progress int) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := r.db.Exec(
		"UPDATE tasks SET progress = ?, updated_at = ? WHERE id = ?",
		progress, now, id,
	)
	return err
}

func (r *TaskSQLiteRepository) UpdateRetryCount(id int64, count int) error {
	_, err := r.db.Exec("UPDATE tasks SET retry_count = ? WHERE id = ?", count, id)
	return err
}

func (r *TaskSQLiteRepository) CancelTaskAtomic(id int64) (bool, error) {
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
	log.Printf("[TaskRepo] 任务已取消: id=%d", id)
	return true, nil
}

func (r *TaskSQLiteRepository) FindPendingTasks() ([]*model.TaskRecord, error) {
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

func (r *TaskSQLiteRepository) ListTasks(taskType, status string, limit, offset int) ([]*model.TaskRecord, error) {
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

func (r *TaskSQLiteRepository) CleanupOlderThan(hours int) (int64, error) {
	res, err := r.db.Exec(
		"DELETE FROM tasks WHERE completed_at IS NOT NULL AND completed_at < datetime('now', ?)",
		fmt.Sprintf("-%d hours", hours),
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
