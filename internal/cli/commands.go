package cli

import (
	"fmt"
	"os"

	"strconv"
	"strings"
	"time"

	"github.com/dzmitry-papkou/scraper/internal/analyzer"
	"github.com/dzmitry-papkou/scraper/internal/config"
	"github.com/dzmitry-papkou/scraper/internal/database"
	"github.com/dzmitry-papkou/scraper/internal/scraper"
	"github.com/fatih/color"
)

type Commander struct {
	repo                *database.Repository
	currentScraper      *scraper.Scraper
	currentScraperName  string
	descriptiveAnalyzer *analyzer.DescriptiveAnalyzer
	inferentialAnalyzer *analyzer.InferentialAnalyzer
	scheduler           *scraper.MultiScheduler
	config              *config.Config
	
	// color
	green  func(a ...interface{}) string
	red    func(a ...interface{}) string
	yellow func(a ...interface{}) string
	cyan   func(a ...interface{}) string
	blue   func(a ...interface{}) string
}

func NewCommanderWithConfig(repo *database.Repository, scraperName string, cfg *config.Config) (*Commander, error) {
	scraperInstance, err := scraper.NewGenericScraper(repo, scraperName)
	if err != nil {
		scraperInstance = scraper.New(repo)
		scraperName = "hackernews"
	}
	
	return &Commander{
		repo:               repo,
		currentScraper:     scraperInstance,
		currentScraperName: scraperName,
		descriptiveAnalyzer: analyzer.NewDescriptiveAnalyzer(repo),
		inferentialAnalyzer: analyzer.NewInferentialAnalyzer(repo),
		scheduler:          scraper.NewMultiScheduler(repo),
		config:             cfg,
		green:              color.New(color.FgGreen).SprintFunc(),
		red:                color.New(color.FgRed).SprintFunc(),
		yellow:             color.New(color.FgYellow).SprintFunc(),
		cyan:               color.New(color.FgCyan).SprintFunc(),
		blue:               color.New(color.FgBlue).SprintFunc(),
	}, nil
}

func NewCommander(repo *database.Repository) *Commander {
	config.LoadDefault()
	cfg := config.Get()
	commander, _ := NewCommanderWithConfig(repo, cfg.App.DefaultScraper, cfg)
	return commander
}

func (c *Commander) ExecuteCommand(command string, args []string) {
	switch command {
	case "help", "h":
		c.showHelp()
	case "scrape", "s":
		c.scrapeOnce()
	case "scrape-all", "sall":
    	c.scrapeAll()
	case "scrape-new", "snew":
  		 c.scrapeNew()
	case "scrape-history", "history":
    	c.showScrapingHistory()
	case "start":
		c.startAutoScraping()
	case "stop":
		c.stopAutoScraping()
	case "status":
		c.showStatus()
	case "stats":
		c.showStatistics()
	case "show":
		limit := 10
		if len(args) > 0 {
			if n, err := strconv.Atoi(args[0]); err == nil {
				limit = n
			}
		}
		c.showRecentPosts(limit)
	case "analyze", "analyse", "a":
		c.runAnalysis()
	case "export", "e":
		c.exportData()
	case "scrapers":
		c.listScrapers()
	case "clear":
		c.clearScreen()
	case "quit", "exit", "q":
		c.quit()
	default:
		fmt.Printf("%s Unknown command: %s\n", c.red("✗"), command)
		fmt.Println("Type 'help' for available commands")
	}
}

