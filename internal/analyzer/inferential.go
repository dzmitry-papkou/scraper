package analyzer

import (
	"database/sql"
	"fmt"
	"math"

	"github.com/dzmitry-papkou/scraper/internal/database"
)

type InferentialAnalyzer struct {
	repo *database.Repository
	db   *sql.DB
}

func NewInferentialAnalyzer(repo *database.Repository) *InferentialAnalyzer {
	return &InferentialAnalyzer{
		repo: repo,
		db:   database.GetDB(),
	}
}

func (a *InferentialAnalyzer) CorrelationAnalysis() map[string]float64 {
	results := make(map[string]float64)

	if corr, err := a.calculateCorrelation("points", "comments_count"); err == nil {
		results["points_vs_comments"] = corr
	}

	if corr, err := a.calculateCorrelation("EXTRACT(HOUR FROM post_time)", "points"); err == nil {
		results["hour_vs_points"] = corr
	}

	if corr, err := a.calculateCorrelation("LENGTH(title)", "points"); err == nil {
		results["title_length_vs_points"] = corr
	}

	return results
}

func (a *InferentialAnalyzer) calculateCorrelation(field1, field2 string) (float64, error) {
	var correlation sql.NullFloat64
	query := fmt.Sprintf(`
		SELECT CORR(%s::numeric, %s::numeric)
		FROM posts
		WHERE points > 0 AND %s IS NOT NULL AND %s IS NOT NULL`, 
		field1, field2, field1, field2)

	err := a.db.QueryRow(query).Scan(&correlation)
	if err != nil || !correlation.Valid {
		return 0, err
	}
	return correlation.Float64, nil
}

type TTestResult struct {
	Group1Name    string
	Group1Mean    float64
	Group1StdDev  float64
	Group1Count   int
	Group2Name    string
	Group2Mean    float64
	Group2StdDev  float64
	Group2Count   int
	TStatistic    float64
	DegreesOfFreedom float64
	PValue        float64
	Significant   bool
	Interpretation string
}

func (a *InferentialAnalyzer) WeekdayVsWeekendTTest() (*TTestResult, error) {
	result := &TTestResult{
		Group1Name: "Weekday",
		Group2Name: "Weekend",
	}

	var weekdayStdDev, weekdayVariance sql.NullFloat64
	err := a.db.QueryRow(`
		SELECT COUNT(*), 
		       COALESCE(AVG(points), 0), 
		       STDDEV(points), 
		       VARIANCE(points)
		FROM posts
		WHERE EXTRACT(DOW FROM post_time) IN (1,2,3,4,5)
		AND points > 0`).Scan(
		&result.Group1Count,
		&result.Group1Mean,
		&weekdayStdDev,
		&weekdayVariance,
	)
	if err != nil {
		return nil, fmt.Errorf("weekday query failed: %w", err)
	}

	if weekdayStdDev.Valid {
		result.Group1StdDev = weekdayStdDev.Float64
	}

	var weekendStdDev, weekendVariance sql.NullFloat64
	err = a.db.QueryRow(`
		SELECT COUNT(*), 
		       COALESCE(AVG(points), 0), 
		       STDDEV(points), 
		       VARIANCE(points)
		FROM posts
		WHERE EXTRACT(DOW FROM post_time) IN (0,6)
		AND points > 0`).Scan(
		&result.Group2Count,
		&result.Group2Mean,
		&weekendStdDev,
		&weekendVariance,
	)
	if err != nil {
		return nil, fmt.Errorf("weekend query failed: %w", err)
	}

	if weekendStdDev.Valid {
		result.Group2StdDev = weekendStdDev.Float64
	}

	if result.Group1Count > 1 && result.Group2Count > 1 && 
	   weekdayVariance.Valid && weekendVariance.Valid {
		
		meanDiff := result.Group1Mean - result.Group2Mean
		se := math.Sqrt((weekdayVariance.Float64/float64(result.Group1Count)) + 
		               (weekendVariance.Float64/float64(result.Group2Count)))
		
		if se > 0 {
			result.TStatistic = meanDiff / se
			
			v1 := weekdayVariance.Float64 / float64(result.Group1Count)
			v2 := weekendVariance.Float64 / float64(result.Group2Count)
			result.DegreesOfFreedom = math.Pow(v1+v2, 2) / 
				(math.Pow(v1, 2)/float64(result.Group1Count-1) + 
				 math.Pow(v2, 2)/float64(result.Group2Count-1))
			
			criticalValue := 2.0
			result.Significant = math.Abs(result.TStatistic) > criticalValue
			
			if result.Significant {
				if meanDiff > 0 {
					result.Interpretation = fmt.Sprintf("%s posts have significantly higher points than %s posts", 
						result.Group1Name, result.Group2Name)
				} else {
					result.Interpretation = fmt.Sprintf("%s posts have significantly higher points than %s posts", 
						result.Group2Name, result.Group1Name)
				}
			} else {
				result.Interpretation = fmt.Sprintf("No significant difference between %s and %s posts", 
					result.Group1Name, result.Group2Name)
			}
		}
	} else {
		result.Interpretation = "Insufficient data for statistical analysis"
	}

	return result, nil
}

