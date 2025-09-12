package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dzmitry-papkou/scraper/internal/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository() *Repository {
	return &Repository{
		db: GetDB(),
	}
}

// posts operations

func (r *Repository) InsertPost(post *models.Post) error {
	query := `
		INSERT INTO posts (hn_id, title, url, author, points, comments_count, post_time, scraped_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (hn_id) DO UPDATE SET
			points = EXCLUDED.points,
			comments_count = EXCLUDED.comments_count,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id`

	err := r.db.QueryRow(query,
		post.HnID, post.Title, post.URL, post.Author,
		post.Points, post.CommentsCount, post.PostTime, time.Now(),
	).Scan(&post.ID)

	return err
}

func (r *Repository) GetRecentPosts(limit int) ([]models.Post, error) {
	query := `
		SELECT id, hn_id, title, url, author, points, comments_count, post_time, scraped_at
		FROM posts
		ORDER BY post_time DESC
		LIMIT $1`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		err := rows.Scan(&p.ID, &p.HnID, &p.Title, &p.URL, &p.Author,
			&p.Points, &p.CommentsCount, &p.PostTime, &p.ScrapedAt)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	return posts, nil
}

func (r *Repository) GetPostCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count)
	return count, err
}

// post history operations

func (r *Repository) InsertPostHistory(postID int, points, comments int) error {
	query := `
		INSERT INTO post_history (post_id, points, comments_count)
		VALUES ($1, $2, $3)`
	
	_, err := r.db.Exec(query, postID, points, comments)
	return err
}

// scraping job operations

func (r *Repository) CreateScrapingJob() (int, error) {
	var jobID int
	query := `
		INSERT INTO scraping_jobs (started_at, status)
		VALUES ($1, 'running')
		RETURNING id`

	err := r.db.QueryRow(query, time.Now()).Scan(&jobID)
	return jobID, err
}

func (r *Repository) UpdateScrapingJob(jobID int, status string, postsScraped int, errorMsg string) error {
	query := `
		UPDATE scraping_jobs
		SET status = $1, posts_scraped = $2, error_message = NULLIF($3, ''), 
		    completed_at = CURRENT_TIMESTAMP
		WHERE id = $4`

	_, err := r.db.Exec(query, status, postsScraped, errorMsg, jobID)
	return err
}

func (r *Repository) GetLastScrapingJob() (*models.ScrapingJob, error) {
	var job models.ScrapingJob
	query := `
		SELECT id, started_at, completed_at, status, posts_scraped, error_message
		FROM scraping_jobs
		WHERE status = 'completed'
		ORDER BY completed_at DESC
		LIMIT 1`

	err := r.db.QueryRow(query).Scan(
		&job.ID, &job.StartedAt, &job.CompletedAt,
		&job.Status, &job.PostsScraped, &job.ErrorMessage,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &job, err
}

// statistics operations

func (r *Repository) GetBasicStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalPosts int
	r.db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&totalPosts)
	stats["total_posts"] = totalPosts

	var uniqueAuthors int
	r.db.QueryRow("SELECT COUNT(DISTINCT author) FROM posts").Scan(&uniqueAuthors)
	stats["unique_authors"] = uniqueAuthors

	var avgPoints, avgComments sql.NullFloat64
	r.db.QueryRow("SELECT AVG(points), AVG(comments_count) FROM posts").Scan(&avgPoints, &avgComments)

	stats["avg_points"] = avgPoints.Float64
	stats["avg_comments"] = avgComments.Float64

	var maxPoints, maxComments int
	r.db.QueryRow("SELECT COALESCE(MAX(points), 0) FROM posts").Scan(&maxPoints)
	r.db.QueryRow("SELECT COALESCE(MAX(comments_count), 0) FROM posts").Scan(&maxComments)
	stats["max_points"] = maxPoints
	stats["max_comments"] = maxComments

	return stats, nil
}

func (r *Repository) GetTopPosts(limit int) ([]models.Post, error) {
	query := `
		SELECT id, hn_id, title, url, author, points, comments_count, post_time, scraped_at
		FROM posts
		ORDER BY points DESC
		LIMIT $1`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		err := rows.Scan(&p.ID, &p.HnID, &p.Title, &p.URL, &p.Author,
			&p.Points, &p.CommentsCount, &p.PostTime, &p.ScrapedAt)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	return posts, nil
}

// analysis queries

func (r *Repository) GetCorrelation(field1, field2 string) (float64, error) {
	var correlation sql.NullFloat64
	query := fmt.Sprintf(`
		SELECT CORR(%s::numeric, %s::numeric)
		FROM posts
		WHERE %s > 0 AND %s > 0`,
		field1, field2, field1, field2)
	
	err := r.db.QueryRow(query).Scan(&correlation)
	if err != nil || !correlation.Valid {
		return 0, err
	}
	return correlation.Float64, nil
}

