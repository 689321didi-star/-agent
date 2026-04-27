package model

import (
	"time"

	"gorm.io/gorm"
)

// TaskStatus 任务状态枚举
type TaskStatus int8

const (
	TaskStatusPending TaskStatus = iota // 0: 等待调度
	TaskStatusRunning                   // 1: 执行中
	TaskStatusDone                      // 2: 完成
	TaskStatusFailed                    // 3: 失败
	TaskStatusRetry                     // 4: 等待重试
)

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusPending:
		return "pending"
	case TaskStatusRunning:
		return "running"
	case TaskStatusDone:
		return "done"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusRetry:
		return "retry"
	default:
		return "unknown"
	}
}

// Task 任务主表
type Task struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID     string         `gorm:"uniqueIndex;size:64"      json:"task_id"`
	UserID     uint64         `gorm:"index"                    json:"user_id"`
	Title      string         `gorm:"size:256"                 json:"title"`
	Goal       string         `gorm:"type:text"                json:"goal"`
	Status     TaskStatus     `gorm:"index"                    json:"status"`
	Priority   int8           `gorm:"default:5"                json:"priority"`
	RetryCount int8           `gorm:"default:0"                json:"retry_count"`
	MaxRetries int8           `gorm:"default:3"                json:"max_retries"`
	Deadline   *time.Time     `                                json:"deadline,omitempty"`
	ResultURL  string         `gorm:"size:512"                 json:"result_url,omitempty"`
	ErrorMsg   string         `gorm:"type:text"                json:"error_msg,omitempty"`
	CreatedAt  time.Time      `                                json:"created_at"`
	UpdatedAt  time.Time      `                                json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index"                    json:"-"`
}

// User 用户表
type User struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string         `gorm:"uniqueIndex;size:64"      json:"username"`
	Email     string         `gorm:"uniqueIndex;size:128"     json:"email"`
	Password  string         `gorm:"size:256"                 json:"-"`
	Level     int8           `gorm:"default:1"                json:"level"`
	CreatedAt time.Time      `                                json:"created_at"`
	UpdatedAt time.Time      `                                json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                    json:"-"`
}
