package cli

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/dzmitry-papkou/scraper/internal/database"
)

type Exporter struct {
	repo *database.Repository
}

func NewExporter(repo *database.Repository) *Exporter {
	return &Exporter{
		repo: repo,
	}
}

func (e *Exporter) ExportToCSV() (string, error) {
	filename := fmt.Sprintf("hn_export_%s.csv", time.Now().Format("20060102_150405"))
	
	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"ID", "HN_ID", "Title", "URL", "Author", 
		"Points", "Comments", "PostTime", "ScrapedAt",
	}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write header: %w", err)
	}

	db := database.GetDB()
	query := `
		SELECT id, hn_id, title, url, author, points, comments_count, post_time, scraped_at
		FROM posts
		ORDER BY scraped_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return "", fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, hnID, points, comments int
		var title, url, author string
		var postTime, scrapedAt time.Time

		err := rows.Scan(&id, &hnID, &title, &url, &author, &points, &comments, &postTime, &scrapedAt)
		if err != nil {
			continue
		}

		record := []string{
			strconv.Itoa(id),
			strconv.Itoa(hnID),
			title,
			url,
			author,
			strconv.Itoa(points),
			strconv.Itoa(comments),
			postTime.Format(time.RFC3339),
			scrapedAt.Format(time.RFC3339),
		}

		if err := writer.Write(record); err != nil {
			return "", fmt.Errorf("failed to write record: %w", err)
		}
		count++
	}

	return filename, nil
}