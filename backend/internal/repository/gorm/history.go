package gorm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
	"gorm.io/gorm"
)

type HistoryRepository struct {
	db *gorm.DB
}

func NewHistoryRepository(db *gorm.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) InsertRecord(prompt string, images []string, mode string, extra any) (int64, error) {
	imagesJSON, _ := json.Marshal(images)
	var extraStr *string
	if extra != nil {
		b, _ := json.Marshal(extra)
		s := string(b)
		extraStr = &s
	}
	h := History{Prompt: prompt, Mode: mode, Images: string(imagesJSON), Extra: extraStr}
	if err := r.db.Create(&h).Error; err != nil {
		return 0, err
	}
	return h.ID, nil
}

func (r *HistoryRepository) GetRecords(limit int) ([]model.HistoryRecord, error) {
	var hs []History
	if err := r.db.Order("id DESC").Limit(limit).Find(&hs).Error; err != nil {
		return nil, err
	}
	return toHistoryRecords(hs), nil
}

func (r *HistoryRepository) GetRecordsPaginated(page, perPage int, assetType, search string, favIDs map[int64]bool) ([]model.HistoryRecord, int, error) {
	var total int64
	query := r.db.Model(&History{})
	if assetType != "" {
		switch assetType {
		case "image":
			query = query.Where("mode IN (?)", []string{"text2image", "image2image", "batch"})
		case "video":
			query = query.Where("mode IN (?)", []string{"text2video", "image2video", "multi_image_video"})
		}
	}
	if search != "" {
		query = query.Where("prompt LIKE ?", "%"+search+"%")
	}
	query.Count(&total)

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var hs []History
	if err := query.Order("id DESC").Limit(perPage).Offset(offset).Find(&hs).Error; err != nil {
		return nil, 0, err
	}
	return toHistoryRecords(hs), int(total), nil
}

func (r *HistoryRepository) GetRecordsByIDs(ids []int64) ([]model.HistoryRecord, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var hs []History
	if err := r.db.Where("id IN ?", ids).Order("id DESC").Find(&hs).Error; err != nil {
		return nil, err
	}
	return toHistoryRecords(hs), nil
}

func (r *HistoryRepository) DeleteRecord(id int64) error {
	return r.db.Delete(&History{}, id).Error
}

func (r *HistoryRepository) DeleteRecords(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("id IN ?", ids).Delete(&History{}).Error
}

func (r *HistoryRepository) ClearRecords() error {
	return r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&History{}).Error
}

func (r *HistoryRepository) UpdateRecordImages(id int64, images []string) error {
	imagesJSON, _ := json.Marshal(images)
	return r.db.Model(&History{}).Where("id = ?", id).Update("images", string(imagesJSON)).Error
}

func (r *HistoryRepository) FindByTaskId(taskId int64) (int64, error) {
	var h History
	err := r.db.Where("json_extract(extra, '$.taskId') = ?", taskId).Or("json_extract(extra, '$.taskId') = ?", fmt.Sprintf("%d", taskId)).Order("id DESC").First(&h).Error
	if err != nil {
		return 0, err
	}
	return h.ID, nil
}

func (r *HistoryRepository) FindPendingVideos() ([]repository.PendingVideoInfo, error) {
	var hs []History
	if err := r.db.Where("images = '[]' AND extra IS NOT NULL AND extra != ''").Order("id DESC").Find(&hs).Error; err != nil {
		return nil, err
	}
	var results []repository.PendingVideoInfo
	for _, h := range hs {
		if !strings.HasSuffix(h.Mode, "video") && h.Mode != "video" {
			continue
		}
		if h.Extra == nil {
			continue
		}
		var extra map[string]any
		if err := json.Unmarshal([]byte(*h.Extra), &extra); err != nil {
			continue
		}
		taskID := ""
		switch v := extra["taskId"].(type) {
		case string:
			taskID = v
		case float64:
			taskID = fmt.Sprintf("%.0f", v)
		}
		if taskID == "" {
			continue
		}
		results = append(results, repository.PendingVideoInfo{
			ID:     h.ID,
			TaskID: taskID,
			Prompt: h.Prompt,
			Mode:   h.Mode,
		})
	}
	return results, nil
}

func (r *HistoryRepository) TrimRecords(max int) error {
	sub := r.db.Select("id").Order("id DESC").Limit(max)
	return r.db.Where("id NOT IN (?)", sub).Delete(&History{}).Error
}

func (r *HistoryRepository) ToggleFavorite(historyID int64, favorite bool) error {
	if favorite {
		return r.db.Create(&Favorite{HistoryID: historyID}).Error
	}
	return r.db.Where("history_id = ?", historyID).Delete(&Favorite{}).Error
}

func (r *HistoryRepository) GetFavoriteIDs() (map[int64]bool, error) {
	var favs []Favorite
	if err := r.db.Find(&favs).Error; err != nil {
		return nil, err
	}
	result := make(map[int64]bool, len(favs))
	for _, f := range favs {
		result[f.HistoryID] = true
	}
	return result, nil
}

// toHistoryRecords 转换 GORM History → model.HistoryRecord
func toHistoryRecords(hs []History) []model.HistoryRecord {
	records := make([]model.HistoryRecord, 0, len(hs))
	for _, h := range hs {
		var images []string
		json.Unmarshal([]byte(h.Images), &images)
		if images == nil {
			images = []string{}
		}
		rec := model.HistoryRecord{
			ID:     h.ID,
			Time:   h.Time,
			Mode:   h.Mode,
			Prompt: h.Prompt,
			Images: images,
		}
		if h.Extra != nil {
			var extra any
			if err := json.Unmarshal([]byte(*h.Extra), &extra); err == nil {
				rec.Extra = extra
			}
		}
		records = append(records, rec)
	}
	return records
}
