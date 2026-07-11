package gorm

import (
	"gorm.io/gorm"
)

type CollectionRepository struct {
	db *gorm.DB
}

func NewCollectionRepository(db *gorm.DB) *CollectionRepository {
	return &CollectionRepository{db: db}
}

func (r *CollectionRepository) List() ([]Collection, error) {
	var collections []Collection
	err := r.db.Preload("Assets").Find(&collections).Error
	return collections, err
}

func (r *CollectionRepository) Create(name string) (*Collection, error) {
	c := &Collection{Name: name}
	err := r.db.Create(c).Error
	return c, err
}

func (r *CollectionRepository) Update(id uint, name string) error {
	return r.db.Model(&Collection{}).Where("id = ?", id).Update("name", name).Error
}

func (r *CollectionRepository) Delete(id uint) error {
	r.db.Where("collection_id = ?", id).Delete(&AssetCollection{})
	return r.db.Delete(&Collection{}, id).Error
}

func (r *CollectionRepository) AddAssets(collectionID uint, assetIDs []uint) error {
	for _, aid := range assetIDs {
		r.db.FirstOrCreate(&AssetCollection{AssetID: aid, CollectionID: collectionID})
	}
	return nil
}

func (r *CollectionRepository) RemoveAssets(collectionID uint, assetIDs []uint) error {
	return r.db.Where("collection_id = ? AND asset_id IN ?", collectionID, assetIDs).
		Delete(&AssetCollection{}).Error
}
