package gorm

import (
	"gorm.io/gorm"
)

type ProjectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// List 获取项目列表（按更新时间倒序）
func (r *ProjectRepository) List() ([]Project, error) {
	var projects []Project
	err := r.db.Order("updated_at desc").Find(&projects).Error
	return projects, err
}

// GetByID 获取项目详情（含步骤，按 position 排序）
func (r *ProjectRepository) GetByID(id int64) (*Project, error) {
	var project Project
	err := r.db.Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("position asc")
	}).First(&project, id).Error
	return &project, err
}

// Create 创建项目
func (r *ProjectRepository) Create(project *Project) error {
	return r.db.Create(project).Error
}

// Update 更新项目字段
func (r *ProjectRepository) Update(project *Project) error {
	return r.db.Model(&Project{}).Where("id = ?", project.ID).Updates(map[string]interface{}{
		"title":     project.Title,
		"brief":     project.Brief,
		"ai_result": project.AIResult,
		"status":    project.Status,
		"cover_url": project.CoverURL,
		"final_url": project.FinalURL,
		"asset_ids": project.AssetIDs,
		"notes":     project.Notes,
	}).Error
}

// UpdateField 更新项目单个字段（用于 step_progress 等）
func (r *ProjectRepository) UpdateField(id int64, field, value string) error {
	return r.db.Model(&Project{}).Where("id = ?", id).Update(field, value).Error
}

// Delete 删除项目（级联删步骤）
func (r *ProjectRepository) Delete(id int64) error {
	r.db.Where("project_id = ?", id).Delete(&ProjectStep{})
	return r.db.Delete(&Project{}, id).Error
}

// AddStep 添加步骤
func (r *ProjectRepository) AddStep(step *ProjectStep) error {
	return r.db.Create(step).Error
}

// UpdateStep 更新步骤
func (r *ProjectRepository) UpdateStep(step *ProjectStep) error {
	return r.db.Model(&ProjectStep{}).Where("id = ?", step.ID).Updates(map[string]interface{}{
		"input":  step.Input,
		"output": step.Output,
	}).Error
}

// DeleteStep 删除步骤
func (r *ProjectRepository) DeleteStep(stepID int64) error {
	return r.db.Delete(&ProjectStep{}, stepID).Error
}

// GetStepByID 获取单个步骤
func (r *ProjectRepository) GetStepByID(stepID int64) (*ProjectStep, error) {
	var step ProjectStep
	err := r.db.First(&step, stepID).Error
	return &step, err
}

// Duplicate 复制项目（含步骤，新项目状态为 draft）
func (r *ProjectRepository) Duplicate(id int64) (*Project, error) {
	orig, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	newProject := &Project{
		Title: orig.Title + " (副本)",
		Brief: orig.Brief,
		Notes: orig.Notes,
		Status: "draft",
	}
	if err := r.db.Create(newProject).Error; err != nil {
		return nil, err
	}
	// 复制步骤（重置 ID）
	for _, step := range orig.Steps {
		newStep := &ProjectStep{
			ProjectID: newProject.ID,
			StepType:  step.StepType,
			Position:  step.Position,
			Input:     step.Input,
			Output:    step.Output,
		}
		if err := r.db.Create(newStep).Error; err != nil {
			return nil, err
		}
	}
	return newProject, nil
}
