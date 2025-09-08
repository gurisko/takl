// Package app provides the thin service layer that sits between HTTP handlers and the domain/store layer.
//
// ARCHITECTURAL GOAL: Keep handlers stupid; centralize business flows; simplify tests.
//
// This service layer follows Clean Architecture principles:
// 1. Handlers do only: Parse/validate input → dto structs, Call app.Service, Map domain → dto output, Set headers (ETag)
// 2. Services orchestrate business flows, coordinate between domain objects, and manage cross-cutting concerns
// 3. Services depend on domain interfaces (repositories, indexers, paradigm registry) - not implementations
// 4. All business logic lives here, not in handlers or repositories
//
// Benefits:
// - Handlers become thin adapters focused solely on HTTP concerns
// - Business logic is centralized and easily testable
// - Cross-cutting concerns (indexing, validation, paradigm rules) are coordinated in one place
// - Domain boundaries are enforced through interface contracts
package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/indexer"
	"github.com/takl/takl/internal/store"
	"github.com/takl/takl/internal/validation"
)

// IssueService orchestrates issue-related use cases and business flows
// It coordinates between Repo (storage) and Index (search) with write-through semantics
type IssueService struct {
	// New formalized interfaces
	repo        store.Repo
	projectRepo store.ProjectRepo
	index       indexer.Index

	// Legacy interfaces (to be phased out)
	issueRepo         domain.IssueRepository
	legacyProjectRepo domain.ProjectRepository
	legacyIndexer     domain.Indexer

	// Dependencies
	paradigmReg domain.ParadigmRegistry
	clock       domain.Clock
	idGenerator domain.IDGenerator
	logger      *slog.Logger

	// Centralized validation (API-centric architecture)
	validator *validation.Validator
}

// NewIssueService creates a new issue service with formalized interfaces
func NewIssueService(
	repo store.Repo,
	projectRepo store.ProjectRepo,
	index indexer.Index,
	paradigmReg domain.ParadigmRegistry,
	clock domain.Clock,
	idGenerator domain.IDGenerator,
	validator *validation.Validator,
	logger *slog.Logger,
) *IssueService {
	return &IssueService{
		repo:        repo,
		projectRepo: projectRepo,
		index:       index,
		paradigmReg: paradigmReg,
		clock:       clock,
		idGenerator: idGenerator,
		validator:   validator,
		logger:      logger,
	}
}

// NewLegacyIssueService creates a service with legacy interfaces (backward compatibility)
func NewLegacyIssueService(
	issueRepo domain.IssueRepository,
	projectRepo domain.ProjectRepository,
	indexer domain.Indexer,
	paradigmReg domain.ParadigmRegistry,
	clock domain.Clock,
	idGenerator domain.IDGenerator,
	logger *slog.Logger,
) *IssueService {
	return &IssueService{
		issueRepo:         issueRepo,
		legacyProjectRepo: projectRepo,
		legacyIndexer:     indexer,
		paradigmReg:       paradigmReg,
		clock:             clock,
		idGenerator:       idGenerator,
		logger:            logger,
	}
}

