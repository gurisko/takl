package jira

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestADFToMarkdown_SimpleText tests conversion of simple text
func TestADFToMarkdown_SimpleText(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "Hello, world!"
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "Hello, world!"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Bold tests bold text conversion
func TestADFToMarkdown_Bold(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "This is ",
						"marks": []
					},
					{
						"type": "text",
						"text": "bold",
						"marks": [{"type": "strong"}]
					},
					{
						"type": "text",
						"text": " text"
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "This is **bold** text"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Italic tests italic text conversion
func TestADFToMarkdown_Italic(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "This is "
					},
					{
						"type": "text",
						"text": "italic",
						"marks": [{"type": "em"}]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "This is *italic*"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_InlineCode tests inline code conversion
func TestADFToMarkdown_InlineCode(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "Use "
					},
					{
						"type": "text",
						"text": "fmt.Println()",
						"marks": [{"type": "code"}]
					},
					{
						"type": "text",
						"text": " to print"
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "Use `fmt.Println()` to print"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Strikethrough tests strikethrough text conversion
func TestADFToMarkdown_Strikethrough(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "This is "
					},
					{
						"type": "text",
						"text": "deleted",
						"marks": [{"type": "strike"}]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "This is ~~deleted~~"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Link tests link conversion
func TestADFToMarkdown_Link(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "Visit "
					},
					{
						"type": "text",
						"text": "Google",
						"marks": [
							{
								"type": "link",
								"attrs": {
									"href": "https://google.com"
								}
							}
						]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "Visit [Google](https://google.com)"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Heading tests heading conversion
func TestADFToMarkdown_Heading(t *testing.T) {
	tests := []struct {
		level    int
		expected string
	}{
		{1, "# Heading 1"},
		{2, "## Heading 2"},
		{3, "### Heading 3"},
		{4, "#### Heading 4"},
		{5, "##### Heading 5"},
		{6, "###### Heading 6"},
	}

	for _, tt := range tests {
		adf := `{
			"type": "doc",
			"version": 1,
			"content": [
				{
					"type": "heading",
					"attrs": {"level": ` + string(rune(tt.level+'0')) + `},
					"content": [
						{
							"type": "text",
							"text": "Heading ` + string(rune(tt.level+'0')) + `"
						}
					]
				}
			]
		}`

		result, err := ADFToMarkdown(json.RawMessage(adf))
		if err != nil {
			t.Fatalf("ADFToMarkdown failed for level %d: %v", tt.level, err)
		}

		if result != tt.expected {
			t.Errorf("Level %d: Expected %q, got %q", tt.level, tt.expected, result)
		}
	}
}

// TestADFToMarkdown_BulletList tests unordered list conversion
func TestADFToMarkdown_BulletList(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "bulletList",
				"content": [
					{
						"type": "listItem",
						"content": [
							{
								"type": "paragraph",
								"content": [
									{
										"type": "text",
										"text": "First item"
									}
								]
							}
						]
					},
					{
						"type": "listItem",
						"content": [
							{
								"type": "paragraph",
								"content": [
									{
										"type": "text",
										"text": "Second item"
									}
								]
							}
						]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "- First item\n- Second item"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_OrderedList tests ordered list conversion
func TestADFToMarkdown_OrderedList(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "orderedList",
				"content": [
					{
						"type": "listItem",
						"content": [
							{
								"type": "paragraph",
								"content": [
									{
										"type": "text",
										"text": "First step"
									}
								]
							}
						]
					},
					{
						"type": "listItem",
						"content": [
							{
								"type": "paragraph",
								"content": [
									{
										"type": "text",
										"text": "Second step"
									}
								]
							}
						]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "1. First step\n2. Second step"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_CodeBlock tests code block conversion
func TestADFToMarkdown_CodeBlock(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "codeBlock",
				"attrs": {"language": "go"},
				"content": [
					{
						"type": "text",
						"text": "func main() {\n\tfmt.Println(\"Hello\")\n}"
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Blockquote tests blockquote conversion
func TestADFToMarkdown_Blockquote(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "blockquote",
				"content": [
					{
						"type": "paragraph",
						"content": [
							{
								"type": "text",
								"text": "This is a quote"
							}
						]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "> This is a quote"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_HorizontalRule tests horizontal rule conversion
func TestADFToMarkdown_HorizontalRule(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "Above"
					}
				]
			},
			{
				"type": "rule"
			},
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "Below"
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "Above\n\n---\n\nBelow"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_MultipleParagraphs tests multiple paragraphs
func TestADFToMarkdown_MultipleParagraphs(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "First paragraph"
					}
				]
			},
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "Second paragraph"
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "First paragraph\n\nSecond paragraph"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_ComplexFormatting tests multiple marks on same text
func TestADFToMarkdown_ComplexFormatting(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "bold and italic",
						"marks": [
							{"type": "strong"},
							{"type": "em"}
						]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "***bold and italic***"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Subscript tests subscript conversion to BBCode
func TestADFToMarkdown_Subscript(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "H"
					},
					{
						"type": "text",
						"text": "2",
						"marks": [{"type": "subsup", "attrs": {"type": "sub"}}]
					},
					{
						"type": "text",
						"text": "O"
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "H<sub>2</sub>O"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Superscript tests superscript conversion to BBCode
func TestADFToMarkdown_Superscript(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "E=mc"
					},
					{
						"type": "text",
						"text": "2",
						"marks": [{"type": "subsup", "attrs": {"type": "sup"}}]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "E=mc<sup>2</sup>"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_TextColor tests text color conversion to BBCode
func TestADFToMarkdown_TextColor(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "This is "
					},
					{
						"type": "text",
						"text": "red",
						"marks": [{"type": "textColor", "attrs": {"color": "#ff0000"}}]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "This is <span style=\"color: #ff0000\">red</span>"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Border tests border conversion to BBCode
func TestADFToMarkdown_Border(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "This is "
					},
					{
						"type": "text",
						"text": "boxed",
						"marks": [{"type": "border", "attrs": {"color": "#000000"}}]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "This is <span style=\"border: 1px solid #000000\">boxed</span>"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Underline tests underline conversion to BBCode
func TestADFToMarkdown_Underline(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "This is "
					},
					{
						"type": "text",
						"text": "underlined",
						"marks": [{"type": "underline"}]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "This is <u>underlined</u>"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Empty tests empty document
func TestADFToMarkdown_Empty(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": []
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := ""
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestMarkdownToADF_SimpleText tests conversion of simple text
func TestMarkdownToADF_SimpleText(t *testing.T) {
	markdown := "Hello, world!"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if doc.Type != "doc" || doc.Version != 1 {
		t.Errorf("Invalid document structure")
	}

	if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
		t.Errorf("Expected single paragraph")
	}

	if len(doc.Content[0].Content) != 1 || doc.Content[0].Content[0].Text != "Hello, world!" {
		t.Errorf("Expected text 'Hello, world!', got %q", doc.Content[0].Content[0].Text)
	}
}

// TestMarkdownToADF_Bold tests bold text conversion
func TestMarkdownToADF_Bold(t *testing.T) {
	markdown := "This is **bold** text"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	content := doc.Content[0].Content
	if len(content) != 3 {
		t.Fatalf("Expected 3 text nodes, got %d", len(content))
	}

	if content[0].Text != "This is " || len(content[0].Marks) != 0 {
		t.Errorf("Expected plain text 'This is ', got %q with %d marks", content[0].Text, len(content[0].Marks))
	}

	if content[1].Text != "bold" || len(content[1].Marks) != 1 || content[1].Marks[0].Type != "strong" {
		t.Errorf("Expected bold 'bold', got %q with marks %+v", content[1].Text, content[1].Marks)
	}

	if content[2].Text != " text" || len(content[2].Marks) != 0 {
		t.Errorf("Expected plain text ' text', got %q with %d marks", content[2].Text, len(content[2].Marks))
	}
}

// TestMarkdownToADF_Italic tests italic text conversion
func TestMarkdownToADF_Italic(t *testing.T) {
	markdown := "This is *italic* text"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	content := doc.Content[0].Content
	if content[1].Text != "italic" || content[1].Marks[0].Type != "em" {
		t.Errorf("Expected italic mark, got %+v", content[1].Marks)
	}
}

// TestMarkdownToADF_InlineCode tests inline code conversion
func TestMarkdownToADF_InlineCode(t *testing.T) {
	markdown := "Use `fmt.Println()` to print"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	content := doc.Content[0].Content
	if content[1].Text != "fmt.Println()" || content[1].Marks[0].Type != "code" {
		t.Errorf("Expected code mark on 'fmt.Println()', got %+v", content[1])
	}
}

// TestMarkdownToADF_Strikethrough tests strikethrough conversion
func TestMarkdownToADF_Strikethrough(t *testing.T) {
	markdown := "This is ~~deleted~~ text"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	content := doc.Content[0].Content
	if content[1].Text != "deleted" || content[1].Marks[0].Type != "strike" {
		t.Errorf("Expected strike mark, got %+v", content[1].Marks)
	}
}

// TestMarkdownToADF_Link tests link conversion
func TestMarkdownToADF_Link(t *testing.T) {
	markdown := "Visit [Google](https://google.com) now"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	content := doc.Content[0].Content
	if len(content) < 2 {
		t.Fatalf("Expected at least 2 nodes, got %d", len(content))
	}

	linkNode := content[1]
	if linkNode.Text != "Google" {
		t.Errorf("Expected link text 'Google', got %q", linkNode.Text)
	}
	if len(linkNode.Marks) != 1 || linkNode.Marks[0].Type != "link" {
		t.Errorf("Expected link mark, got %+v", linkNode.Marks)
	}
	if href, ok := linkNode.Marks[0].Attrs["href"].(string); !ok || href != "https://google.com" {
		t.Errorf("Expected href 'https://google.com', got %v", linkNode.Marks[0].Attrs["href"])
	}
}

// TestMarkdownToADF_Heading tests heading conversion
func TestMarkdownToADF_Heading(t *testing.T) {
	tests := []struct {
		markdown string
		level    int
	}{
		{"# Heading 1", 1},
		{"## Heading 2", 2},
		{"### Heading 3", 3},
		{"#### Heading 4", 4},
		{"##### Heading 5", 5},
		{"###### Heading 6", 6},
	}

	for _, tt := range tests {
		result, err := MarkdownToADF(tt.markdown)
		if err != nil {
			t.Fatalf("MarkdownToADF failed for %q: %v", tt.markdown, err)
		}

		var doc ADFDocument
		if err := json.Unmarshal(result, &doc); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		if len(doc.Content) != 1 || doc.Content[0].Type != "heading" {
			t.Errorf("Expected heading node, got %+v", doc.Content[0])
		}

		level := int(doc.Content[0].Attrs["level"].(float64))
		if level != tt.level {
			t.Errorf("Expected level %d, got %d", tt.level, level)
		}
	}
}

// TestMarkdownToADF_BulletList tests unordered list conversion
func TestMarkdownToADF_BulletList(t *testing.T) {
	markdown := `- First item
- Second item`

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(doc.Content) != 1 || doc.Content[0].Type != "bulletList" {
		t.Errorf("Expected bulletList, got %+v", doc.Content[0])
	}

	items := doc.Content[0].Content
	if len(items) != 2 {
		t.Errorf("Expected 2 list items, got %d", len(items))
	}
}

// TestMarkdownToADF_OrderedList tests ordered list conversion
func TestMarkdownToADF_OrderedList(t *testing.T) {
	markdown := `1. First step
2. Second step`

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(doc.Content) != 1 || doc.Content[0].Type != "orderedList" {
		t.Errorf("Expected orderedList, got %+v", doc.Content[0])
	}

	items := doc.Content[0].Content
	if len(items) != 2 {
		t.Errorf("Expected 2 list items, got %d", len(items))
	}
}

// TestMarkdownToADF_CodeBlock tests code block conversion
func TestMarkdownToADF_CodeBlock(t *testing.T) {
	markdown := "```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(doc.Content) != 1 || doc.Content[0].Type != "codeBlock" {
		t.Errorf("Expected codeBlock, got %+v", doc.Content[0])
	}

	lang := doc.Content[0].Attrs["language"].(string)
	if lang != "go" {
		t.Errorf("Expected language 'go', got %q", lang)
	}

	code := doc.Content[0].Content[0].Text
	expected := "func main() {\n\tfmt.Println(\"Hello\")\n}"
	if code != expected {
		t.Errorf("Expected code %q, got %q", expected, code)
	}
}

// TestMarkdownToADF_Blockquote tests blockquote conversion
func TestMarkdownToADF_Blockquote(t *testing.T) {
	markdown := "> This is a quote"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(doc.Content) != 1 || doc.Content[0].Type != "blockquote" {
		t.Errorf("Expected blockquote, got %+v", doc.Content[0])
	}
}

// TestMarkdownToADF_HorizontalRule tests horizontal rule conversion
func TestMarkdownToADF_HorizontalRule(t *testing.T) {
	tests := []string{"---", "***", "___"}

	for _, markdown := range tests {
		result, err := MarkdownToADF(markdown)
		if err != nil {
			t.Fatalf("MarkdownToADF failed for %q: %v", markdown, err)
		}

		var doc ADFDocument
		if err := json.Unmarshal(result, &doc); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		if len(doc.Content) != 1 || doc.Content[0].Type != "rule" {
			t.Errorf("Expected rule for %q, got %+v", markdown, doc.Content[0])
		}
	}
}

// TestMarkdownToADF_Empty tests empty markdown
func TestMarkdownToADF_Empty(t *testing.T) {
	markdown := ""

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(doc.Content) != 0 {
		t.Errorf("Expected empty content, got %d nodes", len(doc.Content))
	}
}

// TestRoundTrip_SimpleText tests round-trip conversion of simple text
func TestRoundTrip_SimpleText(t *testing.T) {
	original := "Hello, world!"

	// Markdown -> ADF
	adf, err := MarkdownToADF(original)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	// ADF -> Markdown
	result, err := ADFToMarkdown(adf)
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	if result != original {
		t.Errorf("Round-trip failed: expected %q, got %q", original, result)
	}
}

// TestRoundTrip_FormattedText tests round-trip with formatting
func TestRoundTrip_FormattedText(t *testing.T) {
	tests := []string{
		"This is **bold** text",
		"This is *italic* text",
		"This is `code` text",
		"This is ~~deleted~~ text",
		"# Heading 1",
		"## Heading 2",
	}

	for _, original := range tests {
		// Markdown -> ADF
		adf, err := MarkdownToADF(original)
		if err != nil {
			t.Fatalf("MarkdownToADF failed for %q: %v", original, err)
		}

		// ADF -> Markdown
		result, err := ADFToMarkdown(adf)
		if err != nil {
			t.Fatalf("ADFToMarkdown failed for %q: %v", original, err)
		}

		if result != original {
			t.Errorf("Round-trip failed for %q: got %q", original, result)
		}
	}
}

// TestRoundTrip_Lists tests round-trip with lists
func TestRoundTrip_Lists(t *testing.T) {
	tests := []string{
		"- First item\n- Second item",
		"1. First step\n2. Second step",
	}

	for _, original := range tests {
		// Markdown -> ADF
		adf, err := MarkdownToADF(original)
		if err != nil {
			t.Fatalf("MarkdownToADF failed for %q: %v", original, err)
		}

		// ADF -> Markdown
		result, err := ADFToMarkdown(adf)
		if err != nil {
			t.Fatalf("ADFToMarkdown failed for %q: %v", original, err)
		}

		if strings.TrimSpace(result) != strings.TrimSpace(original) {
			t.Errorf("Round-trip failed for %q: got %q", original, result)
		}
	}
}

// TestADFToMarkdown_NestedBulletList tests nested unordered list conversion
func TestADFToMarkdown_NestedBulletList(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "bulletList",
				"content": [
					{
						"type": "listItem",
						"content": [
							{
								"type": "paragraph",
								"content": [
									{
										"type": "text",
										"text": "Parent item 1"
									}
								]
							},
							{
								"type": "bulletList",
								"content": [
									{
										"type": "listItem",
										"content": [
											{
												"type": "paragraph",
												"content": [
													{
														"type": "text",
														"text": "Child item 1"
													}
												]
											}
										]
									},
									{
										"type": "listItem",
										"content": [
											{
												"type": "paragraph",
												"content": [
													{
														"type": "text",
														"text": "Child item 2"
													}
												]
											}
										]
									}
								]
							}
						]
					},
					{
						"type": "listItem",
						"content": [
							{
								"type": "paragraph",
								"content": [
									{
										"type": "text",
										"text": "Parent item 2"
									}
								]
							}
						]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "- Parent item 1\n  - Child item 1\n  - Child item 2\n- Parent item 2"
	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// TestADFToMarkdown_NestedOrderedList tests nested ordered list conversion
func TestADFToMarkdown_NestedOrderedList(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "orderedList",
				"content": [
					{
						"type": "listItem",
						"content": [
							{
								"type": "paragraph",
								"content": [
									{
										"type": "text",
										"text": "Step 1"
									}
								]
							},
							{
								"type": "orderedList",
								"content": [
									{
										"type": "listItem",
										"content": [
											{
												"type": "paragraph",
												"content": [
													{
														"type": "text",
														"text": "Sub-step 1.1"
													}
												]
											}
										]
									},
									{
										"type": "listItem",
										"content": [
											{
												"type": "paragraph",
												"content": [
													{
														"type": "text",
														"text": "Sub-step 1.2"
													}
												]
											}
										]
									}
								]
							}
						]
					},
					{
						"type": "listItem",
						"content": [
							{
								"type": "paragraph",
								"content": [
									{
										"type": "text",
										"text": "Step 2"
									}
								]
							}
						]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "1. Step 1\n  1. Sub-step 1.1\n  2. Sub-step 1.2\n2. Step 2"
	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// TestMarkdownToADF_NestedBulletList tests nested unordered list conversion from Markdown
func TestMarkdownToADF_NestedBulletList(t *testing.T) {
	markdown := `- Parent item 1
  - Child item 1
  - Child item 2
- Parent item 2`

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(doc.Content) != 1 || doc.Content[0].Type != "bulletList" {
		t.Fatalf("Expected bulletList, got %+v", doc.Content[0])
	}

	items := doc.Content[0].Content
	if len(items) != 2 {
		t.Fatalf("Expected 2 parent items, got %d", len(items))
	}

	// Check first parent item has a nested list
	firstItem := items[0]
	if len(firstItem.Content) < 2 {
		t.Fatalf("Expected first item to have paragraph and nested list, got %d content nodes", len(firstItem.Content))
	}

	// Check nested list
	nestedList := firstItem.Content[1]
	if nestedList.Type != "bulletList" {
		t.Errorf("Expected nested bulletList, got %s", nestedList.Type)
	}

	if len(nestedList.Content) != 2 {
		t.Errorf("Expected 2 nested items, got %d", len(nestedList.Content))
	}
}

// TestMarkdownToADF_NestedOrderedList tests nested ordered list conversion from Markdown
func TestMarkdownToADF_NestedOrderedList(t *testing.T) {
	markdown := `1. Step 1
  1. Sub-step 1.1
  2. Sub-step 1.2
2. Step 2`

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(doc.Content) != 1 || doc.Content[0].Type != "orderedList" {
		t.Fatalf("Expected orderedList, got %+v", doc.Content[0])
	}

	items := doc.Content[0].Content
	if len(items) != 2 {
		t.Fatalf("Expected 2 parent items, got %d", len(items))
	}

	// Check first parent item has a nested list
	firstItem := items[0]
	if len(firstItem.Content) < 2 {
		t.Fatalf("Expected first item to have paragraph and nested list, got %d content nodes", len(firstItem.Content))
	}

	// Check nested list
	nestedList := firstItem.Content[1]
	if nestedList.Type != "orderedList" {
		t.Errorf("Expected nested orderedList, got %s", nestedList.Type)
	}

	if len(nestedList.Content) != 2 {
		t.Errorf("Expected 2 nested items, got %d", len(nestedList.Content))
	}
}

// TestMarkdownToADF_MixedNestedList tests mixed ordered/unordered nested lists
func TestMarkdownToADF_MixedNestedList(t *testing.T) {
	markdown := `1. First ordered item
  - Nested unordered item
  - Another nested unordered
2. Second ordered item`

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(doc.Content) != 1 || doc.Content[0].Type != "orderedList" {
		t.Fatalf("Expected orderedList, got %+v", doc.Content[0])
	}

	items := doc.Content[0].Content
	firstItem := items[0]

	// Check nested list is bulletList (unordered)
	nestedList := firstItem.Content[1]
	if nestedList.Type != "bulletList" {
		t.Errorf("Expected nested bulletList in ordered list, got %s", nestedList.Type)
	}
}

// TestRoundTrip_NestedLists tests round-trip conversion with nested lists
func TestRoundTrip_NestedLists(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
	}{
		{
			name: "nested bullet list",
			markdown: `- Parent item 1
  - Child item 1
  - Child item 2
- Parent item 2`,
		},
		{
			name: "nested ordered list",
			markdown: `1. Step 1
  1. Sub-step 1.1
  2. Sub-step 1.2
2. Step 2`,
		},
		{
			name: "mixed nested list",
			markdown: `1. First ordered item
  - Nested unordered item
  - Another nested unordered
2. Second ordered item`,
		},
		{
			name: "deeply nested list",
			markdown: `- Level 1
  - Level 2
    - Level 3`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Markdown -> ADF
			adf, err := MarkdownToADF(tt.markdown)
			if err != nil {
				t.Fatalf("MarkdownToADF failed: %v", err)
			}

			// ADF -> Markdown
			result, err := ADFToMarkdown(adf)
			if err != nil {
				t.Fatalf("ADFToMarkdown failed: %v", err)
			}

			if strings.TrimSpace(result) != strings.TrimSpace(tt.markdown) {
				t.Errorf("Round-trip failed:\nExpected:\n%s\n\nGot:\n%s", tt.markdown, result)
			}
		})
	}
}

// TestReverseRoundTrip_NestedLists tests ADF → Markdown → ADF for nested lists
func TestReverseRoundTrip_NestedLists(t *testing.T) {
	tests := []struct {
		name string
		adf  string
	}{
		{
			name: "nested bullet list",
			adf: `{
				"type": "doc",
				"version": 1,
				"content": [
					{
						"type": "bulletList",
						"content": [
							{
								"type": "listItem",
								"content": [
									{
										"type": "paragraph",
										"content": [{"type": "text", "text": "Parent 1"}]
									},
									{
										"type": "bulletList",
										"content": [
											{
												"type": "listItem",
												"content": [
													{
														"type": "paragraph",
														"content": [{"type": "text", "text": "Child 1"}]
													}
												]
											}
										]
									}
								]
							}
						]
					}
				]
			}`,
		},
		{
			name: "nested ordered list",
			adf: `{
				"type": "doc",
				"version": 1,
				"content": [
					{
						"type": "orderedList",
						"content": [
							{
								"type": "listItem",
								"content": [
									{
										"type": "paragraph",
										"content": [{"type": "text", "text": "Step 1"}]
									},
									{
										"type": "orderedList",
										"content": [
											{
												"type": "listItem",
												"content": [
													{
														"type": "paragraph",
														"content": [{"type": "text", "text": "Sub-step 1.1"}]
													}
												]
											}
										]
									}
								]
							}
						]
					}
				]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse original ADF
			var originalDoc ADFDocument
			if err := json.Unmarshal([]byte(tt.adf), &originalDoc); err != nil {
				t.Fatalf("Failed to parse original ADF: %v", err)
			}

			// ADF → Markdown
			markdown, err := ADFToMarkdown(json.RawMessage(tt.adf))
			if err != nil {
				t.Fatalf("ADFToMarkdown failed: %v", err)
			}

			// Markdown → ADF
			resultADF, err := MarkdownToADF(markdown)
			if err != nil {
				t.Fatalf("MarkdownToADF failed: %v", err)
			}

			// Parse result ADF
			var resultDoc ADFDocument
			if err := json.Unmarshal(resultADF, &resultDoc); err != nil {
				t.Fatalf("Failed to parse result ADF: %v", err)
			}

			// Compare structures
			if !compareADFDocuments(originalDoc, resultDoc) {
				origJSON, _ := json.MarshalIndent(originalDoc, "", "  ")
				resultJSON, _ := json.MarshalIndent(resultDoc, "", "  ")
				t.Errorf("ADF round-trip mismatch:\n\nOriginal:\n%s\n\nResult:\n%s",
					string(origJSON), string(resultJSON))
			}
		})
	}
}

// compareADFDocuments compares two ADF documents for structural equality
func compareADFDocuments(a, b ADFDocument) bool {
	if a.Type != b.Type || a.Version != b.Version {
		return false
	}
	if len(a.Content) != len(b.Content) {
		return false
	}
	for i := range a.Content {
		if !compareADFNodes(a.Content[i], b.Content[i]) {
			return false
		}
	}
	return true
}

// compareADFNodes compares two ADF nodes recursively
func compareADFNodes(a, b ADFNode) bool {
	if a.Type != b.Type || a.Text != b.Text {
		return false
	}

	// Compare marks
	if len(a.Marks) != len(b.Marks) {
		return false
	}
	for i := range a.Marks {
		if a.Marks[i].Type != b.Marks[i].Type {
			return false
		}
	}

	// Compare content
	if len(a.Content) != len(b.Content) {
		return false
	}
	for i := range a.Content {
		if !compareADFNodes(a.Content[i], b.Content[i]) {
			return false
		}
	}

	return true
}

// TestRoundTrip_CodeBlock tests round-trip with code blocks
func TestRoundTrip_CodeBlock(t *testing.T) {
	original := "```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```"

	// Markdown -> ADF
	adf, err := MarkdownToADF(original)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	// ADF -> Markdown
	result, err := ADFToMarkdown(adf)
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	if result != original {
		t.Errorf("Round-trip failed:\nExpected:\n%s\nGot:\n%s", original, result)
	}
}

// TestRoundTrip_Comprehensive tests Markdown → ADF → Markdown with ADF validation
func TestRoundTrip_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
	}{
		{"simple text", "Hello, world!"},
		{"bold", "This is **bold** text"},
		{"italic", "This is *italic* text"},
		{"bold and italic", "This is ***bold italic*** text"},
		{"code", "Use `code` here"},
		{"strikethrough", "This is ~~deleted~~ text"},
		{"link", "Visit [Google](https://google.com)"},
		{"heading 1", "# Heading 1"},
		{"heading 2", "## Heading 2"},
		{"heading 3", "### Heading 3"},
		{"bullet list", "- First\n- Second\n- Third"},
		{"ordered list", "1. First\n2. Second\n3. Third"},
		{"nested bullet", "- Parent\n  - Child\n- Parent 2"},
		{"nested ordered", "1. Step 1\n  1. Sub 1\n  2. Sub 2\n2. Step 2"},
		{"mixed nested", "1. Ordered\n  - Unordered\n  - Item\n2. Next"},
		{"code block", "```go\nfunc main() {\n}\n```"},
		{"blockquote", "> This is a quote"},
		{"horizontal rule", "---"},
		{"multiple paragraphs", "First paragraph\n\nSecond paragraph"},
		{"complex formatting", "**Bold** and *italic* with `code` and [link](https://example.com)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Markdown → ADF
			adfJSON, err := MarkdownToADF(tt.markdown)
			if err != nil {
				t.Fatalf("MarkdownToADF failed: %v", err)
			}

			// Validate ADF structure
			var doc ADFDocument
			if err := json.Unmarshal(adfJSON, &doc); err != nil {
				t.Fatalf("Invalid ADF JSON: %v", err)
			}

			// Validate required fields
			if doc.Type != "doc" {
				t.Errorf("Invalid ADF: expected type 'doc', got %q", doc.Type)
			}
			if doc.Version != 1 {
				t.Errorf("Invalid ADF: expected version 1, got %d", doc.Version)
			}
			if doc.Content == nil {
				t.Errorf("Invalid ADF: content is nil")
			}

			// Validate all nodes have required type field
			validateNodes(t, doc.Content)

			// ADF → Markdown
			result, err := ADFToMarkdown(adfJSON)
			if err != nil {
				t.Fatalf("ADFToMarkdown failed: %v", err)
			}

			// Compare - normalize whitespace
			normalizedOriginal := strings.TrimSpace(tt.markdown)
			normalizedResult := strings.TrimSpace(result)

			if normalizedResult != normalizedOriginal {
				t.Errorf("Round-trip mismatch:\nOriginal:\n%s\n\nResult:\n%s\n\nADF:\n%s",
					normalizedOriginal, normalizedResult, string(adfJSON))
			}
		})
	}
}

// validateNodes recursively validates ADF node structure
func validateNodes(t *testing.T, nodes []ADFNode) {
	for i, node := range nodes {
		if node.Type == "" {
			t.Errorf("Node %d has empty type", i)
		}

		// Validate content recursively
		if len(node.Content) > 0 {
			validateNodes(t, node.Content)
		}

		// Validate marks structure
		for j, mark := range node.Marks {
			if mark.Type == "" {
				t.Errorf("Node %d, mark %d has empty type", i, j)
			}
		}
	}
}

// TestADFToMarkdown_Table tests table conversion
func TestADFToMarkdown_Table(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "table",
				"content": [
					{
						"type": "tableRow",
						"content": [
							{
								"type": "tableHeader",
								"content": [
									{
										"type": "paragraph",
										"content": [{"type": "text", "text": "Name"}]
									}
								]
							},
							{
								"type": "tableHeader",
								"content": [
									{
										"type": "paragraph",
										"content": [{"type": "text", "text": "Age"}]
									}
								]
							}
						]
					},
					{
						"type": "tableRow",
						"content": [
							{
								"type": "tableCell",
								"content": [
									{
										"type": "paragraph",
										"content": [{"type": "text", "text": "Alice"}]
									}
								]
							},
							{
								"type": "tableCell",
								"content": [
									{
										"type": "paragraph",
										"content": [{"type": "text", "text": "30"}]
									}
								]
							}
						]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "| Name | Age |\n| --- | --- |\n| Alice | 30 |"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Media tests media node conversion
func TestADFToMarkdown_Media(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "mediaSingle",
				"content": [
					{
						"type": "media",
						"attrs": {
							"url": "https://example.com/image.png",
							"alt": "Example Image"
						}
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "![Example Image](https://example.com/image.png)"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Emoji tests emoji conversion
func TestADFToMarkdown_Emoji(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "Hello "
					},
					{
						"type": "emoji",
						"attrs": {
							"shortName": ":smile:",
							"text": "\u263a"
						}
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "Hello :smile:"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Mention tests mention conversion
func TestADFToMarkdown_Mention(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "Hey "
					},
					{
						"type": "mention",
						"attrs": {
							"id": "user123",
							"text": "john.doe"
						}
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "Hey @john.doe"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Status tests status badge conversion
func TestADFToMarkdown_Status(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "status",
						"attrs": {
							"text": "In Progress",
							"color": "blue"
						}
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "<span data-status=\"blue\">In Progress</span>"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Panel tests panel conversion
func TestADFToMarkdown_Panel(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "panel",
				"attrs": {
					"panelType": "info"
				},
				"content": [
					{
						"type": "paragraph",
						"content": [
							{
								"type": "text",
								"text": "This is important information"
							}
						]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "<div data-panel=\"info\">\n\nThis is important information\n\n</div>"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Expand tests expand/details conversion
func TestADFToMarkdown_Expand(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "expand",
				"attrs": {
					"title": "Click to expand"
				},
				"content": [
					{
						"type": "paragraph",
						"content": [
							{
								"type": "text",
								"text": "Hidden content here"
							}
						]
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "<details>\n<summary>Click to expand</summary>\n\nHidden content here\n</details>"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_InlineCard tests inline card conversion
func TestADFToMarkdown_InlineCard(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "Check out "
					},
					{
						"type": "inlineCard",
						"attrs": {
							"url": "https://example.com",
							"data": {
								"title": "Example Site"
							}
						}
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "Check out [Example Site](https://example.com)"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestADFToMarkdown_Date tests date conversion
func TestADFToMarkdown_Date(t *testing.T) {
	adf := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "text",
						"text": "Due date: "
					},
					{
						"type": "date",
						"attrs": {
							"timestamp": "2025-10-09"
						}
					}
				]
			}
		]
	}`

	result, err := ADFToMarkdown(json.RawMessage(adf))
	if err != nil {
		t.Fatalf("ADFToMarkdown failed: %v", err)
	}

	expected := "Due date: 2025-10-09"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestMarkdownToADF_Table tests table conversion
func TestMarkdownToADF_Table(t *testing.T) {
	markdown := `| Name | Age |
| --- | --- |
| Alice | 30 |
| Bob | 25 |`

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(doc.Content) != 1 || doc.Content[0].Type != "table" {
		t.Errorf("Expected table, got %+v", doc.Content[0])
	}

	rows := doc.Content[0].Content
	if len(rows) != 3 {
		t.Errorf("Expected 3 rows (1 header + 2 data), got %d", len(rows))
	}

	// Check header row
	if rows[0].Type != "tableRow" {
		t.Errorf("Expected tableRow, got %s", rows[0].Type)
	}
	if len(rows[0].Content) != 2 {
		t.Errorf("Expected 2 header cells, got %d", len(rows[0].Content))
	}
	if rows[0].Content[0].Type != "tableHeader" {
		t.Errorf("Expected tableHeader, got %s", rows[0].Content[0].Type)
	}
}

// TestMarkdownToADF_Image tests image conversion
func TestMarkdownToADF_Image(t *testing.T) {
	markdown := "![Example Image](https://example.com/image.png)"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	content := doc.Content[0].Content
	if len(content) < 1 {
		t.Fatalf("Expected at least 1 node, got %d", len(content))
	}

	mediaNode := content[0]
	if mediaNode.Type != "media" {
		t.Errorf("Expected media node, got %s", mediaNode.Type)
	}

	url := mediaNode.Attrs["url"].(string)
	if url != "https://example.com/image.png" {
		t.Errorf("Expected URL 'https://example.com/image.png', got %q", url)
	}

	alt := mediaNode.Attrs["alt"].(string)
	if alt != "Example Image" {
		t.Errorf("Expected alt 'Example Image', got %q", alt)
	}
}

// TestMarkdownToADF_Emoji tests emoji conversion
func TestMarkdownToADF_Emoji(t *testing.T) {
	markdown := "Hello :smile: world"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	content := doc.Content[0].Content
	if len(content) < 3 {
		t.Fatalf("Expected at least 3 nodes, got %d", len(content))
	}

	emojiNode := content[1]
	if emojiNode.Type != "emoji" {
		t.Errorf("Expected emoji node, got %s", emojiNode.Type)
	}

	shortName := emojiNode.Attrs["shortName"].(string)
	if shortName != ":smile:" {
		t.Errorf("Expected shortName ':smile:', got %q", shortName)
	}
}

// TestMarkdownToADF_Mention tests mention conversion
func TestMarkdownToADF_Mention(t *testing.T) {
	markdown := "Hey @john.doe"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	content := doc.Content[0].Content
	if len(content) < 2 {
		t.Fatalf("Expected at least 2 nodes, got %d", len(content))
	}

	mentionNode := content[1]
	if mentionNode.Type != "mention" {
		t.Errorf("Expected mention node, got %s", mentionNode.Type)
	}

	text := mentionNode.Attrs["text"].(string)
	if text != "john.doe" {
		t.Errorf("Expected text 'john.doe', got %q", text)
	}
}

// TestMarkdownToADF_Status tests status badge conversion
func TestMarkdownToADF_Status(t *testing.T) {
	markdown := "<span data-status=\"blue\">In Progress</span>"

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	content := doc.Content[0].Content
	if len(content) < 1 {
		t.Fatalf("Expected at least 1 node, got %d", len(content))
	}

	statusNode := content[0]
	if statusNode.Type != "status" {
		t.Errorf("Expected status node, got %s", statusNode.Type)
	}

	text := statusNode.Attrs["text"].(string)
	if text != "In Progress" {
		t.Errorf("Expected text 'In Progress', got %q", text)
	}

	color := statusNode.Attrs["color"].(string)
	if color != "blue" {
		t.Errorf("Expected color 'blue', got %q", color)
	}
}

// TestMarkdownToADF_Panel tests panel conversion
func TestMarkdownToADF_Panel(t *testing.T) {
	markdown := `<div data-panel="info">

This is important information

</div>`

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(doc.Content) != 1 || doc.Content[0].Type != "panel" {
		t.Errorf("Expected panel, got %+v", doc.Content[0])
	}

	panelType := doc.Content[0].Attrs["panelType"].(string)
	if panelType != "info" {
		t.Errorf("Expected panelType 'info', got %q", panelType)
	}
}

// TestMarkdownToADF_Expand tests expand/details conversion
func TestMarkdownToADF_Expand(t *testing.T) {
	markdown := `<details>
<summary>Click to expand</summary>

Hidden content here
</details>`

	result, err := MarkdownToADF(markdown)
	if err != nil {
		t.Fatalf("MarkdownToADF failed: %v", err)
	}

	var doc ADFDocument
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(doc.Content) != 1 || doc.Content[0].Type != "expand" {
		t.Errorf("Expected expand, got %+v", doc.Content[0])
	}

	title := doc.Content[0].Attrs["title"].(string)
	if title != "Click to expand" {
		t.Errorf("Expected title 'Click to expand', got %q", title)
	}
}
