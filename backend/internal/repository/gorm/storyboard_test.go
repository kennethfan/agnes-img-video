package gorm

import (
	"testing"
)

func TestStoryboardCreateAndListProjects(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&StoryboardProject{}, &StoryboardShot{})
	repo := NewStoryboardRepository(db)

	id, err := repo.CreateProject("测试项目", "脚本内容")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	projects, err := repo.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Title != "测试项目" {
		t.Fatalf("expected title '测试项目', got %q", projects[0].Title)
	}
}

func TestStoryboardGetProject(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&StoryboardProject{})
	repo := NewStoryboardRepository(db)

	id, _ := repo.CreateProject("get test", "body")
	p, err := repo.GetProject(id)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if p.Title != "get test" {
		t.Fatalf("expected title 'get test', got %q", p.Title)
	}
	if p.Script != "body" {
		t.Fatalf("expected script 'body', got %q", p.Script)
	}
}

func TestStoryboardUpdateProject(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&StoryboardProject{})
	repo := NewStoryboardRepository(db)

	id, _ := repo.CreateProject("old title", "old script")
	if err := repo.UpdateProject(id, "new title", "new script"); err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	p, _ := repo.GetProject(id)
	if p.Title != "new title" {
		t.Fatalf("expected 'new title', got %q", p.Title)
	}
	if p.Script != "new script" {
		t.Fatalf("expected 'new script', got %q", p.Script)
	}
}

func TestStoryboardDeleteProject(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&StoryboardProject{}, &StoryboardShot{})
	repo := NewStoryboardRepository(db)

	id, _ := repo.CreateProject("delete me", "")
	if err := repo.DeleteProject(id); err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	projects, _ := repo.ListProjects()
	if len(projects) != 0 {
		t.Fatal("expected 0 projects after delete")
	}
}

func TestStoryboardDuplicateProject(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&StoryboardProject{}, &StoryboardShot{})
	repo := NewStoryboardRepository(db)

	origID, _ := repo.CreateProject("original", "script")
	repo.CreateShot(origID, 1, "shot1", "text", "")
	repo.CreateShot(origID, 2, "shot2", "image", "")

	dupID, err := repo.DuplicateProject(origID)
	if err != nil {
		t.Fatalf("DuplicateProject failed: %v", err)
	}
	if dupID == origID {
		t.Fatal("duplicate id should differ from original")
	}

	dup, _ := repo.GetProject(dupID)
	if dup.Title != "original (副本)" {
		t.Fatalf("expected 'original (副本)', got %q", dup.Title)
	}

	shots, _ := repo.ListShots(dupID)
	if len(shots) != 2 {
		t.Fatalf("expected 2 shots in duplicate, got %d", len(shots))
	}
}

func TestStoryboardCreateAndListShots(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&StoryboardProject{}, &StoryboardShot{})
	repo := NewStoryboardRepository(db)

	projID, _ := repo.CreateProject("shots test", "")
	for i := range 3 {
		_, err := repo.CreateShot(projID, i, "shot body", "text", "")
		if err != nil {
			t.Fatalf("CreateShot %d failed: %v", i, err)
		}
	}

	shots, err := repo.ListShots(projID)
	if err != nil {
		t.Fatalf("ListShots failed: %v", err)
	}
	if len(shots) != 3 {
		t.Fatalf("expected 3 shots, got %d", len(shots))
	}
	for i, s := range shots {
		if s.Sequence != i {
			t.Fatalf("shot %d: expected sequence %d, got %d", s.ID, i, s.Sequence)
		}
	}
}

func TestStoryboardUpdateShot(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&StoryboardProject{}, &StoryboardShot{})
	repo := NewStoryboardRepository(db)

	projID, _ := repo.CreateProject("update shot", "")
	shotID, _ := repo.CreateShot(projID, 0, "old prompt", "text", "")

	if err := repo.UpdateShot(shotID, "new prompt", "image", "ref.png"); err != nil {
		t.Fatalf("UpdateShot failed: %v", err)
	}

	shots, _ := repo.ListShots(projID)
	if shots[0].Prompt != "new prompt" {
		t.Fatalf("expected 'new prompt', got %q", shots[0].Prompt)
	}
}

func TestStoryboardDeleteShot(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&StoryboardProject{}, &StoryboardShot{})
	repo := NewStoryboardRepository(db)

	projID, _ := repo.CreateProject("delete shot", "")
	shotID, _ := repo.CreateShot(projID, 0, "delete me", "text", "")

	if err := repo.DeleteShot(shotID); err != nil {
		t.Fatalf("DeleteShot failed: %v", err)
	}

	shots, _ := repo.ListShots(projID)
	if len(shots) != 0 {
		t.Fatal("expected 0 shots after delete")
	}
}

func TestStoryboardReorderShots(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&StoryboardProject{}, &StoryboardShot{})
	repo := NewStoryboardRepository(db)

	projID, _ := repo.CreateProject("reorder", "")
	id1, _ := repo.CreateShot(projID, 10, "a", "text", "")
	id2, _ := repo.CreateShot(projID, 20, "b", "text", "")

	if err := repo.ReorderShots([]int64{id2, id1}); err != nil {
		t.Fatalf("ReorderShots failed: %v", err)
	}

	shots, _ := repo.ListShots(projID)
	if shots[0].ID != id2 {
		t.Fatal("expected shot id2 first after reorder")
	}
	if shots[1].ID != id1 {
		t.Fatal("expected shot id1 second after reorder")
	}
}

func TestStoryboardClose(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&StoryboardProject{})
	repo := NewStoryboardRepository(db)

	if err := repo.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