func (r *Repository) GetWeekdayWeekendStats() (weekdayAvg, weekendAvg float64, weekdayCount, weekendCount int, err error) {
	err = r.db.QueryRow(`
		SELECT COUNT(*), COALESCE(AVG(points), 0)
		FROM posts
		WHERE EXTRACT(DOW FROM post_time) IN (1,2,3,4,5)`).Scan(&weekdayCount, &weekdayAvg)
	if err != nil {
		return
	}

	err = r.db.QueryRow(`
		SELECT COUNT(*), COALESCE(AVG(points), 0)
		FROM posts
		WHERE EXTRACT(DOW FROM post_time) IN (0,6)`).Scan(&weekendCount, &weekendAvg)
	
	return
}


func (r *Repository) GetLatestHNPostID() (int, error) {
	var maxID int
	err := r.db.QueryRow(`
		SELECT COALESCE(MAX(hn_id), 0) 
		FROM posts 
	`).Scan(&maxID)
	return maxID, err
}

func (r *Repository) PostExists(hnID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM posts WHERE hn_id = $1)
	`, hnID).Scan(&exists)
	return exists, err
}

func (r *Repository) UpdatePost(post *models.Post) error {
	query := `
		UPDATE posts 
		SET points = $1, 
		    comments_count = $2,
		    updated_at = CURRENT_TIMESTAMP,
		    last_seen = CURRENT_TIMESTAMP
		WHERE hn_id = $3`
	
	_, err := r.db.Exec(query, post.Points, post.CommentsCount, post.HnID)
	
	if err == nil {
		r.recordPostHistory(post.HnID, post.Points, post.CommentsCount)
	}
	
	return err
}

func (r *Repository) GetRecentPostsNotUpdatedSince(since time.Time, limit int) ([]models.Post, error) {
	query := `
		SELECT id, hn_id, title, url, author, points, comments_count, post_time, scraped_at
		FROM posts
		WHERE updated_at < $1 
		  AND post_time > CURRENT_TIMESTAMP - INTERVAL '7 days'
		ORDER BY post_time DESC
		LIMIT $2`
	
	rows, err := r.db.Query(query, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var posts []models.Post
	for rows.Next() {
		var p models.Post
		err := rows.Scan(&p.ID, &p.HnID, &p.Title, &p.URL, &p.Author,
			&p.Points, &p.CommentsCount, &p.PostTime, &p.ScrapedAt)
		if err != nil {
			continue
		}
		posts = append(posts, p)
	}
	
	return posts, nil
}

func (r *Repository) GetPostsSinceID(hnID int) ([]models.Post, error) {
	query := `
		SELECT id, hn_id, title, url, author, points, comments_count, post_time, scraped_at
		FROM posts
		WHERE hn_id > $1
		ORDER BY hn_id DESC`
	
	rows, err := r.db.Query(query, hnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var posts []models.Post
	for rows.Next() {
		var p models.Post
		err := rows.Scan(&p.ID, &p.HnID, &p.Title, &p.URL, &p.Author,
			&p.Points, &p.CommentsCount, &p.PostTime, &p.ScrapedAt)
		if err != nil {
			continue
		}
		posts = append(posts, p)
	}
	
	return posts, nil
}

func (r *Repository) recordPostHistory(hnID, points, comments int) error {
	var postID int
	err := r.db.QueryRow("SELECT id FROM posts WHERE hn_id = $1", hnID).Scan(&postID)
	if err != nil {
		return err
	}
	
	query := `
		INSERT INTO post_history (post_id, points, comments_count)
		VALUES ($1, $2, $3)`
	
	_, err = r.db.Exec(query, postID, points, comments)
	return err
}

func (r *Repository) CreateDetailedScrapingJob(result interface{}) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}
	
	query := `
		INSERT INTO scraping_jobs (
			started_at, 
			completed_at, 
			status, 
			posts_scraped, 
			details
		) VALUES ($1, $2, $3, $4, $5)`
	
	// extract basic fields from result need proper type assertion based on ScrapingResult
	_, err = r.db.Exec(query, 
		time.Now(), 
		time.Now(), 
		"completed", 
		0,
		string(resultJSON))
	
	return err
}

func (r *Repository) GetScrapingHistory(limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			id,
			started_at,
			completed_at,
			status,
			posts_scraped,
			details
		FROM scraping_jobs
		ORDER BY started_at DESC
		LIMIT $1`
	
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var history []map[string]interface{}
	for rows.Next() {
		var id int
		var startedAt, completedAt sql.NullTime
		var status string
		var postsScraped int
		var details sql.NullString
		
		err := rows.Scan(&id, &startedAt, &completedAt, &status, &postsScraped, &details)
		if err != nil {
			continue
		}
		
		job := map[string]interface{}{
			"id":            id,
			"started_at":    startedAt.Time,
			"completed_at":  completedAt.Time,
			"status":        status,
			"posts_scraped": postsScraped,
		}
		
		if details.Valid {
			var detailsMap map[string]interface{}
			if err := json.Unmarshal([]byte(details.String), &detailsMap); err == nil {
				job["details"] = detailsMap
			}
		}
		
		history = append(history, job)
	}
	
	return history, nil
}