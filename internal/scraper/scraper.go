package scraper

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dzmitry-papkou/scraper/internal/config"
	"github.com/dzmitry-papkou/scraper/internal/database"
	"github.com/dzmitry-papkou/scraper/internal/models"
)

type Scraper struct {
	repo   *database.Repository
	config *config.ScraperConfig
	parser *Parser
}

func New(repo *database.Repository) *Scraper {
	scraperConfig, _ := config.GetScraper("hackernews")
	if scraperConfig == nil {
		scraperConfig = &config.ScraperConfig{
			Name:    "hackernews",
			URL:     "https://news.ycombinator.com/newest",
			Enabled: true,
		}
	}

	return &Scraper{
		repo:   repo,
		config: scraperConfig,
		parser: NewParser(),
	}
}

func NewWithConfig(repo *database.Repository, scraperConfig *config.ScraperConfig) *Scraper {
	return &Scraper{
		repo:   repo,
		config: scraperConfig,
		parser: NewParser(),
	}
}

func NewGenericScraper(repo *database.Repository, scraperName string) (*Scraper, error) {
	scraperConfig, err := config.GetScraper(scraperName)
	if err != nil {
		return nil, fmt.Errorf("scraper %s not found in config: %w", scraperName, err)
	}

	return &Scraper{
		repo:   repo,
		config: scraperConfig,
		parser: NewParser(),
	}, nil
}

func (s *Scraper) ScrapeOnce() (int, error) {
	startTime := time.Now()
	log.Printf("Scraping %s from %s", s.config.Name, s.config.URL)

	jobID, err := s.repo.CreateScrapingJob()
	if err != nil {
		return 0, fmt.Errorf("failed to create job: %w", err)
	}

	posts, err := s.fetchAndParse()
	if err != nil {
		s.repo.UpdateScrapingJob(jobID, "failed", 0, err.Error())
		return 0, fmt.Errorf("failed to fetch/parse: %w", err)
	}

	saved := 0
	for _, post := range posts {
		if post.PostTime.IsZero() || post.PostTime.Year() < 2000 {
			log.Printf("WARNING: Post %d has invalid time %v, using current time", post.HnID, post.PostTime)
			post.PostTime = time.Now()
		}

		if err := s.repo.InsertPost(&post); err != nil {
			log.Printf("Failed to insert post %d: %v", post.HnID, err)
			continue
		}
		saved++

		if post.ID > 0 {
			s.repo.InsertPostHistory(post.ID, post.Points, post.CommentsCount)
		}
	}

	s.repo.UpdateScrapingJob(jobID, "completed", saved, "")

	duration := time.Since(startTime)
	log.Printf("Scraped %d posts from %s in %.2f seconds", saved, s.config.Name, duration.Seconds())

	return saved, nil
}

func (s *Scraper) fetchAndParse() ([]models.Post, error) {
	resp, err := http.Get(s.config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()


	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}

	if s.config.Name == "hackernews" {
		return s.parser.ParseDocument(doc)
	}

	return s.parser.ParseDocument(doc)
}

func (s *Scraper) GetConfig() *config.ScraperConfig {
	return s.config
}