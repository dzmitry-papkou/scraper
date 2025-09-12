package scraper

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dzmitry-papkou/scraper/internal/config"
	"github.com/dzmitry-papkou/scraper/internal/database"
	"github.com/dzmitry-papkou/scraper/internal/models"
)

type SmartScraper struct {
	repo            *database.Repository
	config          *config.ScraperConfig
	parser          *Parser
	mode            ScrapingMode
	maxPages        int
	stopOnDuplicate bool
}

type ScrapingMode string

const (
	ModeLatestOnly    ScrapingMode = "latest"
	ModeUntilExisting ScrapingMode = "until_existing"
	ModeFullArchive   ScrapingMode = "full"
	ModeSinceLast     ScrapingMode = "since_last"
)

func NewSmartScraper(repo *database.Repository, scraperConfig *config.ScraperConfig, mode ScrapingMode, maxPages int) *SmartScraper {
	return &SmartScraper{
		repo:            repo,
		config:          scraperConfig,
		parser:          NewParser(),
		mode:            mode,
		maxPages:        maxPages,
		stopOnDuplicate: mode == ModeUntilExisting || mode == ModeSinceLast,
	}
}

func (s *SmartScraper) ScrapeWithStrategy() (*ScrapingResult, error) {
	result := &ScrapingResult{
		StartTime: time.Now(),
		Mode:      s.mode,
	}

	lastKnownID, err := s.repo.GetLatestHNPostID()
	if err != nil {
		log.Printf("Warning: Could not get latest post ID: %v", err)
	}
	result.LastKnownID = lastKnownID

	log.Printf("Starting %s scrape for %s. Last known post ID: %d", s.mode, s.config.Name, lastKnownID)

	switch s.mode {
	case ModeLatestOnly:
		err = s.scrapeLatestPage(result)
	case ModeUntilExisting:
		err = s.scrapeUntilExisting(result)
	case ModeSinceLast:
		err = s.scrapeSinceLast(result, lastKnownID)
	case ModeFullArchive:
		err = s.scrapeFullArchive(result)
	default:
		err = s.scrapeLatestPage(result)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	s.saveScrapingResult(result)

	return result, err
}

func (s *SmartScraper) scrapeLatestPage(result *ScrapingResult) error {
	posts, err := s.scrapePage(s.config.URL, 1)
	if err != nil {
		return err
	}

	saved := s.savePosts(posts, result)
	result.PostsScraped += saved
	result.PagesScraped = 1

	return nil
}

func (s *SmartScraper) scrapeSinceLast(result *ScrapingResult, lastKnownID int) error {
	allNewPosts := []models.Post{}
	foundLastKnown := false

	for page := 1; page <= s.maxPages && !foundLastKnown; page++ {
		url := s.buildPageURL(page)
		posts, err := s.scrapePage(url, page)
		if err != nil {
			log.Printf("Error scraping page %d: %v", page, err)
			break
		}

		for _, post := range posts {
			if post.HnID <= lastKnownID {
				foundLastKnown = true
				break
			}
			allNewPosts = append(allNewPosts, post)
		}

		result.PagesScraped = page
		time.Sleep(1 * time.Second)
	}

	for _, post := range allNewPosts {
		if err := s.repo.InsertPost(&post); err == nil {
			result.PostsScraped++
			result.NewPosts++
		}
	}

	log.Printf("Found %d new posts since ID %d", len(allNewPosts), lastKnownID)
	return nil
}

func (s *SmartScraper) scrapePage(url string, pageNum int) ([]models.Post, error) {
	log.Printf("Scraping page %d: %s", pageNum, url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page: %w", err)
	}

	posts, err := s.parser.ParseDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page: %w", err)
	}

	for i := range posts {
		if posts[i].PostTime.IsZero() || posts[i].PostTime.Year() < 2000 {
			log.Printf("Warning: Post %d has invalid time, using current time", posts[i].HnID)
			posts[i].PostTime = time.Now()
		}
	}

	return posts, nil
}

