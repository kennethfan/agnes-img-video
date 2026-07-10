package gorm

import (
	"fmt"

	"github.com/agnes-image-tool/backend/internal/model"
	"gorm.io/gorm"
)

type AssetRepository struct {
	db *gorm.DB
}

func NewAssetRepository(db *gorm.DB) *AssetRepository {
	return &AssetRepository{db: db}
}

func (r *AssetRepository) Insert(asset *model.Asset) (int64, error) {
	if err := r.db.Create(asset).Error; err != nil {
		return 0, fmt.Errorf("插入资产失败: %w", err)
	}
	return asset.ID, nil
}

func (r *AssetRepository) List(page, perPage int, assetType, search string, favoriteFilter bool) ([]model.Asset, int, error) {
	query := r.db.Model(&model.Asset{})
	if assetType != "" && assetType != "all" {
		query = query.Where("type = ?", assetType)
	}
	if search != "" {
		query = query.Where("prompt LIKE ?", "%"+search+"%")
	}
	if favoriteFilter {
		query = query.Where("favorite = ?", true)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计资产总数失败: %w", err)
	}
	var assets []model.Asset
	if err := query.Order("id DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&assets).Error; err != nil {
		return nil, 0, fmt.Errorf("查询资产列表失败: %w", err)
	}
	return assets, int(total), nil
}

func (r *AssetRepository) GetByIDs(ids []int64) ([]model.Asset, error) {
	var assets []model.Asset
	if err := r.db.Where("id IN ?", ids).Find(&assets).Error; err != nil {
		return nil, fmt.Errorf("查询资产失败: %w", err)
	}
	return assets, nil
}

func (r *AssetRepository) ToggleFavorite(id int64, favorite bool) error {
	if err := r.db.Model(&model.Asset{}).Where("id = ?", id).Update("favorite", favorite).Error; err != nil {
		return fmt.Errorf("更新收藏状态失败: %w", err)
	}
	return nil
}

func (r *AssetRepository) UpdateGithubURL(id int64, githubURL string) error {
	if err := r.db.Model(&model.Asset{}).Where("id = ?", id).Update("github_url", githubURL).Error; err != nil {
		return fmt.Errorf("更新 GitHub URL 失败: %w", err)
	}
	return nil
}

func (r *AssetRepository) Delete(ids []int64) error {
	if err := r.db.Where("id IN ?", ids).Delete(&model.Asset{}).Error; err != nil {
		return fmt.Errorf("删除资产失败: %w", err)
	}
	return nil
}
