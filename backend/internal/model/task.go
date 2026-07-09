package model

// ==================== 任务类型 ====================

type TaskType string

const (
	TaskTypeTextToImage     TaskType = "text2image"
	TaskTypeImageToImage    TaskType = "image2image"
	TaskTypeBatch           TaskType = "batch"
	TaskTypeTextToVideo     TaskType = "text2video"
	TaskTypeImageToVideo    TaskType = "image2video"
	TaskTypeMultiImageVideo TaskType = "multi_image_video"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)
