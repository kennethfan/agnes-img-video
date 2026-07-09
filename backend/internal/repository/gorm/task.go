package gorm

import (
	"github.com/agnes-image-tool/backend/internal/model"
	"gorm.io/gorm"
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Insert(task *model.TaskRecord) error {
	t := TaskRecord{
		Type:     task.Type,
		Status:   task.Status,
		Params:   task.Params,
		Progress: task.Progress,
	}
	return r.db.Create(&t).Error
}

func (r *TaskRepository) Get(id int64) (*model.TaskRecord, error) {
	var t TaskRecord
	if err := r.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return toTaskRecord(&t), nil
}

func (r *TaskRepository) List(page, pageSize int, status string) ([]model.TaskRecord, int, error) {
	var total int64
	query := r.db.Model(&TaskRecord{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var ts []TaskRecord
	if err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&ts).Error; err != nil {
		return nil, 0, err
	}
	records := make([]model.TaskRecord, len(ts))
	for i, t := range ts {
		records[i] = *toTaskRecord(&t)
	}
	return records, int(total), nil
}

func (r *TaskRepository) Update(task *model.TaskRecord) error {
	return r.db.Model(&TaskRecord{}).Where("id = ?", task.ID).Updates(map[string]any{
		"status":      task.Status,
		"result":      task.Result,
		"progress":    task.Progress,
		"error":       task.Error,
		"retry_count": task.RetryCount,
	}).Error
}

func (r *TaskRepository) Delete(id int64) error {
	return r.db.Delete(&TaskRecord{}, id).Error
}

func (r *TaskRepository) FindPending() ([]*model.TaskRecord, error) {
	var ts []TaskRecord
	if err := r.db.Where("status IN ?", []string{"pending", "processing"}).Order("created_at ASC").Find(&ts).Error; err != nil {
		return nil, err
	}
	result := make([]*model.TaskRecord, len(ts))
	for i, t := range ts {
		result[i] = toTaskRecord(&t)
	}
	return result, nil
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
