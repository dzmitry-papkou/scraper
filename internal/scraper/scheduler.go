package scraper

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type Scheduler struct {
	scraper  *Scraper
	ticker   *time.Ticker
	stopChan chan bool
	isActive bool
	mu       sync.Mutex
}

func NewScheduler(scraper *Scraper) *Scheduler {
	return &Scheduler{
		scraper:  scraper,
		stopChan: make(chan bool),
	}
}

func (s *Scheduler) Start(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isActive {
		return
	}

	s.ticker = time.NewTicker(interval)
	s.isActive = true

	go func() {
		count, err := s.scraper.ScrapeOnce()
		if err != nil {
			log.Printf("Scrape error: %v", err)
		} else {
			fmt.Printf("✓ Auto-scraped %d posts\n", count)
		}
	}()

	go func() {
		for {
			select {
			case <-s.ticker.C:
				count, err := s.scraper.ScrapeOnce()
				if err != nil {
					log.Printf("Auto-scrape error: %v", err)
				} else {
					fmt.Printf("\n✓ Auto-scraped %d posts\n➜ ", count)
				}
			case <-s.stopChan:
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isActive {
		return
	}

	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.stopChan <- true
	s.isActive = false
}

func (s *Scheduler) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isActive
}