func (c *Commander) showHelp() {
   fmt.Println(c.blue("\nAvailable Commands:"))
    fmt.Println("\n" + c.cyan("Basic:"))
    fmt.Println("  help         - Show this help message")
    fmt.Println("  status       - Show current status")
    fmt.Println("  quit         - Exit program")
    
    fmt.Println("\n" + c.cyan("Scraping:"))
    fmt.Println("  scrape       - Quick scrape (latest page only)")
    fmt.Println("  scrape-new   - Scrape only new posts since last run")
    fmt.Println("  scrape-all   - Full archive scrape (multiple pages)")
    fmt.Println("  start/stop   - Start/stop automatic scraping")
    
    fmt.Println("\n" + c.cyan("Analysis:"))
    fmt.Println("  stats        - Display statistics")
    fmt.Println("  analyze      - Run statistical analysis")
    fmt.Println("  coverage     - Show database coverage")
    
    fmt.Println("\n" + c.cyan("Data:"))
    fmt.Println("  show [n]     - Show n recent posts")
    fmt.Println("  export       - Export data to CSV")
    fmt.Println("  history      - Show scraping history")
    
    fmt.Println("\n" + c.cyan("Configuration:"))
    fmt.Println("  scrapers     - List available scrapers")
    fmt.Println("  clear        - Clear screen")
}



func (c *Commander) scrapeAll() {
    fmt.Println(c.cyan("Starting FULL archive scrape..."))
    fmt.Println(c.yellow("This may take a while and will scrape multiple pages"))
    
    scraperConfig := c.currentScraper.GetConfig()
    
    smartScraper := scraper.NewSmartScraper(
        c.repo, 
        scraperConfig,
        scraper.ModeFullArchive,
        50,
    )
    
    result, err := smartScraper.ScrapeWithStrategy()
    
    if err != nil {
        fmt.Printf("%s Error: %v\n", c.red("✗"), err)
        return
    }
    
    c.printScrapingResult(result)
}

func (c *Commander) scrapeNew() {
    fmt.Println(c.cyan("Scraping only NEW posts since last scrape..."))
    
    lastID, _ := c.repo.GetLatestHNPostID()
    fmt.Printf("Last known post ID: %d\n", lastID)
    
    scraperConfig := c.currentScraper.GetConfig()
    
    smartScraper := scraper.NewSmartScraper(
        c.repo,
        scraperConfig,
        scraper.ModeSinceLast,
        10,
    )
    
    result, err := smartScraper.ScrapeWithStrategy()
    
    if err != nil {
        fmt.Printf("%s Error: %v\n", c.red("✗"), err)
        return
    }
    
    c.printScrapingResult(result)
}

func (c *Commander) printScrapingResult(result *scraper.ScrapingResult) {
    fmt.Println(c.green("\n✓ Scraping Complete!"))
    fmt.Println(strings.Repeat("─", 40))
    fmt.Printf("Mode:           %s\n", result.Mode)
    fmt.Printf("Duration:       %.2f seconds\n", result.Duration.Seconds())
    fmt.Printf("Pages scraped:  %d\n", result.PagesScraped)
    fmt.Printf("Posts scraped:  %d\n", result.PostsScraped)
    fmt.Printf("New posts:      %s\n", c.green(fmt.Sprintf("%d", result.NewPosts)))
    fmt.Printf("Updated posts:  %s\n", c.yellow(fmt.Sprintf("%d", result.UpdatedPosts)))
    
    if result.DeletedPosts > 0 {
        fmt.Printf("Deleted posts:  %s\n", c.red(fmt.Sprintf("%d", result.DeletedPosts)))
    }
    
    if result.HighestIDSeen > result.LastKnownID {
        fmt.Printf("ID range:       %d → %d\n", result.LastKnownID, result.HighestIDSeen)
    }
}

func (c *Commander) showScrapingHistory() {
    fmt.Println(c.blue("\nScraping History"))
    fmt.Println(strings.Repeat("─", 70))
    
    history, err := c.repo.GetScrapingHistory(10)
    if err != nil {
        fmt.Printf("%s Error: %v\n", c.red("✗"), err)
        return
    }
    
    for _, job := range history {
        startTime := job["started_at"].(time.Time)
        status := job["status"].(string)
        posts := job["posts_scraped"].(int)
        
        statusColor := c.green
        switch status {
			case "failed":
            	statusColor = c.red
        	case "running":
            	statusColor = c.yellow
        }
        
        fmt.Printf("%s | %s | %d posts",
            startTime.Format("Jan 02 15:04"),
            statusColor(status),
            posts)
        
        if details, ok := job["details"].(map[string]interface{}); ok {
            if newPosts, ok := details["new_posts"].(float64); ok {
                fmt.Printf(" | %s new", c.green(fmt.Sprintf("%.0f", newPosts)))
            }
            if pages, ok := details["pages_scraped"].(float64); ok {
                fmt.Printf(" | %.0f pages", pages)
            }
        }
        fmt.Println()
    }
}

