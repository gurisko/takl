package git

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Repository interface abstracts git operations for testing
type Repository interface {
	IsClean() (bool, error)
	AddAndCommit(filePath, message string, author *object.Signature) error
	GetFileHistory(filePath string, maxCount int) ([]*object.Commit, error)
}

// gitRepository is the concrete implementation of Repository
type gitRepository struct {
	repo     *git.Repository
	worktree *git.Worktree
	path     string
}

func OpenRepository(path string) (Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	return &gitRepository{
		repo:     repo,
		worktree: worktree,
		path:     path,
	}, nil
}

func (r *gitRepository) IsClean() (bool, error) {
	status, err := r.worktree.Status()
	if err != nil {
		return false, err
	}
	return status.IsClean(), nil
}

func (r *gitRepository) AddAndCommit(filePath, message string, author *object.Signature) error {
	// Add file to staging area
	_, err := r.worktree.Add(filePath)
	if err != nil {
		return fmt.Errorf("failed to add file %s: %w", filePath, err)
	}

	// Set default author if not provided
	if author == nil {
		author = &object.Signature{
			Name:  "TAKL User",
			Email: "takl@localhost",
			When:  time.Now(),
		}
	}

	// Commit changes
	_, err = r.worktree.Commit(message, &git.CommitOptions{
		Author: author,
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

func (r *gitRepository) GetFileHistory(filePath string, maxCount int) ([]*object.Commit, error) {
	// Get commit history for file
	commits, err := r.repo.Log(&git.LogOptions{
		FileName: &filePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file history: %w", err)
	}

	var result []*object.Commit
	count := 0
	err = commits.ForEach(func(c *object.Commit) error {
		if maxCount > 0 && count >= maxCount {
			return fmt.Errorf("limit reached")
		}
		result = append(result, c)
		count++
		return nil
	})

	if err != nil && err.Error() != "limit reached" {
		return nil, err
	}

	return result, nil
}
