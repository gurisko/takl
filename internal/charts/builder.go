package charts

import (
	"time"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/paradigm"
)

// ChartBuilder provides utilities for creating charts with common patterns
type ChartBuilder struct {
	chartType string
	data      map[string]any
}

// NewChart creates a new chart builder
func NewChart(chartType string) *ChartBuilder {
	return &ChartBuilder{
		chartType: chartType,
		data:      make(map[string]any),
	}
}

// WithData adds data to the chart
func (b *ChartBuilder) WithData(key string, value any) *ChartBuilder {
	b.data[key] = value
	return b
}

// WithTimeRange adds time range data to the chart
func (b *ChartBuilder) WithTimeRange(start, end time.Time) *ChartBuilder {
	b.data["start_date"] = start
	b.data["end_date"] = end
	return b
}

// WithCurrentTime adds current timestamp to the chart
func (b *ChartBuilder) WithCurrentTime(now time.Time) *ChartBuilder {
	b.data["current"] = now
	return b
}

// Build creates the final chart
func (b *ChartBuilder) Build() paradigm.Chart {
	return paradigm.Chart{
		Type: b.chartType,
		Data: b.data,
	}
}

// Common chart utilities

// CalculateStoryPoints calculates total story points from issues
func CalculateStoryPoints(issues []*domain.Issue) (int, int) {
	totalPoints := 0
	completedPoints := 0

	for _, issue := range issues {
		points := extractStoryPoints(issue)
		totalPoints += points

		if issue.Status == "done" {
			completedPoints += points
		}
	}

	return totalPoints, completedPoints
}

// extractStoryPoints extracts story points from issue labels
func extractStoryPoints(issue *domain.Issue) int {
	for _, label := range issue.Labels {
		if len(label) > 7 && label[:7] == "points:" {
			// Simple parsing - in production you'd want more robust parsing
			pointsStr := label[7:]
			switch pointsStr {
			case "1":
				return 1
			case "2":
				return 2
			case "3":
				return 3
			case "5":
				return 5
			case "8":
				return 8
			case "13":
				return 13
			default:
				return 0
			}
		}
	}
	return 0
}

// FilterIssuesByStatus filters issues by status
func FilterIssuesByStatus(issues []*domain.Issue, status string) []*domain.Issue {
	filtered := make([]*domain.Issue, 0)
	for _, issue := range issues {
		if issue.Status == status {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// FilterIssuesByTimeRange filters issues by time range
func FilterIssuesByTimeRange(issues []*domain.Issue, start, end time.Time) []*domain.Issue {
	filtered := make([]*domain.Issue, 0)
	for _, issue := range issues {
		if (start.IsZero() || issue.Updated.After(start)) &&
			(end.IsZero() || issue.Updated.Before(end)) {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// GroupIssuesByStatus groups issues by their status
func GroupIssuesByStatus(issues []*domain.Issue) map[string][]*domain.Issue {
	groups := make(map[string][]*domain.Issue)
	for _, issue := range issues {
		groups[issue.Status] = append(groups[issue.Status], issue)
	}
	return groups
}