func (c *Commander) scrapeOnce() {
	fmt.Printf(c.cyan("Scraping %s...\n"), c.currentScraperName)
	count, err := c.currentScraper.ScrapeOnce()
	if err != nil {
		fmt.Printf("%s Error: %v\n", c.red("✗"), err)
		return
	}
	fmt.Printf("%s Scraped %d posts from %s\n", c.green("✓"), count, c.currentScraperName)
}

func (c *Commander) startAutoScraping() {
	scraperConfig := c.currentScraper.GetConfig()
	
	if c.scheduler.IsActive(c.currentScraperName) {
		fmt.Printf("%s Auto-scraping for %s is already active\n", 
			c.yellow("⚠"), c.currentScraperName)
		return
	}
	
	c.scheduler.StartScraper(c.currentScraperName, scraperConfig.Interval)
	fmt.Printf("%s Started auto-scraping %s (every %s)\n", 
		c.green("✓"), c.currentScraperName, scraperConfig.Interval)
}

func (c *Commander) stopAutoScraping() {
	if !c.scheduler.IsActive(c.currentScraperName) {
		fmt.Printf("%s Auto-scraping for %s is not active\n", 
			c.yellow("⚠"), c.currentScraperName)
		return
	}
	
	c.scheduler.StopScraper(c.currentScraperName)
	fmt.Printf("%s Stopped auto-scraping for %s\n", c.green("✓"), c.currentScraperName)
}

func (c *Commander) showStatus() {
	fmt.Println(c.blue("\n System Status"))
	fmt.Println(strings.Repeat("─", 40))
	
	fmt.Printf("Current scraper: %s\n", c.cyan(c.currentScraperName))
	
	activeScrapers := c.scheduler.GetActiveScrapers()
	if len(activeScrapers) > 0 {
		fmt.Printf("Active scrapers: %s\n", c.green(strings.Join(activeScrapers, ", ")))
	} else {
		fmt.Printf("Active scrapers: %s\n", c.red("none"))
	}
	
	if err := database.GetDB().Ping(); err == nil {
		fmt.Printf("Database:        %s\n", c.green("CONNECTED ●"))
	} else {
		fmt.Printf("Database:        %s\n", c.red("DISCONNECTED ○"))
	}
	
	if job, err := c.repo.GetLastScrapingJob(); err == nil && job != nil {
		fmt.Printf("Last scrape:     %s (%d posts)\n",
			job.CompletedAt.Format("15:04:05"), job.PostsScraped)
	}
	
	if count, err := c.repo.GetPostCount(); err == nil {
		fmt.Printf("Total posts:     %d\n", count)
	}
	
	var todayCount int
	db := database.GetDB()
	db.QueryRow(`
		SELECT COUNT(*) FROM posts 
		WHERE DATE(scraped_at) = CURRENT_DATE
	`).Scan(&todayCount)
	fmt.Printf("Today's posts:   %d\n", todayCount)
}

