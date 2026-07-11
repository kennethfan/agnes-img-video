package gorm

import (
	"gorm.io/gorm"
)

type TemplateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) List(category string) ([]PromptTemplate, error) {
	query := r.db.Order("updated_at desc")
	if category != "" {
		query = query.Where("category = ?", category)
	}
	var templates []PromptTemplate
	err := query.Find(&templates).Error
	return templates, err
}

func (r *TemplateRepository) GetByID(id int64) (*PromptTemplate, error) {
	var t PromptTemplate
	err := r.db.First(&t, id).Error
	return &t, err
}

func (r *TemplateRepository) Create(t *PromptTemplate) error {
	return r.db.Create(t).Error
}

func (r *TemplateRepository) Update(t *PromptTemplate) error {
	return r.db.Model(&PromptTemplate{}).Where("id = ?", t.ID).Updates(map[string]interface{}{
		"name": t.Name, "type": t.Type, "category": t.Category,
		"prompt": t.Prompt, "negative_prompt": t.NegativePrompt,
		"size": t.Size, "strength": t.Strength, "model": t.Model,
	}).Error
}

func (r *TemplateRepository) Delete(id int64) error {
	return r.db.Delete(&PromptTemplate{}, id).Error
}

func (r *TemplateRepository) Export() ([]PromptTemplate, error) {
	var templates []PromptTemplate
	err := r.db.Find(&templates).Error
	return templates, err
}

func (r *TemplateRepository) Import(templates []PromptTemplate) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i := range templates {
			templates[i].ID = 0
			if err := tx.Create(&templates[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
