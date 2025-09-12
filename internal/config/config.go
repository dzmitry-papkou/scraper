package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig   `yaml:"database"`
	Scrapers []ScraperConfig  `yaml:"scrapers"`
	App      AppConfig        `yaml:"app"`
}

type DatabaseConfig struct {
	URL                string        `yaml:"url"`
	MaxConnections     int           `yaml:"max_connections"`
	MaxIdle            int           `yaml:"max_idle"`
	ConnectionLifetime time.Duration `yaml:"connection_lifetime"`
}

type ScraperConfig struct {
	Name      string            `yaml:"name"`
	URL       string            `yaml:"url"`
	Interval  time.Duration     `yaml:"interval"`
	Enabled   bool              `yaml:"enabled"`
	Selectors ScraperSelectors  `yaml:"selectors"`
}

type ScraperSelectors struct {
	Item        string `yaml:"item"`
	Title       string `yaml:"title"`
	URL         string `yaml:"url"`
	Points      string `yaml:"points"`
	Comments    string `yaml:"comments"`
	Author      string `yaml:"author"`
	MetadataRow string `yaml:"metadata_row,omitempty"`
	Time        string `yaml:"time,omitempty"`
	Date        string `yaml:"date,omitempty"`
}

type AppConfig struct {
	DefaultScraper string           `yaml:"default_scraper"`
	LogLevel       string           `yaml:"log_level"`
	ExportPath     string           `yaml:"export_path"`
	CLI            CLIConfig        `yaml:"cli"`
	Analysis       AnalysisConfig   `yaml:"analysis"`
}

type CLIConfig struct {
	Prompt string            `yaml:"prompt"`
	Colors map[string]string `yaml:"colors"`
}

type AnalysisConfig struct {
	MinPostsForAuthorStats int     `yaml:"min_posts_for_author_stats"`
	TopPostsLimit          int     `yaml:"top_posts_limit"`
	CorrelationThreshold   float64 `yaml:"correlation_threshold"`
	SignificanceLevel      float64 `yaml:"significance_level"`
}

var cfg *Config

func Load(path string) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	cfg = &Config{}
	if err := yaml.Unmarshal(file, cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	setDefaults()

	return nil
}

func Get() *Config {
	if cfg == nil {
		LoadDefault()
	}
	return cfg
}

func GetScraper(name string) (*ScraperConfig, error) {
	for _, scraper := range cfg.Scrapers {
		if scraper.Name == name {
			return &scraper, nil
		}
	}
	return nil, fmt.Errorf("scraper '%s' not found", name)
}

func GetEnabledScrapers() []ScraperConfig {
	var enabled []ScraperConfig
	for _, scraper := range cfg.Scrapers {
		if scraper.Enabled {
			enabled = append(enabled, scraper)
		}
	}
	return enabled
}

func LoadDefault() {
	cfg = &Config{
		Database: DatabaseConfig{
			URL:                "postgres://scraperuser:supersecret@localhost:5432/scraperdb?sslmode=disable",
			MaxConnections:     25,
			MaxIdle:            5,
			ConnectionLifetime: 5 * time.Minute,
		},
		Scrapers: []ScraperConfig{
			{
				Name:     "hackernews",
				URL:      "https://news.ycombinator.com/newest",
				Interval: 5 * time.Minute,
				Enabled:  true,
				Selectors: ScraperSelectors{
					Item:        "tr.athing",
					Title:       ".titleline a",
					URL:         ".titleline a",
					Points:      ".score",
					Comments:    "a:contains('comment')",
					Author:      ".hnuser",
					MetadataRow: "next",
					Time:        ".age",
				},
			},
		},
		App: AppConfig{
			DefaultScraper: "hackernews",
			LogLevel:       "info",
			ExportPath:     "./exports",
			CLI: CLIConfig{
				Prompt: "➜",
				Colors: map[string]string{
					"success": "green",
					"error":   "red",
					"warning": "yellow",
					"info":    "cyan",
				},
			},
			Analysis: AnalysisConfig{
				MinPostsForAuthorStats: 3,
				TopPostsLimit:          5,
				CorrelationThreshold:   0.3,
				SignificanceLevel:      0.05,
			},
		},
	}
}

func setDefaults() {
	if cfg.Database.MaxConnections == 0 {
		cfg.Database.MaxConnections = 25
	}
	if cfg.Database.MaxIdle == 0 {
		cfg.Database.MaxIdle = 5
	}
	if cfg.Database.ConnectionLifetime == 0 {
		cfg.Database.ConnectionLifetime = 5 * time.Minute
	}
	if cfg.App.ExportPath == "" {
		cfg.App.ExportPath = "./exports"
	}
	if cfg.App.Analysis.TopPostsLimit == 0 {
		cfg.App.Analysis.TopPostsLimit = 5
	}
	if cfg.App.Analysis.MinPostsForAuthorStats == 0 {
		cfg.App.Analysis.MinPostsForAuthorStats = 3
	}
	if cfg.App.Analysis.SignificanceLevel == 0 {
		cfg.App.Analysis.SignificanceLevel = 0.05
	}
}