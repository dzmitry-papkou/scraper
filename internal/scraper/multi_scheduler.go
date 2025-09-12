package scraper

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dzmitry-papkou/scraper/internal/database"
)

type ScraperJob struct {
	Scraper  *Scraper
	Ticker   *time.Ticker
	StopChan chan bool
	IsActive bool
}

type MultiScheduler struct {
	repo     *database.Repository
	scrapers map[string]*ScraperJob
	mu       sync.RWMutex
}

func NewMultiScheduler(repo *database.Repository) *MultiScheduler {
	return &MultiScheduler{
		repo:     repo,
		scrapers: make(map[string]*ScraperJob),
	}
}

func (s *MultiScheduler) StartScraper(name string, interval time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, exists := s.scrapers[name]; exists && job.IsActive {
		return fmt.Errorf("scraper %s is already running", name)
	}

	scraperInstance, err := NewGenericScraper(s.repo, name)
	if err != nil {
		return fmt.Errorf("failed to create scraper %s: %w", name, err)
	}

	job := &ScraperJob{
		Scraper:  scraperInstance,
		Ticker:   time.NewTicker(interval),
		StopChan: make(chan bool),
		IsActive: true,
	}

	s.scrapers[name] = job

	go func() {
		count, err := scraperInstance.ScrapeOnce()
		if err != nil {
			log.Printf("Error scraping %s: %v", name, err)
		} else {
			fmt.Printf("✓ Auto-scraped %d posts from %s\n", count, name)
		}
	}()

	go func() {
		for {
			select {
			case <-job.Ticker.C:
				count, err := scraperInstance.ScrapeOnce()
				if err != nil {
					log.Printf("Auto-scrape error for %s: %v", name, err)
				} else {
					fmt.Printf("\n✓ Auto-scraped %d posts from %s\n➜ ", count, name)
				}
			case <-job.StopChan:
				return
			}
		}
	}()

	log.Printf("Started scheduler for %s with interval %s", name, interval)
	return nil
}

func (s *MultiScheduler) StopScraper(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.scrapers[name]
	if !exists || !job.IsActive {
		return fmt.Errorf("scraper %s is not running", name)
	}

	job.Ticker.Stop()
	close(job.StopChan)
	job.IsActive = false

	log.Printf("Stopped scheduler for %s", name)
	return nil
}

func (s *MultiScheduler) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, job := range s.scrapers {
		if job.IsActive {
			job.Ticker.Stop()
			close(job.StopChan)
			job.IsActive = false
			log.Printf("Stopped scheduler for %s", name)
		}
	}
}

func (s *MultiScheduler) IsActive(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.scrapers[name]
	return exists && job.IsActive
}

func (s *MultiScheduler) GetActiveScrapers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []string
	for name, job := range s.scrapers {
		if job.IsActive {
			active = append(active, name)
		}
	}
	return active
}