package scraper

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dzmitry-papkou/scraper/internal/models"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseDocument(doc *goquery.Document) ([]models.Post, error) {
	var posts []models.Post

	doc.Find("tr.athing").Each(func(i int, s *goquery.Selection) {
		post, err := p.parsePost(s)
		if err != nil {
			log.Printf("Error parsing post #%d: %v", i+1, err)
		} else if post.HnID > 0 {
			posts = append(posts, post)
		}
	})

	log.Printf("Parsed %d posts", len(posts))
	return posts, nil
}

func (p *Parser) parsePost(s *goquery.Selection) (models.Post, error) {
	var post models.Post

	hnIDStr, exists := s.Attr("id")
	if !exists {
		return post, fmt.Errorf("no ID found")
	}

	hnID, err := strconv.Atoi(hnIDStr)
	if err != nil {
		return post, fmt.Errorf("invalid ID: %s", hnIDStr)
	}
	post.HnID = hnID

	// title and url from .titleline
	titleline := s.Find(".titleline")
	titleLink := titleline.Find("a").First()
	post.Title = strings.TrimSpace(titleLink.Text())
	post.URL, _ = titleLink.Attr("href")

	if post.URL != "" && !strings.HasPrefix(post.URL, "http") {
		post.URL = "https://news.ycombinator.com/" + post.URL
	}

	// metadata from the next <tr> (subtext row)
	metaRow := s.Next()
	if metaRow.Length() == 0 {
		return post, fmt.Errorf("no metadata row found")
	}

	subtext := metaRow.Find(".subtext")

	// points
	scoreText := subtext.Find(".score").Text()
	if scoreText != "" {
		fmt.Sscanf(scoreText, "%d points", &post.Points)
	}

	// author
	post.Author = strings.TrimSpace(subtext.Find(".hnuser").Text())
	if post.Author == "" {
		post.Author = "unknown"
	}

	// post time
	ageElement := subtext.Find(".age")
	if ageElement.Length() > 0 {
		timeStr, hasTitle := ageElement.Attr("title")
		
		if hasTitle && timeStr != "" {
			parts := strings.Fields(timeStr)
			if len(parts) > 0 {
				timeToParse := parts[0]
				
				// time ISO format
				if t, err := time.Parse("2006-01-02T15:04:05", timeToParse); err == nil {
					post.PostTime = t
				} else {
					// fallback to relative time
					ageText := strings.TrimSpace(ageElement.Text())
					post.PostTime = p.parseRelativeTime(ageText)
				}
			}
		} else {
			// relative time from text 
			ageText := strings.TrimSpace(ageElement.Text())
			post.PostTime = p.parseRelativeTime(ageText)
		}
	}

	// current time if parsing failed
	if post.PostTime.IsZero() || post.PostTime.Year() < 2000 {
		post.PostTime = time.Now()
	}

	// comments count
	post.CommentsCount = p.parseComments(subtext)

	post.ScrapedAt = time.Now()

	return post, nil
}

func (p *Parser) parseRelativeTime(ageText string) time.Time {
	now := time.Now()
	ageText = strings.TrimSpace(strings.ToLower(ageText))
	
	ageText = strings.TrimSuffix(ageText, " ago")

	if ageText == "just now" {
		return now
	}
	if ageText == "yesterday" {
		return now.AddDate(0, 0, -1)
	}

	// patterns "2 hours", "1 day", etc.
	var value int
	var unit string

	parts := strings.Fields(ageText)
	if len(parts) >= 2 {
		if n, err := strconv.Atoi(parts[0]); err == nil {
			value = n
			unit = parts[1]
		} else if parts[0] == "a" || parts[0] == "an" {
			value = 1
			unit = parts[1]
		}
	}

	switch {
	case strings.Contains(unit, "second"):
		return now.Add(-time.Duration(value) * time.Second)
	case strings.Contains(unit, "minute"):
		return now.Add(-time.Duration(value) * time.Minute)
	case strings.Contains(unit, "hour"):
		return now.Add(-time.Duration(value) * time.Hour)
	case strings.Contains(unit, "day"):
		return now.AddDate(0, 0, -value)
	case strings.Contains(unit, "week"):
		return now.AddDate(0, 0, -value*7)
	case strings.Contains(unit, "month"):
		return now.AddDate(0, -value, 0)
	case strings.Contains(unit, "year"):
		return now.AddDate(-value, 0, 0)
	default:
		return now
	}
}

func (p *Parser) parseComments(subtext *goquery.Selection) int {
	comments := 0

	subtext.Find("a").Each(func(i int, link *goquery.Selection) {
		text := strings.TrimSpace(link.Text())

		if text == "discuss" {
			return
		}

		if strings.Contains(text, "comment") {
			text = strings.ReplaceAll(text, "\u00a0", " ")
			text = strings.ReplaceAll(text, "&nbsp;", " ")
			
			var num int
			if n, err := fmt.Sscanf(text, "%d comment", &num); err == nil && n == 1 {
				comments = num
			} else if n, err := fmt.Sscanf(text, "%d comments", &num); err == nil && n == 1 {
				comments = num
			}
		}
	})

	return comments
}