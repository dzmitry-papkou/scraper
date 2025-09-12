package models

import (
	"time"
)

type Post struct {
	ID            int       `db:"id"`
	HnID          int       `db:"hn_id"`
	Title         string    `db:"title"`
	URL           string    `db:"url"`
	Author        string    `db:"author"`
	Points        int       `db:"points"`
	CommentsCount int       `db:"comments_count"`
	PostTime      time.Time `db:"post_time"`
	ScrapedAt     time.Time `db:"scraped_at"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type PostHistory struct {
	ID            int       `db:"id"`
	PostID        int       `db:"post_id"`
	Points        int       `db:"points"`
	CommentsCount int       `db:"comments_count"`
	RecordedAt    time.Time `db:"recorded_at"`
}


type ScrapingJob struct {
	ID           int        `db:"id"`
	StartedAt    time.Time  `db:"started_at"`
	CompletedAt  *time.Time `db:"completed_at"`
	Status       string     `db:"status"`
	PostsScraped int        `db:"posts_scraped"`
	ErrorMessage *string    `db:"error_message"`
}

type AnalysisResult struct {
	ID           int       `db:"id"`
	AnalysisType string    `db:"analysis_type"`
	AnalysisDate time.Time `db:"analysis_date"`
	Results      string    `db:"results"`
	CreatedAt    time.Time `db:"created_at"`
}