func (a *InferentialAnalyzer) MorningVsEveningTTest() (*TTestResult, error) {
	result := &TTestResult{
		Group1Name: "Morning (6AM-12PM)",
		Group2Name: "Evening (6PM-11PM)",
	}

	var morningStdDev, morningVariance sql.NullFloat64
	err := a.db.QueryRow(`
		SELECT COUNT(*), 
		       COALESCE(AVG(points), 0), 
		       STDDEV(points), 
		       VARIANCE(points)
		FROM posts
		WHERE EXTRACT(HOUR FROM post_time) BETWEEN 6 AND 12
		AND points > 0`).Scan(
		&result.Group1Count,
		&result.Group1Mean,
		&morningStdDev,
		&morningVariance,
	)
	if err != nil {
		return nil, fmt.Errorf("morning query failed: %w", err)
	}

	if morningStdDev.Valid {
		result.Group1StdDev = morningStdDev.Float64
	}

	var eveningStdDev, eveningVariance sql.NullFloat64
	err = a.db.QueryRow(`
		SELECT COUNT(*), 
		       COALESCE(AVG(points), 0), 
		       STDDEV(points), 
		       VARIANCE(points)
		FROM posts
		WHERE EXTRACT(HOUR FROM post_time) BETWEEN 18 AND 23
		AND points > 0`).Scan(
		&result.Group2Count,
		&result.Group2Mean,
		&eveningStdDev,
		&eveningVariance,
	)
	if err != nil {
		return nil, fmt.Errorf("evening query failed: %w", err)
	}

	if eveningStdDev.Valid {
		result.Group2StdDev = eveningStdDev.Float64
	}

	// t-test if valid data presented
	if result.Group1Count > 1 && result.Group2Count > 1 && 
	   morningVariance.Valid && eveningVariance.Valid {
		
		meanDiff := result.Group1Mean - result.Group2Mean
		se := math.Sqrt((morningVariance.Float64/float64(result.Group1Count)) + 
		               (eveningVariance.Float64/float64(result.Group2Count)))
		
		if se > 0 {
			result.TStatistic = meanDiff / se
			
			// degrees of freedom
			v1 := morningVariance.Float64 / float64(result.Group1Count)
			v2 := eveningVariance.Float64 / float64(result.Group2Count)
			result.DegreesOfFreedom = math.Pow(v1+v2, 2) / 
				(math.Pow(v1, 2)/float64(result.Group1Count-1) + 
				 math.Pow(v2, 2)/float64(result.Group2Count-1))
			
			// significance
			criticalValue := 2.0
			result.Significant = math.Abs(result.TStatistic) > criticalValue
			
			if result.Significant {
				if meanDiff > 0 {
					result.Interpretation = "Morning posts receive significantly more points than evening posts"
				} else {
					result.Interpretation = "Evening posts receive significantly more points than morning posts"
				}
			} else {
				result.Interpretation = "No significant difference between morning and evening posts"
			}
		}
	} else {
		result.Interpretation = "Insufficient data for statistical analysis"
	}

	return result, nil
}