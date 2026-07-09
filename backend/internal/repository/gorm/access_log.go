package gorm

import (
	"log"
	"strings"
	"time"

	"github.com/agnes-image-tool/backend/internal/repository"
	"gorm.io/gorm"
)

type AccessLogRepository struct {
	db *gorm.DB
}

func NewAccessLogRepository(db *gorm.DB) *AccessLogRepository {
	return &AccessLogRepository{db: db}
}

func (r *AccessLogRepository) InsertRecord(method, path string, status int, durationMs int, clientIP, userAgent, requestBody, responseBody, errMsg string) (int64, error) {
	al := AccessLog{
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		Method:       method,
		Path:         path,
		Status:       status,
		DurationMs:   durationMs,
		ClientIP:     clientIP,
		UserAgent:    userAgent,
		RequestBody:  requestBody,
		ResponseBody: responseBody,
		Error:        errMsg,
	}
	if err := r.db.Create(&al).Error; err != nil {
		return 0, err
	}
	return al.ID, nil
}

func (r *AccessLogRepository) Query(q repository.AccessLogQuery) (*repository.AccessLogQueryResult, error) {
	var total int64
	query := r.db.Model(&AccessLog{})
	if q.Method != "" {
		query = query.Where("method = ?", q.Method)
	}
	if q.Path != "" {
		query = query.Where("path LIKE ?", "%"+q.Path+"%")
	}
	if q.StatusMin > 0 {
		query = query.Where("status >= ?", q.StatusMin)
	}
	if q.StatusMax > 0 {
		query = query.Where("status <= ?", q.StatusMax)
	}
	if q.From != "" {
		query = query.Where("timestamp >= ?", q.From)
	}
	if q.To != "" {
		query = query.Where("timestamp <= ?", q.To)
	}
	query.Count(&total)

	order := "id DESC"
	if strings.ToLower(q.Sort) == "asc" {
		order = "id ASC"
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 50
	}
	offset := (q.Page - 1) * q.PageSize

	var logs []AccessLog
	if err := query.Order(order).Limit(q.PageSize).Offset(offset).Find(&logs).Error; err != nil {
		return nil, err
	}

	items := make([]repository.AccessLogRecord, len(logs))
	for i, l := range logs {
		items[i] = repository.AccessLogRecord{
			ID:           l.ID,
			Timestamp:    l.Timestamp,
			Method:       l.Method,
			Path:         l.Path,
			Status:       l.Status,
			DurationMs:   l.DurationMs,
			ClientIP:     l.ClientIP,
			UserAgent:    l.UserAgent,
			RequestBody:  l.RequestBody,
			ResponseBody: l.ResponseBody,
			Error:        l.Error,
		}
	}
	return &repository.AccessLogQueryResult{Items: items, Total: int(total), Page: q.Page, Size: q.PageSize}, nil
}

func (r *AccessLogRepository) DeleteRecord(id int64) error {
	return r.db.Delete(&AccessLog{}, id).Error
}

func (r *AccessLogRepository) ClearRecords() error {
	return r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&AccessLog{}).Error
}

func (r *AccessLogRepository) StartDailyCleanup(days int) {
	go func() {
		for {
			cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
			if err := r.db.Where("timestamp < ?", cutoff).Delete(&AccessLog{}).Error; err != nil {
				log.Printf("[AccessLog] 清理过期日志失败: %v", err)
			}
			time.Sleep(24 * time.Hour)
		}
	}()
}
