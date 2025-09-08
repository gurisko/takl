package dto

import (
	"time"

	"github.com/takl/takl/internal/domain"
)

// CreateIssueRequest represents the HTTP request for creating an issue
type CreateIssueRequest struct {
	Type     string   `json:"type" binding:"required"`
	Title    string   `json:"title" binding:"required"`
	Content  string   `json:"content"`
	Priority string   `json:"priority"`
	Assignee string   `json:"assignee"`
	Labels   []string `json:"labels"`
}

// ToDomain converts HTTP DTO to domain request
func (r CreateIssueRequest) ToDomain() domain.CreateIssueRequest {
	return domain.CreateIssueRequest{
		Type:        r.Type,
		Title:       r.Title,
		Description: r.Content,
		Priority:    r.Priority,
		Assignee:    r.Assignee,
		Labels:      r.Labels,
	}
}

// UpdateIssueRequest represents the HTTP request for updating an issue
type UpdateIssueRequest struct {
	Title    *string  `json:"title"`
	Content  *string  `json:"content"`
	Status   *string  `json:"status"`
	Priority *string  `json:"priority"`
	Assignee *string  `json:"assignee"`
	Labels   []string `json:"labels"`
}

// ToDomain converts HTTP DTO to domain request
func (r UpdateIssueRequest) ToDomain() domain.UpdateIssueRequest {
	return domain.UpdateIssueRequest{
		Title:    r.Title,
		Content:  r.Content,
		Status:   r.Status,
		Priority: r.Priority,
		Assignee: r.Assignee,
		Labels:   r.Labels,
	}
}

// PatchIssueRequest represents JSON Merge Patch request for issues
type PatchIssueRequest struct {
	Title    *string  `json:"title,omitempty"`
	Content  *string  `json:"content,omitempty"`
	Status   *string  `json:"status,omitempty"`
	Priority *string  `json:"priority,omitempty"`
	Assignee *string  `json:"assignee,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

// IssueResponse represents the HTTP response for an issue
type IssueResponse struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"`
	Title    string    `json:"title"`
	Status   string    `json:"status"`
	Priority string    `json:"priority"`
	Assignee string    `json:"assignee,omitempty"`
	Labels   []string  `json:"labels,omitempty"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
	Version  int       `json:"version"`
	FilePath string    `json:"filepath"`
	Content  string    `json:"content,omitempty"`
}

// FromDomainIssue creates HTTP response from domain issue
func FromDomainIssue(issue *domain.Issue) *IssueResponse {
	return &IssueResponse{
		ID:       issue.ID,
		Type:     issue.Type,
		Title:    issue.Title,
		Status:   issue.Status,
		Priority: issue.Priority,
		Assignee: issue.Assignee,
		Labels:   issue.Labels,
		Created:  issue.Created,
		Updated:  issue.Updated,
		Version:  issue.Version,
		FilePath: issue.FilePath,
		Content:  issue.Content,
	}
}

// IssueResponseFromDomain creates HTTP response from domain issue (legacy)
func IssueResponseFromDomain(issue *domain.Issue) IssueResponse {
	return IssueResponse{
		ID:       issue.ID,
		Type:     issue.Type,
		Title:    issue.Title,
		Status:   issue.Status,
		Priority: issue.Priority,
		Assignee: issue.Assignee,
		Labels:   issue.Labels,
		Created:  issue.Created,
		Updated:  issue.Updated,
		Version:  issue.Version,
		FilePath: issue.FilePath,
		Content:  issue.Content,
	}
}

// IssueListResponse represents a paginated list of issues
type IssueListResponse struct {
	Issues []IssueResponse `json:"issues"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// TransitionRequest represents a status transition request
type TransitionRequest struct {
	From   string `json:"from" binding:"required"`
	To     string `json:"to" binding:"required"`
	Reason string `json:"reason"`
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
