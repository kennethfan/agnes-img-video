package gorm

import (
	"github.com/agnes-image-tool/backend/internal/model"
	"gorm.io/gorm"
)

type StoryboardRepository struct {
	db *gorm.DB
}

func NewStoryboardRepository(db *gorm.DB) *StoryboardRepository {
	return &StoryboardRepository{db: db}
}

func (r *StoryboardRepository) ListProjects() ([]model.StoryboardProject, error) {
	var projects []StoryboardProject
	if err := r.db.Order("updated_at DESC").Find(&projects).Error; err != nil {
		return nil, err
	}
	result := make([]model.StoryboardProject, len(projects))
	for i, p := range projects {
		var shotCount int64
		r.db.Model(&StoryboardShot{}).Where("project_id = ?", p.ID).Count(&shotCount)
		result[i] = model.StoryboardProject{
			ID:        p.ID,
			Title:     p.Title,
			Script:    p.Script,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
			ShotCount: int(shotCount),
		}
	}
	return result, nil
}

func (r *StoryboardRepository) CreateProject(title, script string) (int64, error) {
	p := StoryboardProject{Title: title, Script: script}
	if err := r.db.Create(&p).Error; err != nil {
		return 0, err
	}
	return p.ID, nil
}

func (r *StoryboardRepository) GetProject(id int64) (*model.StoryboardProject, error) {
	var p StoryboardProject
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &model.StoryboardProject{
		ID:        p.ID,
		Title:     p.Title,
		Script:    p.Script,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}, nil
}

func (r *StoryboardRepository) UpdateProject(id int64, title, script string) error {
	return r.db.Model(&StoryboardProject{}).Where("id = ?", id).Updates(map[string]any{
		"title": title, "script": script,
	}).Error
}

func (r *StoryboardRepository) DeleteProject(id int64) error {
	return r.db.Delete(&StoryboardProject{}, id).Error
}

func (r *StoryboardRepository) DuplicateProject(id int64) (int64, error) {
	tx := r.db.Begin()
	var orig StoryboardProject
	if err := tx.First(&orig, id).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	dup := StoryboardProject{Title: orig.Title + " (副本)", Script: orig.Script}
	if err := tx.Create(&dup).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	var shots []StoryboardShot
	if err := tx.Where("project_id = ?", id).Find(&shots).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	for _, s := range shots {
		s.ID = 0
		s.ProjectID = dup.ID
		if err := tx.Create(&s).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}
	tx.Commit()
	return dup.ID, nil
}

func (r *StoryboardRepository) ListShots(projectID int64) ([]model.StoryboardShot, error) {
	var shots []StoryboardShot
	if err := r.db.Where("project_id = ?", projectID).Order("sequence ASC").Find(&shots).Error; err != nil {
		return nil, err
	}
	result := make([]model.StoryboardShot, len(shots))
	for i, s := range shots {
		result[i] = *r.toShotModel(&s)
	}
	return result, nil
}

// toShotModel 将 GORM 模型转换为 API 模型
func (r *StoryboardRepository) toShotModel(s *StoryboardShot) *model.StoryboardShot {
	return &model.StoryboardShot{
		ID:             s.ID,
		ProjectID:      s.ProjectID,
		Sequence:       s.Sequence,
		Prompt:         s.Prompt,
		Type:           s.Type,
		ReferenceImage: s.ReferenceImage,
		Status:         s.Status,
		ResultVideo:    s.ResultVideo,
		TaskID:         s.TaskID,
		TaskRecordID:   s.TaskRecordID,
		CreatedAt:      s.CreatedAt,
	}
}

func (r *StoryboardRepository) CreateShot(projectID int64, seq int, prompt, shotType, refImage string) (int64, error) {
	s := StoryboardShot{
		ProjectID:      projectID,
		Sequence:       seq,
		Prompt:         prompt,
		Type:           shotType,
		ReferenceImage: refImage,
		Status:         "pending",
	}
	if err := r.db.Create(&s).Error; err != nil {
		return 0, err
	}
	return s.ID, nil
}

func (r *StoryboardRepository) UpdateShot(id int64, prompt, shotType, refImage string) error {
	return r.db.Model(&StoryboardShot{}).Where("id = ?", id).Updates(map[string]any{
		"prompt":          prompt,
		"type":            shotType,
		"reference_image": refImage,
	}).Error
}

func (r *StoryboardRepository) DeleteShot(id int64) error {
	return r.db.Delete(&StoryboardShot{}, id).Error
}

func (r *StoryboardRepository) ReorderShots(ids []int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, sid := range ids {
			if err := tx.Model(&StoryboardShot{}).Where("id = ?", sid).Update("sequence", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *StoryboardRepository) GetShot(id int64) (*model.StoryboardShot, error) {
	var s StoryboardShot
	if err := r.db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return r.toShotModel(&s), nil
}

func (r *StoryboardRepository) UpdateShotStatus(id int64, status, taskID string, taskRecordID int64) error {
	return r.db.Model(&StoryboardShot{}).Where("id = ?", id).Updates(map[string]any{
		"status":          status,
		"task_id":         taskID,
		"task_record_id":  taskRecordID,
	}).Error
}

func (r *StoryboardRepository) UpdateShotResult(id int64, resultVideo string) error {
	return r.db.Model(&StoryboardShot{}).Where("id = ?", id).Updates(map[string]any{
		"status":       "completed",
		"result_video": resultVideo,
	}).Error
}

func (r *StoryboardRepository) BatchCreateShots(projectID int64, prompts []string, shotType string) ([]model.StoryboardShot, error) {
	var maxSeq int64
	r.db.Model(&StoryboardShot{}).Where("project_id = ?", projectID).Select("COALESCE(MAX(sequence), 0)").Scan(&maxSeq)

	var gormShots []StoryboardShot
	for i, prompt := range prompts {
		gormShots = append(gormShots, StoryboardShot{
			ProjectID: projectID,
			Sequence:  int(maxSeq) + i + 1,
			Prompt:    prompt,
			Type:      shotType,
			Status:    "pending",
		})
	}
	if err := r.db.Create(&gormShots).Error; err != nil {
		return nil, err
	}
	result := make([]model.StoryboardShot, len(gormShots))
	for i, s := range gormShots {
		result[i] = *r.toShotModel(&s)
	}
	return result, nil
}

func (r *StoryboardRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
