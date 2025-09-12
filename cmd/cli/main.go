package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/dzmitry-papkou/scraper/internal/cli"
	"github.com/dzmitry-papkou/scraper/internal/config"
	"github.com/dzmitry-papkou/scraper/internal/database"
)

func main() {
	var (
		configFile  = flag.String("config", "configs/config.yaml", "Configuration file path")
		scrapeFlag  = flag.Bool("scrape", false, "Perform single scrape and exit")
		analyzeFlag = flag.Bool("analyze", false, "Run analysis and exit")
		exportFlag  = flag.Bool("export", false, "Export data to CSV and exit")
		scraperName = flag.String("scraper", "", "Specific scraper to use (overrides default)")
		listFlag    = flag.Bool("list", false, "List available scrapers")
	)
	flag.Parse()

	if err := loadConfig(*configFile); err != nil {
		log.Printf("Warning: Could not load config file %s: %v", *configFile, err)
		log.Println("Using default configuration")
		config.LoadDefault()
	}

	cfg := config.Get()

	if *listFlag {
		listScrapers()
		return
	}

	if err := initDatabase(cfg); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	scraperToUse := cfg.App.DefaultScraper
	if *scraperName != "" {
		scraperToUse = *scraperName
	}

	repo := database.NewRepository()
	commander, err := cli.NewCommanderWithConfig(repo, scraperToUse, cfg)
	if err != nil {
		log.Fatal("Failed to initialize commander:", err)
	}

	if *scrapeFlag {
		commander.ExecuteCommand("scrape", nil)
		return
	}
	if *analyzeFlag {
		commander.ExecuteCommand("analyze", nil)
		return
	}
	if *exportFlag {
		commander.ExecuteCommand("export", nil)
		return
	}

	printWelcome(cfg)
	startInteractiveMode(commander, cfg)
}

func loadConfig(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		execPath, _ := os.Executable()
		execDir := filepath.Dir(execPath)
		altPath := filepath.Join(execDir, path)
		
		if _, err := os.Stat(altPath); err == nil {
			path = altPath
		} else {
			return fmt.Errorf("config file not found: %s", path)
		}
	}

	return config.Load(path)
}

func initDatabase(cfg *config.Config) error {
	dbConfig := database.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "scraperuser"),
		Password: getEnv("DB_PASSWORD", "supersecret"),
		Database: getEnv("DB_NAME", "scraperdb"),
		SSLMode:  "disable",
	}

	if cfg.Database.URL != "" {
		return database.InitializeWithURL(cfg.Database.URL, 
			cfg.Database.MaxConnections, 
			cfg.Database.MaxIdle,
			cfg.Database.ConnectionLifetime)
	}

	return database.Initialize(dbConfig)
}

func listScrapers() {
	cfg := config.Get()
	
	fmt.Println("\nAvailable Scrapers:")
	fmt.Println(strings.Repeat("─", 50))
	
	for _, scraper := range cfg.Scrapers {
		status := "disabled"
		statusColor := color.New(color.FgRed).SprintFunc()
		if scraper.Enabled {
			status = "enabled"
			statusColor = color.New(color.FgGreen).SprintFunc()
		}
		
		fmt.Printf("• %s [%s]\n", scraper.Name, statusColor(status))
		fmt.Printf("  URL: %s\n", scraper.URL)
		fmt.Printf("  Interval: %s\n", scraper.Interval)
		fmt.Println()
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func printWelcome(cfg *config.Config) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan("╔══════════════════════════════════════════╗"))
	fmt.Println(cyan("║     Hacker News Scraper & Analyzer       ║"))
	fmt.Println(cyan("║         Fundamentals of AI 101           ║"))
	fmt.Println(cyan("╚══════════════════════════════════════════╝"))
	fmt.Println()
	fmt.Printf("Active scraper: %s\n", cfg.App.DefaultScraper)
	fmt.Println("Type 'help' for available commands")
}

func startInteractiveMode(commander *cli.Commander, cfg *config.Config) {
	scanner := bufio.NewScanner(os.Stdin)
	prompt := cfg.App.CLI.Prompt
	if prompt == "" {
		prompt = "➜"
	}
	
	yellow := color.New(color.FgYellow).SprintFunc()

	for {
		fmt.Print(yellow("\n" + prompt + " "))
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := strings.ToLower(parts[0])
		args := parts[1:]

		commander.ExecuteCommand(command, args)
	}
}