package gorm

import (
	"fmt"
	"time"

	"github.com/agnes-image-tool/backend/internal/model"
	"gorm.io/gorm"
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) InitTable() error {
	return r.db.AutoMigrate(&TaskRecord{})
}

func (r *TaskRepository) CreateTask(taskType, params string) (int64, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	t := TaskRecord{Type: taskType, Status: "pending", Params: params, CreatedAt: now, UpdatedAt: now}
	if err := r.db.Create(&t).Error; err != nil {
		return 0, err
	}
	return t.ID, nil
}

func (r *TaskRepository) GetTask(id int64) (*model.TaskRecord, error) {
	var t TaskRecord
	if err := r.db.First(&t, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return toTaskRecord(&t), nil
}

func (r *TaskRepository) UpdateTaskStatus(id int64, status string, progress int, result, errMsg string) error {
	return r.db.Model(&TaskRecord{}).Where("id = ?", id).Updates(map[string]any{
		"status":   status,
		"progress": progress,
		"result":   result,
		"error":    errMsg,
	}).Error
}

func (r *TaskRepository) UpdateTaskProgress(id int64, progress int) error {
	return r.db.Model(&TaskRecord{}).Where("id = ?", id).Update("progress", progress).Error
}

func (r *TaskRepository) UpdateRetryCount(id int64, count int) error {
	return r.db.Model(&TaskRecord{}).Where("id = ?", id).Update("retry_count", count).Error
}

func (r *TaskRepository) CancelTaskAtomic(id int64) (bool, error) {
	res := r.db.Model(&TaskRecord{}).Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]any{"status": "cancelled"})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		return false, nil
	}
	return true, nil
}

func (r *TaskRepository) FindPendingTasks() ([]*model.TaskRecord, error) {
	var ts []TaskRecord
	if err := r.db.Where("status IN ?", []string{"pending", "processing"}).
		Order("created_at ASC").Find(&ts).Error; err != nil {
		return nil, err
	}
	result := make([]*model.TaskRecord, len(ts))
	for i, t := range ts {
		result[i] = toTaskRecord(&t)
	}
	return result, nil
}

func (r *TaskRepository) ListTasks(taskType, status string, limit, offset int) ([]*model.TaskRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := r.db.Model(&TaskRecord{})
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var ts []TaskRecord
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&ts).Error; err != nil {
		return nil, err
	}
	result := make([]*model.TaskRecord, len(ts))
	for i, t := range ts {
		result[i] = toTaskRecord(&t)
	}
	return result, nil
}

func (r *TaskRepository) ListByProjectID(projectID int64) ([]*model.TaskRecord, error) {
	var ts []TaskRecord
	if err := r.db.Where("project_id = ?", projectID).Order("created_at DESC").Find(&ts).Error; err != nil {
		return nil, err
	}
	result := make([]*model.TaskRecord, len(ts))
	for i, t := range ts {
		result[i] = toTaskRecord(&t)
	}
	return result, nil
}

func (r *TaskRepository) CleanupOlderThan(hours int) (int64, error) {
	res := r.db.Where("completed_at IS NOT NULL AND completed_at < datetime('now', ?)",
		fmt.Sprintf("-%d hours", hours)).Delete(&TaskRecord{})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func toTaskRecord(t *TaskRecord) *model.TaskRecord {
	rec := &model.TaskRecord{
		ID:         t.ID,
		Type:       t.Type,
		Status:     t.Status,
		Params:     t.Params,
		Progress:   t.Progress,
		RetryCount: t.RetryCount,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
		ProjectID:  t.ProjectID,
	}
	if t.Result != nil {
		rec.Result = *t.Result
	}
	if t.Error != nil {
		rec.Error = *t.Error
	}
	if t.CompletedAt != nil {
		rec.CompletedAt = *t.CompletedAt
	}
	return rec
}