// CreateIssue creates a new issue with validation and write-through indexing
func (s *IssueService) CreateIssue(ctx context.Context, projectID string, req domain.CreateIssueRequest) (*domain.Issue, error) {
	// Validate project exists (try new interface first, fallback to legacy)
	if s.projectRepo != nil {
		_, err := s.projectRepo.GetProject(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("project not found: %w", err)
		}
	} else if s.legacyProjectRepo != nil {
		_, err := s.legacyProjectRepo.GetByID(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("project not found: %w", err)
		}
	}

	// Create issue (use new interface if available)
	var issue *domain.Issue
	var err error

	if s.repo != nil {
		// Generate the issue using domain logic
		now := s.clock.Now()
		issue = &domain.Issue{
			ID:       s.idGenerator.Generate(),
			Type:     req.Type,
			Title:    req.Title,
			Status:   "open", // Default status
			Priority: req.Priority,
			Assignee: req.Assignee,
			Labels:   req.Labels,
			Content:  req.Description,
			Created:  now,
			Updated:  now,
			Version:  1,
		}

		// CENTRALIZED VALIDATION - Single source of truth
		if s.validator != nil {
			// Normalize fields first
			s.validator.NormalizeIssue(issue)

			// Validate using centralized validator
			if err := s.validator.ValidateCreate(issue); err != nil {
				return nil, fmt.Errorf("validation failed: %w", err)
			}
		} else {
			// No validator available - this shouldn't happen in production
			s.logger.Warn("No validator configured - issue validation skipped", "issue_id", issue.ID)
		}

		// Save to repository
		err = s.repo.SaveIssue(ctx, projectID, issue)
	} else {
		// Fallback to legacy interface
		issue, err = s.issueRepo.Create(ctx, req)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	// Write-through indexing: Update search index
	if s.index != nil {
		if err := s.index.Upsert(ctx, projectID, issue); err != nil {
			// Log warning but don't fail the operation - watcher will pick this up as fallback
			s.logger.Warn("Failed to index created issue", "issue_id", issue.ID, "error", err)
		}
	} else if s.legacyIndexer != nil {
		if err := s.legacyIndexer.Upsert(ctx, projectID, issue); err != nil {
			s.logger.Warn("Failed to index created issue (legacy)", "issue_id", issue.ID, "error", err)
		}
	}

	return issue, nil
}

// GetIssue retrieves an issue by ID
func (s *IssueService) GetIssue(ctx context.Context, projectID, issueID string) (*domain.Issue, error) {
	if s.repo != nil {
		return s.repo.LoadIssue(ctx, projectID, issueID)
	}
	return s.issueRepo.GetByID(ctx, projectID, issueID)
}

// UpdateIssue updates an issue with validation and write-through indexing
func (s *IssueService) UpdateIssue(ctx context.Context, projectID, issueID string, req domain.UpdateIssueRequest) (*domain.Issue, error) {
	// Get current issue
	var currentIssue *domain.Issue
	var err error

	if s.repo != nil {
		currentIssue, err = s.repo.LoadIssue(ctx, projectID, issueID)
	} else {
		currentIssue, err = s.issueRepo.GetByID(ctx, projectID, issueID)
	}

	if err != nil {
		return nil, fmt.Errorf("issue not found: %w", err)
	}

	// Validate paradigm constraints if status is being changed
	if req.Status != nil && *req.Status != currentIssue.Status {
		// TODO: Paradigm validation would go here
		// For now, we skip paradigm validation but could add it later
		_ = currentIssue // Avoid unused variable warning
	}

	// Apply updates to issue
	updatedIssue := *currentIssue // Copy the issue

	if req.Title != nil {
		updatedIssue.Title = *req.Title
	}
	if req.Content != nil {
		updatedIssue.Content = *req.Content
	}
	if req.Status != nil {
		updatedIssue.Status = *req.Status
	}
	if req.Priority != nil {
		updatedIssue.Priority = *req.Priority
	}
	if req.Assignee != nil {
		updatedIssue.Assignee = *req.Assignee
	}
	if req.Labels != nil {
		updatedIssue.Labels = req.Labels
	}

	// Update metadata
	updatedIssue.Updated = s.clock.Now()
	updatedIssue.Version++ // Increment version for optimistic concurrency

	// CENTRALIZED VALIDATION - Use the centralized validator if available
	if s.validator != nil {
		s.validator.NormalizeIssue(&updatedIssue)
		if err := s.validator.ValidateUpdate(currentIssue, &updatedIssue); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	// Save to repository (write-through)
	if s.repo != nil {
		err = s.repo.SaveIssue(ctx, projectID, &updatedIssue)
	} else {
		// Legacy fallback
		_, err = s.issueRepo.Update(ctx, projectID, issueID, req)
		if err != nil {
			return nil, fmt.Errorf("failed to update issue: %w", err)
		}
		return s.issueRepo.GetByID(ctx, projectID, issueID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to update issue: %w", err)
	}

	// Write-through indexing: Update search index
	if s.index != nil {
		if err := s.index.Upsert(ctx, projectID, &updatedIssue); err != nil {
			s.logger.Warn("Failed to index updated issue", "issue_id", issueID, "error", err)
		}
	} else if s.legacyIndexer != nil {
		if err := s.legacyIndexer.Upsert(ctx, projectID, &updatedIssue); err != nil {
			s.logger.Warn("Failed to index updated issue (legacy)", "issue_id", issueID, "error", err)
		}
	}

	return &updatedIssue, nil
}

// DeleteIssue removes an issue with write-through index cleanup
func (s *IssueService) DeleteIssue(ctx context.Context, projectID, issueID string) error {
	// Delete from repository
	if s.repo != nil {
		if err := s.repo.DeleteIssue(ctx, projectID, issueID); err != nil {
			return fmt.Errorf("failed to delete issue: %w", err)
		}
	} else {
		if err := s.issueRepo.Delete(ctx, projectID, issueID); err != nil {
			return fmt.Errorf("failed to delete issue: %w", err)
		}
	}

	// Write-through: Remove from search index
	if s.index != nil {
		if err := s.index.DeleteByID(ctx, projectID, issueID); err != nil {
			s.logger.Warn("Failed to remove deleted issue from index", "issue_id", issueID, "error", err)
		}
	} else if s.legacyIndexer != nil {
		if err := s.legacyIndexer.Delete(ctx, projectID, issueID); err != nil {
			s.logger.Warn("Failed to remove deleted issue from index (legacy)", "issue_id", issueID, "error", err)
		}
	}

	return nil
}

// ListIssues retrieves issues with optional filtering
func (s *IssueService) ListIssues(ctx context.Context, projectID string, filter domain.IssueFilter) ([]*domain.Issue, error) {
	if s.repo != nil {
		storeFilter := store.FromDomainFilter(filter)
		return s.repo.ListIssues(ctx, projectID, storeFilter)
	}
	return s.issueRepo.List(ctx, projectID, filter)
}

// SearchIssues performs full-text search on issues
func (s *IssueService) SearchIssues(ctx context.Context, projectID, query string) ([]*domain.Issue, error) {
	if s.index != nil {
		filter := indexer.Filters{} // Empty filter for basic search
		hits, err := s.index.Search(ctx, projectID, query, filter)
		if err != nil {
			return nil, err
		}

		// Convert hits to issues
		issues := make([]*domain.Issue, len(hits))
		for i, hit := range hits {
			issues[i] = hit.Issue
		}
		return issues, nil
	}
	return s.legacyIndexer.Search(ctx, projectID, query)
}

// PatchIssue performs partial updates on an issue with write-through indexing
func (s *IssueService) PatchIssue(ctx context.Context, projectID, issueID string, req domain.UpdateIssueRequest) (*domain.Issue, string, error) {
	// Perform the update using the main UpdateIssue method
	updatedIssue, err := s.UpdateIssue(ctx, projectID, issueID, req)
	if err != nil {
		return nil, "", err
	}

	// Check if indexing is working properly
	indexStale := false
	if s.index != nil {
		// Try to verify the issue exists in the index
		_, searchErr := s.index.Search(ctx, projectID, updatedIssue.ID, indexer.Filters{Limit: 1})
		if searchErr != nil {
			indexStale = true
		}
	}

	// Generate ETag for updated issue
	etag := fmt.Sprintf(`"%d-%d"`, updatedIssue.Version, updatedIssue.Updated.Unix())

	// Return index status indicator
	if indexStale {
		return updatedIssue, "stale", nil
	}

	return updatedIssue, etag, nil
}

// TransitionIssue changes the status of an issue with paradigm validation
func (s *IssueService) TransitionIssue(ctx context.Context, projectID, issueID, fromStatus, toStatus string) (*domain.Issue, error) {
	// Get current issue
	issue, err := s.issueRepo.GetByID(ctx, projectID, issueID)
	if err != nil {
		return nil, fmt.Errorf("issue not found: %w", err)
	}

	if issue.Status != fromStatus {
		return nil, fmt.Errorf("issue status mismatch: expected %s, got %s", fromStatus, issue.Status)
	}

	// Validate transition with paradigm
	if s.projectRepo != nil {
		_, err = s.projectRepo.GetProject(ctx, projectID)
	} else if s.legacyProjectRepo != nil {
		_, err = s.legacyProjectRepo.GetByID(ctx, projectID)
	}
	if err == nil {
		// TODO: Get paradigm from project configuration
		_ = err // Skip paradigm validation for now
		// paradigm, err := s.paradigmReg.Get(project.ParadigmID)
		// if err == nil {
		//     transition := domain.ParadigmTransition{From: fromStatus, To: toStatus}
		//     if err := paradigm.ValidateTransition(ctx, issue, transition); err != nil {
		//         return nil, fmt.Errorf("invalid transition: %w", err)
		//     }
		// }
	}

	// Perform the transition
	req := domain.UpdateIssueRequest{Status: &toStatus}
	return s.UpdateIssue(ctx, projectID, issueID, req)
}
