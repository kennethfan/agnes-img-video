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

func (r *CollectionRepository) Update(id int64, name string) error {
	return r.db.Model(&Collection{}).Where("id = ?", id).Update("name", name).Error
}

func (r *CollectionRepository) Delete(id int64) error {
	if err := r.db.Where("collection_id = ?", id).Delete(&AssetCollection{}).Error; err != nil {
		return err
	}
	return r.db.Delete(&Collection{}, id).Error
}

func (r *CollectionRepository) AddAssets(collectionID int64, assetIDs []int64) error {
	for _, aid := range assetIDs {
		if err := r.db.FirstOrCreate(&AssetCollection{AssetID: aid, CollectionID: collectionID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *CollectionRepository) RemoveAssets(collectionID int64, assetIDs []int64) error {
	return r.db.Where("collection_id = ? AND asset_id IN ?", collectionID, assetIDs).
		Delete(&AssetCollection{}).Error
}
