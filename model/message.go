package model

import (
	"time"
)

type Message struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID string    `gorm:"index;not null;type:varchar(36)" json:"session_id"`
	UserName  string    `gorm:"type:varchar(20)" json:"username"`
	Content   string    `gorm:"type:text" json:"content"`
	IsUser    bool      `gorm:"not null;" json:"is_user"`
	CreatedAt time.Time `json:"created_at"`
}

type History struct {
	IsUser  bool   `json:"is_user"`
	Content string `json:"content"`
}

type TavilyResponse struct {
	Query   string `json:"query"`
	Answer  string `json:"answer"`
	Results []struct {
		Title   string  `json:"title"`
		URL     string  `json:"url"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
}

type TavilyRequest struct {
	Query         string `json:"query"`
	SearchDepth   string `json:"search_depth,omitempty"`
	IncludeAnswer bool   `json:"include_answer"`
	MaxResults    int    `json:"max_results"`
}
