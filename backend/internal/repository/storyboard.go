package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/agnes-image-tool/backend/internal/model"
	_ "github.com/mattn/go-sqlite3"
)

type StoryboardRepo struct {
	db *sql.DB
}

func NewStoryboardRepo(dbPath string) (*StoryboardRepo, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("打开故事板数据库失败: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS storyboard_projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL DEFAULT '',
			script TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("创建故事板项目表失败: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS storyboard_shots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			sequence INTEGER NOT NULL DEFAULT 0,
			prompt TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'text2video',
			reference_image TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			result_video TEXT DEFAULT '',
			task_id TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (project_id) REFERENCES storyboard_projects(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("创建故事板镜头表失败: %w", err)
	}

	return &StoryboardRepo{db: db}, nil
}

func (r *StoryboardRepo) Close() error {
	return r.db.Close()
}

func (r *StoryboardRepo) ListProjects() ([]model.StoryboardProject, error) {
	rows, err := r.db.Query(`
		SELECT p.id, p.title, p.script, p.created_at, p.updated_at,
			COALESCE((SELECT COUNT(*) FROM storyboard_shots s WHERE s.project_id = p.id), 0) AS shot_count
		FROM storyboard_projects p
		ORDER BY p.updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.StoryboardProject
	for rows.Next() {
		var p model.StoryboardProject
		if err := rows.Scan(&p.ID, &p.Title, &p.Script, &p.CreatedAt, &p.UpdatedAt, &p.ShotCount); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *StoryboardRepo) GetProject(id int64) (*model.StoryboardProject, error) {
	var p model.StoryboardProject
	err := r.db.QueryRow(`
		SELECT p.id, p.title, p.script, p.created_at, p.updated_at,
			COALESCE((SELECT COUNT(*) FROM storyboard_shots s WHERE s.project_id = p.id), 0) AS shot_count
		FROM storyboard_projects p
		WHERE p.id = ?
	`, id).Scan(&p.ID, &p.Title, &p.Script, &p.CreatedAt, &p.UpdatedAt, &p.ShotCount)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *StoryboardRepo) CreateProject(title, script string) (int64, error) {
	res, err := r.db.Exec(
		"INSERT INTO storyboard_projects (title, script) VALUES (?, ?)",
		title, script,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *StoryboardRepo) UpdateProject(id int64, title, script string) error {
	var sets []string
	var args []any

	if title != "" {
		sets = append(sets, "title = ?")
		args = append(args, title)
	}
	if script != "" {
		sets = append(sets, "script = ?")
		args = append(args, script)
	}
	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = datetime('now')")
	args = append(args, id)

	q := fmt.Sprintf("UPDATE storyboard_projects SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := r.db.Exec(q, args...)
	return err
}

func (r *StoryboardRepo) DeleteProject(id int64) error {
	_, err := r.db.Exec("DELETE FROM storyboard_projects WHERE id = ?", id)
	return err
}

func (r *StoryboardRepo) DuplicateProject(id int64) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	var title, script string
	err = tx.QueryRow("SELECT title, script FROM storyboard_projects WHERE id = ?", id).Scan(&title, &script)
	if err != nil {
		return 0, fmt.Errorf("查找源项目失败: %w", err)
	}

	res, err := tx.Exec(
		"INSERT INTO storyboard_projects (title, script) VALUES (?, ?)",
		title+" (副本)", script,
	)
	if err != nil {
		return 0, fmt.Errorf("复制项目失败: %w", err)
	}

	newID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	rows, err := tx.Query(
		"SELECT sequence, prompt, type, reference_image, status, result_video, task_id FROM storyboard_shots WHERE project_id = ? ORDER BY sequence ASC",
		id,
	)
	if err != nil {
		return 0, fmt.Errorf("查询源镜头失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var seq int
		var prompt, shotType, refImage, status, resultVideo, taskID string
		if err := rows.Scan(&seq, &prompt, &shotType, &refImage, &status, &resultVideo, &taskID); err != nil {
			return 0, err
		}
		_, err := tx.Exec(
			"INSERT INTO storyboard_shots (project_id, sequence, prompt, type, reference_image, status, result_video, task_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			newID, seq, prompt, shotType, refImage, status, resultVideo, taskID,
		)
		if err != nil {
			return 0, fmt.Errorf("复制镜头失败: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}
	return newID, nil
}

func (r *StoryboardRepo) ListShots(projectID int64) ([]model.StoryboardShot, error) {
	rows, err := r.db.Query(`
		SELECT id, project_id, sequence, prompt, type, reference_image, status, result_video, task_id, created_at
		FROM storyboard_shots
		WHERE project_id = ?
		ORDER BY sequence ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shots []model.StoryboardShot
	for rows.Next() {
		var s model.StoryboardShot
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Sequence, &s.Prompt, &s.Type, &s.ReferenceImage,
			&s.Status, &s.ResultVideo, &s.TaskID, &s.CreatedAt); err != nil {
			return nil, err
		}
		shots = append(shots, s)
	}
	return shots, rows.Err()
}

func (r *StoryboardRepo) CreateShot(projectID int64, seq int, prompt, shotType, refImage string) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT INTO storyboard_shots (project_id, sequence, prompt, type, reference_image) VALUES (?, ?, ?, ?, ?)",
		projectID, seq, prompt, shotType, refImage,
	)
	if err != nil {
		return 0, err
	}

	shotID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec("UPDATE storyboard_projects SET updated_at = datetime('now') WHERE id = ?", projectID)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}
	return shotID, nil
}

func (r *StoryboardRepo) UpdateShot(id int64, prompt, shotType, refImage string) error {
	_, err := r.db.Exec(
		"UPDATE storyboard_shots SET prompt = ?, type = ?, reference_image = ? WHERE id = ?",
		prompt, shotType, refImage, id,
	)
	return err
}

func (r *StoryboardRepo) DeleteShot(id int64) error {
	_, err := r.db.Exec("DELETE FROM storyboard_shots WHERE id = ?", id)
	return err
}

func (r *StoryboardRepo) ReorderShots(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE storyboard_shots SET sequence = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, id := range ids {
		if _, err := stmt.Exec(i+1, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *StoryboardRepo) UpdateShotStatus(shotID int64, status, resultVideo, taskID string) error {
	_, err := r.db.Exec(
		"UPDATE storyboard_shots SET status = ?, result_video = ?, task_id = ? WHERE id = ?",
		status, resultVideo, taskID, shotID,
	)
	return err
}
