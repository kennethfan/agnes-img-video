package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var modeFromPrefix = map[string]string{
	"text2img":              "text2image",
	"img2img":               "image2image",
	"batch":                 "batch",
	"video_text2video":      "text2video",
	"video_image2video":     "image2video",
	"video_multi_image_video": "multi_image_video",
}

var tsRe = regexp.MustCompile(`(\d{8}_\d{6})`)

func main() {
	dbPath := "history.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	importFromJSON(db, "bin/history.json")

	outputsDir := "outputs"
	entries, err := os.ReadDir(outputsDir)
	if err != nil {
		log.Fatalf("读取 outputs 目录失败: %v", err)
	}

	recovered := 0
	for _, entry := range entries {
		name := entry.Name()
		tsMatch := tsRe.FindStringSubmatch(name)
		if tsMatch == nil {
			continue
		}
		tsStr := tsMatch[1]

		mode := detectMode(name)
		if mode == "" {
			continue
		}

		t, err := time.Parse("20060102_150405", tsStr)
		if err != nil {
			continue
		}
		timeStr := t.Format("2006-01-02 15:04:05")

		relPath := "outputs/" + name
		var count int
		db.QueryRow("SELECT COUNT(*) FROM history WHERE images LIKE ?", "%"+relPath+"%").Scan(&count)
		if count > 0 {
			continue
		}

		images := []string{relPath}
		imagesJSON, _ := json.Marshal(images)
		prompt := fmt.Sprintf("[已恢复] %s - %s", modeMapLabel(mode), t.Format("2006-01-02 15:04"))

		_, err = db.Exec(
			"INSERT INTO history (time, mode, prompt, images) VALUES (?, ?, ?, ?)",
			timeStr, mode, prompt, string(imagesJSON),
		)
		if err != nil {
			log.Printf("插入失败 %s: %v", name, err)
			continue
		}
		recovered++
		fmt.Printf("  ✓ %s → %s\n", name, mode)
	}

	fmt.Printf("\n成功恢复 %d 条记录\n", recovered)
}

func detectMode(filename string) string {
	for prefix, mode := range modeFromPrefix {
		if len(filename) >= len(prefix) && filename[:len(prefix)] == prefix {
			return mode
		}
	}
	return ""
}

func modeMapLabel(mode string) string {
	labels := map[string]string{
		"text2image":        "文生图",
		"image2image":       "图生图",
		"batch":             "批量生成",
		"text2video":        "文生视频",
		"image2video":       "图生视频",
		"multi_image_video": "多图视频",
	}
	if l, ok := labels[mode]; ok {
		return l
	}
	return mode
}

func importFromJSON(db *sql.DB, jsonPath string) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return
	}

	var records []struct {
		Time   string   `json:"time"`
		Mode   string   `json:"mode"`
		Prompt string   `json:"prompt"`
		Images []string `json:"images"`
	}
	if err := json.Unmarshal(data, &records); err != nil {
		log.Printf("解析 %s 失败: %v", jsonPath, err)
		return
	}

	imported := 0
	for _, rec := range records {
		if rec.Time == "" || rec.Prompt == "" {
			continue
		}
		var count int
		db.QueryRow("SELECT COUNT(*) FROM history WHERE time = ? AND prompt = ?", rec.Time, rec.Prompt).Scan(&count)
		if count > 0 {
			continue
		}
		imagesJSON, _ := json.Marshal(rec.Images)
		_, err := db.Exec(
			"INSERT INTO history (time, mode, prompt, images) VALUES (?, ?, ?, ?)",
			rec.Time, rec.Mode, rec.Prompt, string(imagesJSON),
		)
		if err != nil {
			log.Printf("导入 %s 失败: %v", jsonPath, err)
			continue
		}
		imported++
	}
	if imported > 0 {
		fmt.Printf("从 %s 导入了 %d 条记录\n", jsonPath, imported)
	}
}