func (c *Commander) showStatistics() {
	fmt.Println(c.blue("\nDatabase Statistics"))
	fmt.Println(strings.Repeat("─", 50))
	
	if stats, err := c.descriptiveAnalyzer.BasicStatistics(); err == nil {
		fmt.Printf("Total posts:      %d\n", stats["total_posts"])
		fmt.Printf("Unique authors:   %d\n", stats["unique_authors"])
		fmt.Printf("Average points:   %.1f\n", stats["avg_points"])
		fmt.Printf("Average comments: %.1f\n", stats["avg_comments"])
		fmt.Printf("Max points:       %d\n", stats["max_points"])
		fmt.Printf("Max comments:     %d\n", stats["max_comments"])
	}
	
	fmt.Println(c.blue("\nTop 5 Posts by Points:"))
	if posts, err := c.descriptiveAnalyzer.GetTopPosts(5); err == nil {
		for i, post := range posts {
			title := post.Title
			if len(title) > 50 {
				title = title[:50] + "..."
			}
			fmt.Printf("%d. %s\n   %s (%d points)\n", 
				i+1, title, post.Author, post.Points)
		}
	}
	
	fmt.Println(c.blue("\nPeak Posting Hours:"))
	if patterns, err := c.descriptiveAnalyzer.GetPostingPatterns(); err == nil {
		shown := 0
		for _, p := range patterns {
			if shown >= 5 {
				break
			}
			fmt.Printf("  %02d:00 - %d posts (avg %.1f points)\n",
				p.Hour, p.PostCount, p.AvgPoints)
			shown++
		}
	}
}

func (c *Commander) showRecentPosts(limit int) {
	fmt.Printf(c.blue("\nRecent %d Posts:\n"), limit)
	fmt.Println(strings.Repeat("─", 70))
	
	posts, err := c.repo.GetRecentPosts(limit)
	if err != nil {
		fmt.Printf("%s Error: %v\n", c.red("✗"), err)
		return
	}
	
	for _, post := range posts {
		title := post.Title
		if len(title) > 60 {
			title = title[:60] + "..."
		}
		
		fmt.Printf("\n%s %s\n", c.green("+"), title)
		fmt.Printf("  by %s | %d points | %d comments | %s\n",
			post.Author, post.Points, post.CommentsCount,
			post.ScrapedAt.Format("15:04"))
	}
}

func (c *Commander) runAnalysis() {
	fmt.Println(c.blue("\nStatistical Analysis"))
	fmt.Println(strings.Repeat("─", 50))
	
	fmt.Println(c.cyan("\nCORRELATION ANALYSIS"))
	correlations := c.inferentialAnalyzer.CorrelationAnalysis()
	
	for name, value := range correlations {
		displayName := strings.ReplaceAll(name, "_", " ")
		fmt.Printf("%s: %.3f\n", displayName, value)
		c.interpretCorrelation(value)
	}
	
	fmt.Println(c.cyan("\nT-TEST ANALYSIS"))
	
	if result, err := c.inferentialAnalyzer.WeekdayVsWeekendTTest(); err == nil {
		fmt.Println("\nWeekday vs Weekend performance:")
		c.printTTestResult(result)
	}
	
	if result, err := c.inferentialAnalyzer.MorningVsEveningTTest(); err == nil {
		fmt.Println("\nMorning vs Evening performance:")
		c.printTTestResult(result)
	}
	
	fmt.Println(c.cyan("\n7-DAY TREND"))
	if trends, err := c.descriptiveAnalyzer.GetDailyTrends(7); err == nil {
		for _, trend := range trends {
			fmt.Printf("  %s: %d posts, %.1f avg points, %.1f avg comments\n",
				trend.Date, trend.PostCount, trend.AvgPoints, trend.AvgComments)
		}
	}
}

func (c *Commander) interpretCorrelation(value float64) {
	strength := ""
	absVal := value
	if absVal < 0 {
		absVal = -absVal
	}
	
	switch {
	case absVal < 0.1:
		strength = "no"
	case absVal < 0.3:
		strength = "weak"
	case absVal < 0.5:
		strength = "moderate"
	case absVal < 0.7:
		strength = "strong"
	default:
		strength = "very strong"
	}
	
	direction := "positive"
	if value < 0 {
		direction = "negative"
	}
	
	fmt.Printf("   → %s %s correlation\n", strength, direction)
}