func (s *SmartScraper) savePosts(posts []models.Post, result *ScrapingResult) int {
	saved := 0
	for _, post := range posts {
		exists, _ := s.repo.PostExists(post.HnID)
		
		if exists {
			if err := s.repo.UpdatePost(&post); err == nil {
				result.UpdatedPosts++
			}
		} else {
			if err := s.repo.InsertPost(&post); err == nil {
				saved++
				result.NewPosts++
			}
		}

		if post.HnID > result.HighestIDSeen {
			result.HighestIDSeen = post.HnID
		}
	}
	return saved
}

type ScrapingResult struct {
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	Mode           ScrapingMode
	PagesScraped   int
	PostsScraped   int
	NewPosts       int
	UpdatedPosts   int
	DeletedPosts   int
	LastKnownID    int
	HighestIDSeen  int
	Errors         []string
}

func (s *SmartScraper) saveScrapingResult(result *ScrapingResult) {
	s.repo.CreateDetailedScrapingJob(result)
}

func (s *SmartScraper) buildPageURL(page int) string {
	if strings.Contains(s.config.URL, "news.ycombinator.com") {
		if page == 1 {
			return "https://news.ycombinator.com/"
		}
		return fmt.Sprintf("https://news.ycombinator.com/?p=%d", page)
	}
	
	if page == 1 {
		return s.config.URL
	}
	return fmt.Sprintf("%s?page=%d", s.config.URL, page)
}


func (s *SmartScraper) scrapeFullArchive(result *ScrapingResult) error {
	for page := 1; page <= s.maxPages; page++ {
		url := s.buildPageURL(page)
		log.Printf("Scraping page %d: %s", page, url)
		
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error fetching page %d: %v", page, err)
			result.Errors = append(result.Errors, fmt.Sprintf("Page %d: %v", page, err))
			break
		}
		defer resp.Body.Close()
		
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Printf("Error parsing page %d: %v", page, err)
			result.Errors = append(result.Errors, fmt.Sprintf("Page %d parse: %v", page, err))
			break
		}
		
		posts, err := s.parser.ParseDocument(doc)
		if err != nil {
			log.Printf("Error parsing posts on page %d: %v", page, err)
			result.Errors = append(result.Errors, fmt.Sprintf("Page %d posts: %v", page, err))
			continue
		}
		
		if len(posts) == 0 {
			log.Printf("No posts found on page %d, stopping", page)
			break
		}
		
		saved := s.savePosts(posts, result)
		result.PostsScraped += saved
		result.PagesScraped = page
		
		if s.stopOnDuplicate && saved == 0 {
			log.Printf("No new posts saved on page %d (stop on duplicate enabled), stopping", page)
			break
		}
		
		time.Sleep(2 * time.Second)
	}
	
	return nil
}

func (s *SmartScraper) scrapeUntilExisting(result *ScrapingResult) error {
	duplicateCount := 0
	duplicateThreshold := 5
	consecutiveEmptyPages := 0
	
	for page := 1; page <= s.maxPages; page++ {
		url := s.buildPageURL(page)
		posts, err := s.scrapePage(url, page)
		if err != nil {
			log.Printf("Error scraping page %d: %v", page, err)
			result.Errors = append(result.Errors, fmt.Sprintf("Page %d: %v", page, err))
			break
		}
		
		if len(posts) == 0 {
			consecutiveEmptyPages++
			if consecutiveEmptyPages >= 2 {
				log.Printf("No posts found on %d consecutive pages, stopping", consecutiveEmptyPages)
				break
			}
			continue
		}
		consecutiveEmptyPages = 0
		
		newPosts := 0
		for _, post := range posts {
			exists, err := s.repo.PostExists(post.HnID)
			if err != nil {
				continue
			}
			
			if exists {
				duplicateCount++
				if duplicateCount >= duplicateThreshold {
					log.Printf("Found %d duplicates in a row, stopping", duplicateThreshold)
					return nil
				}
			} else {
				duplicateCount = 0
				if err := s.repo.InsertPost(&post); err == nil {
					newPosts++
					result.NewPosts++
				}
			}
		}
		
		result.PostsScraped += newPosts
		result.PagesScraped = page
		
		if newPosts == 0 {
			log.Printf("No new posts on page %d, stopping", page)
			break
		}
		
		time.Sleep(1 * time.Second)
	}
	
	return nil
}