package charts

import (
	"testing"
	"time"

	"github.com/takl/takl/internal/domain"
)

func TestChartBuilder(t *testing.T) {
	now := time.Now()
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	chart := NewChart("test_chart").
		WithData("key1", "value1").
		WithData("key2", 42).
		WithTimeRange(start, end).
		WithCurrentTime(now).
		Build()

	if chart.Type != "test_chart" {
		t.Errorf("Expected chart type 'test_chart', got %s", chart.Type)
	}

	data, ok := chart.Data.(map[string]any)
	if !ok {
		t.Fatal("Expected chart data to be map[string]any")
	}

	if data["key1"] != "value1" {
		t.Errorf("Expected key1 to be 'value1', got %v", data["key1"])
	}

	if data["key2"] != 42 {
		t.Errorf("Expected key2 to be 42, got %v", data["key2"])
	}

	if data["start_date"] != start {
		t.Errorf("Expected start_date to be %v, got %v", start, data["start_date"])
	}

	if data["end_date"] != end {
		t.Errorf("Expected end_date to be %v, got %v", end, data["end_date"])
	}

	if data["current"] != now {
		t.Errorf("Expected current to be %v, got %v", now, data["current"])
	}
}

func TestCalculateStoryPoints(t *testing.T) {
	issues := []*domain.Issue{
		{
			Status: "open",
			Labels: []string{"points:5", "frontend"},
		},
		{
			Status: "done",
			Labels: []string{"points:3", "backend"},
		},
		{
			Status: "in_progress",
			Labels: []string{"points:8", "api"},
		},
		{
			Status: "done",
			Labels: []string{"points:2"},
		},
		{
			Status: "open",
			Labels: []string{"bug"}, // No story points
		},
	}

	totalPoints, completedPoints := CalculateStoryPoints(issues)

	expectedTotal := 5 + 3 + 8 + 2 // 18 total points
	expectedCompleted := 3 + 2     // 5 completed points

	if totalPoints != expectedTotal {
		t.Errorf("Expected total points %d, got %d", expectedTotal, totalPoints)
	}

	if completedPoints != expectedCompleted {
		t.Errorf("Expected completed points %d, got %d", expectedCompleted, completedPoints)
	}
}

func TestFilterIssuesByStatus(t *testing.T) {
	issues := []*domain.Issue{
		{Status: "open"},
		{Status: "done"},
		{Status: "open"},
		{Status: "in_progress"},
	}

	openIssues := FilterIssuesByStatus(issues, "open")
	if len(openIssues) != 2 {
		t.Errorf("Expected 2 open issues, got %d", len(openIssues))
	}

	doneIssues := FilterIssuesByStatus(issues, "done")
	if len(doneIssues) != 1 {
		t.Errorf("Expected 1 done issue, got %d", len(doneIssues))
	}
}

func TestGroupIssuesByStatus(t *testing.T) {
	issues := []*domain.Issue{
		{Status: "open"},
		{Status: "done"},
		{Status: "open"},
		{Status: "in_progress"},
	}

	groups := GroupIssuesByStatus(issues)

	if len(groups["open"]) != 2 {
		t.Errorf("Expected 2 issues in 'open' group, got %d", len(groups["open"]))
	}

	if len(groups["done"]) != 1 {
		t.Errorf("Expected 1 issue in 'done' group, got %d", len(groups["done"]))
	}

	if len(groups["in_progress"]) != 1 {
		t.Errorf("Expected 1 issue in 'in_progress' group, got %d", len(groups["in_progress"]))
	}
}