func (c *Commander) printTTestResult(result *analyzer.TTestResult) {
	fmt.Printf("  %s: n=%d, mean=%.2f, std=%.2f\n",
		result.Group1Name, result.Group1Count, result.Group1Mean, result.Group1StdDev)
	fmt.Printf("  %s: n=%d, mean=%.2f, std=%.2f\n",
		result.Group2Name, result.Group2Count, result.Group2Mean, result.Group2StdDev)
	fmt.Printf("  T-test: %.3f\n", result.TStatistic)
	fmt.Printf("  Degrees of freedom: %.1f\n", result.DegreesOfFreedom)
	
	if result.Significant {
		fmt.Printf("  Result: %s\n", c.green(result.Interpretation))
	} else {
		fmt.Printf("  Result: %s\n", result.Interpretation)
	}
}

func (c *Commander) exportData() {
	exportPath := c.config.App.ExportPath
	if exportPath == "" {
		exportPath = "./exports"
	}
	
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		fmt.Printf("%s Failed to create export directory: %v\n", c.red("✗"), err)
		return
	}
	
	exporter := NewExporter(c.repo)
	filename, err := exporter.ExportToCSV()
	if err != nil {
		fmt.Printf("%s Error: %v\n", c.red("✗"), err)
		return
	}
	
	newPath := fmt.Sprintf("%s/%s", exportPath, filename)
	if err := os.Rename(filename, newPath); err == nil {
		filename = newPath
	}
	
	if info, err := os.Stat(filename); err == nil {
		size := info.Size()
		sizeStr := fmt.Sprintf("%d bytes", size)
		if size > 1024*1024 {
			sizeStr = fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
		} else if size > 1024 {
			sizeStr = fmt.Sprintf("%.2f KB", float64(size)/1024)
		}
		fmt.Printf("%s Exported data to %s (%s)\n", c.green("✓"), filename, sizeStr)
	} else {
		fmt.Printf("%s Exported data to %s\n", c.green("✓"), filename)
	}
}

func (c *Commander) listScrapers() {
	fmt.Println(c.blue("\nAvailable Scrapers:"))
	fmt.Println(strings.Repeat("─", 50))
	
	for _, scraperConfig := range c.config.Scrapers {
		status := c.red("disabled")
		if scraperConfig.Enabled {
			status = c.green("enabled")
		}
		
		current := ""
		if scraperConfig.Name == c.currentScraperName {
			current = c.cyan(" [CURRENT]")
		}
		
		fmt.Printf("• %s [%s]%s\n", scraperConfig.Name, status, current)
		fmt.Printf("  URL: %s\n", scraperConfig.URL)
		fmt.Printf("  Interval: %s\n", scraperConfig.Interval)
		
		if c.scheduler.IsActive(scraperConfig.Name) {
			fmt.Printf("  Status: %s\n", c.green("RUNNING"))
		}
		fmt.Println()
	}
}

func (c *Commander) clearScreen() {
	fmt.Print("\033[H\033[2J")

	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan("╔══════════════════════════════════════════╗"))
	fmt.Println(cyan("║     Hacker News Scraper & Analyzer       ║"))
	fmt.Println(cyan("║         Fundamentals of AI 101           ║"))
	fmt.Println(cyan("╚══════════════════════════════════════════╝"))
	fmt.Println()
	fmt.Printf("Active scraper: %s\n",  c.currentScraperName)
	fmt.Println("Type 'help' for available commands")
}

func (c *Commander) quit() {
	if activeScrapers := c.scheduler.GetActiveScrapers(); len(activeScrapers) > 0 {
		fmt.Println("Stopping active scrapers...")
		for _, name := range activeScrapers {
			c.scheduler.StopScraper(name)
			fmt.Printf("  Stopped %s\n", name)
		}
	}
	
	fmt.Printf("%s Goodbye!\n", c.green("✓"))
	os.Exit(0)
}