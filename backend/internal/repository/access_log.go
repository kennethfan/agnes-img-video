package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type AccessLogRecord struct {
	ID           int64  `json:"id"`
	Timestamp    string `json:"timestamp"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	Status       int    `json:"status"`
	DurationMs   int    `json:"duration_ms"`
	ClientIP     string `json:"client_ip"`
	UserAgent    string `json:"user_agent"`
	RequestBody  string `json:"request_body"`
	ResponseBody string `json:"response_body"`
	Error        string `json:"error"`
}

type AccessLogQuery struct {
	Page      int
	PageSize  int
	Method    string
	Path      string
	StatusMin int
	StatusMax int
	From      string
	To        string
	Sort      string // asc or desc
}

type AccessLogQueryResult struct {
	Items []AccessLogRecord `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"page_size"`
}

type AccessLogRepo struct {
	db *sql.DB
}

func NewAccessLogRepo(db *sql.DB) (*AccessLogRepo, error) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS access_logs (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp    TEXT NOT NULL,
			method       TEXT NOT NULL,
			path         TEXT NOT NULL,
			status       INTEGER NOT NULL,
			duration_ms  INTEGER NOT NULL DEFAULT 0,
			client_ip    TEXT NOT NULL DEFAULT '',
			user_agent   TEXT NOT NULL DEFAULT '',
			request_body TEXT NOT NULL DEFAULT '',
			response_body TEXT NOT NULL DEFAULT '',
			error        TEXT NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("创建 access_logs 表失败: %w", err)
	}

	// 索引
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_access_logs_timestamp ON access_logs(timestamp)`)
	if err != nil {
		return nil, fmt.Errorf("创建 timestamp 索引失败: %w", err)
	}

	return &AccessLogRepo{db: db}, nil
}

func (r *AccessLogRepo) Insert(record *AccessLogRecord) error {
	_, err := r.db.Exec(
		`INSERT INTO access_logs (timestamp, method, path, status, duration_ms, client_ip, user_agent, request_body, response_body, error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.Timestamp, record.Method, record.Path, record.Status, record.DurationMs, record.ClientIP, record.UserAgent, record.RequestBody, record.ResponseBody, record.Error,
	)
	if err != nil {
		return fmt.Errorf("插入访问日志失败: %w", err)
	}
	return nil
}

func (r *AccessLogRepo) Query(q AccessLogQuery) (*AccessLogQueryResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 200 {
		q.PageSize = 50
	}

	var where []string
	var args []any

	if q.Method != "" {
		where = append(where, "method = ?")
		args = append(args, strings.ToUpper(q.Method))
	}
	if q.Path != "" {
		where = append(where, "path LIKE ?")
		args = append(args, "%"+q.Path+"%")
	}
	if q.StatusMin > 0 {
		where = append(where, "status >= ?")
		args = append(args, q.StatusMin)
	}
	if q.StatusMax > 0 {
		where = append(where, "status <= ?")
		args = append(args, q.StatusMax)
	}
	if q.From != "" {
		where = append(where, "timestamp >= ?")
		args = append(args, q.From)
	}
	if q.To != "" {
		where = append(where, "timestamp <= ?")
		args = append(args, q.To)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}

	// count
	var total int
	countQuery := "SELECT COUNT(*) FROM access_logs" + whereClause
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("查询日志总数失败: %w", err)
	}

	// sort
	sortDir := "DESC"
	if strings.EqualFold(q.Sort, "asc") {
		sortDir = "ASC"
	}

	offset := (q.Page - 1) * q.PageSize
	dataQuery := fmt.Sprintf(
		"SELECT id, timestamp, method, path, status, duration_ms, client_ip, user_agent, request_body, response_body, error FROM access_logs%s ORDER BY id %s LIMIT ? OFFSET ?",
		whereClause, sortDir,
	)
	dataArgs := append(args, q.PageSize, offset)

	rows, err := r.db.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("查询访问日志失败: %w", err)
	}
	defer rows.Close()

	var items []AccessLogRecord
	for rows.Next() {
		var rec AccessLogRecord
		if err := rows.Scan(&rec.ID, &rec.Timestamp, &rec.Method, &rec.Path, &rec.Status, &rec.DurationMs, &rec.ClientIP, &rec.UserAgent, &rec.RequestBody, &rec.ResponseBody, &rec.Error); err != nil {
			return nil, fmt.Errorf("扫描日志记录失败: %w", err)
		}
		items = append(items, rec)
	}

	return &AccessLogQueryResult{
		Items: items,
		Total: total,
		Page:  q.Page,
		Size:  q.PageSize,
	}, nil
}

func (r *AccessLogRepo) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM access_logs WHERE id = ?", id)
	return err
}

func (r *AccessLogRepo) ClearAll() error {
	_, err := r.db.Exec("DELETE FROM access_logs")
	return err
}

// DeleteOlderThan 删除指定天数前的日志
func (r *AccessLogRepo) DeleteOlderThan(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format(time.RFC3339)
	result, err := r.db.Exec("DELETE FROM access_logs WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("清理旧日志失败: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

// StartDailyCleanup 启动每日自动清理 goroutine
func (r *AccessLogRepo) StartDailyCleanup(retentionDays int) {
	go func() {
		for {
			// 每天凌晨 3 点执行
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 3, 0, 0, 0, now.Location())
			duration := next.Sub(now)
			time.Sleep(duration)

			n, err := r.DeleteOlderThan(retentionDays)
			if err != nil {
				log.Printf("[AccessLog] 清理失败: %v", err)
			} else if n > 0 {
				log.Printf("[AccessLog] 已清理 %d 条过期日志", n)
			}
		}
	}()
}
