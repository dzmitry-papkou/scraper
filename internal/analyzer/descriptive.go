package analyzer

import (
	"database/sql"
	"fmt"
	"github.com/dzmitry-papkou/scraper/internal/database"
	"github.com/dzmitry-papkou/scraper/internal/models"
)

type DescriptiveAnalyzer struct {
	repo *database.Repository
	db   *sql.DB
}

func NewDescriptiveAnalyzer(repo *database.Repository) *DescriptiveAnalyzer {
	return &DescriptiveAnalyzer{
		repo: repo,
		db:   database.GetDB(),
	}
}

func (a *DescriptiveAnalyzer) BasicStatistics() (map[string]interface{}, error) {
	return a.repo.GetBasicStats()
}

type HourlyPattern struct {
	Hour      int
	PostCount int
	AvgPoints float64
}

func (a *DescriptiveAnalyzer) GetPostingPatterns() ([]HourlyPattern, error) {
	query := `
		SELECT EXTRACT(HOUR FROM post_time) as hour,
		       COUNT(*) as count,
		       AVG(points) as avg_points
		FROM posts
		GROUP BY hour
		ORDER BY hour`

	rows, err := a.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []HourlyPattern
	for rows.Next() {
		var p HourlyPattern
		err := rows.Scan(&p.Hour, &p.PostCount, &p.AvgPoints)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, p)
	}

	return patterns, nil
}

type AuthorStats struct {
	Author    string
	PostCount int
	AvgPoints float64
	MaxPoints int
}

func (a *DescriptiveAnalyzer) GetTopAuthors(minPosts int, limit int) ([]AuthorStats, error) {
	query := `
		SELECT author,
		       COUNT(*) as post_count,
		       AVG(points) as avg_points,
		       MAX(points) as max_points
		FROM posts
		GROUP BY author
		HAVING COUNT(*) >= $1
		ORDER BY avg_points DESC
		LIMIT $2`

	rows, err := a.db.Query(query, minPosts, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var authors []AuthorStats
	for rows.Next() {
		var auth AuthorStats
		err := rows.Scan(&auth.Author, &auth.PostCount, &auth.AvgPoints, &auth.MaxPoints)
		if err != nil {
			return nil, err
		}
		authors = append(authors, auth)
	}

	return authors, nil
}

func (a *DescriptiveAnalyzer) GetTopPosts(limit int) ([]models.Post, error) {
	return a.repo.GetTopPosts(limit)
}

type DailyTrend struct {
	Date         string
	PostCount    int
	AvgPoints    float64
	AvgComments  float64
}

func (a *DescriptiveAnalyzer) GetDailyTrends(days int) ([]DailyTrend, error) {
	query := fmt.Sprintf(`
		SELECT DATE(post_time)::text as date,
		       COUNT(*) as posts,
		       COALESCE(AVG(points), 0) as avg_points,
		       COALESCE(AVG(comments_count), 0) as avg_comments
		FROM posts
		WHERE post_time > CURRENT_DATE - INTERVAL '%d days'
		GROUP BY DATE(post_time)
		ORDER BY date DESC`, days)

	rows, err := a.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []DailyTrend
	for rows.Next() {
		var t DailyTrend
		err := rows.Scan(&t.Date, &t.PostCount, &t.AvgPoints, &t.AvgComments)
		if err != nil {
			return nil, err
		}
		trends = append(trends, t)
	}

	return trends, nil
}

type Distribution struct {
	Min        float64
	Max        float64
	Mean       float64
	Median     float64
	StdDev     float64
	Percentile25 float64
	Percentile75 float64
}

func (a *DescriptiveAnalyzer) GetPointsDistribution() (*Distribution, error) {
	dist := &Distribution{}

	var stddev sql.NullFloat64
	err := a.db.QueryRow(`
		SELECT COALESCE(MIN(points), 0), 
		       COALESCE(MAX(points), 0), 
		       COALESCE(AVG(points), 0), 
		       STDDEV(points)
		FROM posts
		WHERE points > 0`).Scan(&dist.Min, &dist.Max, &dist.Mean, &stddev)
	if err != nil {
		return nil, err
	}
	
	if stddev.Valid {
		dist.StdDev = stddev.Float64
	}

	err = a.db.QueryRow(`
		SELECT 
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY points) as median,
			PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY points) as q1,
			PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY points) as q3
		FROM posts
		WHERE points > 0`).Scan(&dist.Median, &dist.Percentile25, &dist.Percentile75)
	if err != nil {
		return nil, err
	}

	return dist, nil
}