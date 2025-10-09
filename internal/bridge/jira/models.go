package jira

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// jiraIssueResponse represents the JSON response from Jira API
//
// Example response from POST /rest/api/3/search/jql:
//
//	{
//	  "id": "10849",
//	  "key": "INS-1",
//	  "fields": {
//	    "summary": "Create a PoC system model",
//	    "description": {
//	      "type": "doc",
//	      "version": 1,
//	      "content": [
//	        {
//	          "type": "paragraph",
//	          "content": [
//	            {
//	              "type": "text",
//	              "text": "Create a system model able to provide context..."
//	            },
//	            {
//	              "type": "text",
//	              "text": "Which products have the highest profit margins?",
//	              "marks": [{"type": "strong"}]
//	            }
//	          ]
//	        }
//	      ]
//	    },
//	    "status": {
//	      "name": "Done",
//	      "statusCategory": {"name": "Done", "colorName": "green"}
//	    },
//	    "assignee": {
//	      "displayName": "User123",
//	      "accountId": "712020:45533243-a51d-418d-a609-18e03bd44339"
//	    },
//	    "reporter": {
//	      "displayName": "User123"
//	    },
//	    "created": "2024-12-13T12:21:09.078+0100",
//	    "updated": "2024-12-23T10:19:45.215+0100",
//	    "labels": [],
//	    "comment": {
//	      "comments": [],
//	      "total": 0
//	    },
//	    "attachment": []
//	  }
//	}
//
// Note: description can be null or ADF object (as shown)
type jiraIssueResponse struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		// API v3 uses ADF (Atlassian Document Format) for rich text
		Description json.RawMessage `json:"description"`
		Status      struct {
			Name string `json:"name"`
		} `json:"status"`
		Assignee *struct {
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
		Reporter struct {
			DisplayName string `json:"displayName"`
		} `json:"reporter"`
		Created jiraTime `json:"created"`
		Updated jiraTime `json:"updated"`
		Labels  []string `json:"labels"`
		Comment struct {
			Comments []jiraComment `json:"comments"`
		} `json:"comment"`
		Attachment []jiraAttachment `json:"attachment"`
	} `json:"fields"`
}

// jiraComment represents a comment on an issue
//
// Example comment structure (ADF format):
//
//	{
//	  "id": "10000",
//	  "author": {
//	    "displayName": "John Doe",
//	    "accountId": "712020:..."
//	  },
//	  "body": {
//	    "type": "doc",
//	    "version": 1,
//	    "content": [
//	      {
//	        "type": "paragraph",
//	        "content": [
//	          {
//	            "type": "text",
//	            "text": "This is a comment"
//	          }
//	        ]
//	      }
//	    ]
//	  },
//	  "created": "2024-12-13T12:21:09.078+0100",
//	  "updated": "2024-12-13T12:21:09.078+0100"
//	}
type jiraComment struct {
	ID     string `json:"id"`
	Author struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
	// API v3 uses ADF (Atlassian Document Format) for rich text
	Body    json.RawMessage `json:"body"`
	Created jiraTime        `json:"created"`
	Updated jiraTime        `json:"updated"`
}

type jiraAttachment struct {
	ID       string   `json:"id"`
	Filename string   `json:"filename"`
	Content  string   `json:"content"`
	MimeType string   `json:"mimeType"`
	Size     int64    `json:"size"`
	Created  jiraTime `json:"created"`
}

// jiraTime is a custom type to handle Jira's timestamp format
type jiraTime struct {
	time.Time
}

// UnmarshalJSON handles Jira's timestamp format: "2025-10-07T17:59:40.253+0200"
func (jt *jiraTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = strings.Trim(s, `"`)

	// Jira API v3 primarily uses the millisecond format, but we try fallbacks for robustness
	formats := []string{
		"2006-01-02T15:04:05.999-0700", // Jira format with milliseconds (primary)
		"2006-01-02T15:04:05-0700",     // Jira format without milliseconds
		time.RFC3339,                   // Standard RFC3339 (fallback)
		time.RFC3339Nano,               // RFC3339 with nanoseconds (fallback)
	}

	var err error
	for _, format := range formats {
		jt.Time, err = time.Parse(format, s)
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("unable to parse time %q: %w", s, err)
}

// jiraSearchResponse represents the paginated search response
//
// Example response from POST /rest/api/3/search/jql:
//
//	{
//	  "issues": [
//	    { /* issue objects - see jiraIssueResponse */ }
//	  ],
//	  "nextPageToken": "Ckl1cGRhdGVkJnVwZGF0ZWQmT1JERVJfREVTQyNMb25nJk1UYzFP...",
//	  "isLast": false
//	}
//
// The nextPageToken is used for pagination - include it in the next request
// instead of using traditional startAt/maxResults (deprecated in API v3)
type jiraSearchResponse struct {
	Issues        []jiraIssueResponse `json:"issues"`
	Total         int                 `json:"total"`
	NextPageToken string              `json:"nextPageToken,omitempty"`
}
