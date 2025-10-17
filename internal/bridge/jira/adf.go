package jira

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Precompiled regular expressions for inline content parsing (Markdown -> ADF)
var (
	reImage       = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	reLink        = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reInlineCode  = regexp.MustCompile("`([^`]+)`")
	reStatus      = regexp.MustCompile(`<span data-status="([^"]+)">([^<]+)</span>`)
	reTextColor   = regexp.MustCompile(`<span style="color:\s*([^"]+)">([^<]+)</span>`)
	reBorder      = regexp.MustCompile(`<span style="border:\s*1px solid\s*([^"]+)">([^<]+)</span>`)
	reSubscript   = regexp.MustCompile(`<sub>([^<]+)</sub>`)
	reSuperscript = regexp.MustCompile(`<sup>([^<]+)</sup>`)
	reUnderline   = regexp.MustCompile(`<u>([^<]+)</u>`)
	reMention     = regexp.MustCompile(`@([\w.-]+)\b`)
	reEmoji       = regexp.MustCompile(`:([a-z0-9_+-]+):`)
	reBold1       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reBold2       = regexp.MustCompile(`__([^_]+)__`)
	reItalic1     = regexp.MustCompile(`\*([^*]+)\*`)
	reItalic2     = regexp.MustCompile(`_([^_]+)_`)
	reStrike      = regexp.MustCompile(`~~([^~]+)~~`)
)

// Precompiled regular expressions for block-level parsing (Markdown -> ADF)
var (
	// reHeading matches Markdown headings (# through ######)
	// Group 1: hash symbols, Group 2: heading text
	reHeading = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

	// reHorizontalRule matches horizontal rules (---, ***, ___)
	reHorizontalRule = regexp.MustCompile(`^(-{3,}|\*{3,}|_{3,})$`)

	// reOrderedListItem matches ordered list items (1. text, 2. text, etc.)
	// Group 1: leading whitespace, Group 2: item text
	reOrderedListItem = regexp.MustCompile(`^(\s*)\d+\.\s+(.+)$`)

	// reUnorderedListItem matches unordered list items (-, *, or + prefix)
	// Group 1: leading whitespace, Group 2: item text
	reUnorderedListItem = regexp.MustCompile(`^(\s*)[-*+]\s+(.+)$`)

	// reTaskListItem matches task list items with checkboxes (- [ ] or - [x])
	// Group 1: leading whitespace, Group 2: checkbox state (space or x), Group 3: item text
	reTaskListItem = regexp.MustCompile(`^(\s*)-\s+\[([ xX])\]\s+(.+)$`)

	// reOrderedListStart detects the start of an ordered sublist
	reOrderedListStart = regexp.MustCompile(`^\s*\d+\.\s+`)

	// reUnorderedListStart detects the start of an unordered sublist
	reUnorderedListStart = regexp.MustCompile(`^\s*[-*+]\s+`)

	// reBlockElement detects the start of any block element (for paragraph termination)
	// Matches: headings, lists, code blocks, blockquotes
	reBlockElement = regexp.MustCompile(`^(#{1,6}\s|[-*+]\s|\d+\.\s|` + "```" + `|>)`)

	// reTableSeparator matches table separator cells (dashes, colons, spaces)
	reTableSeparator = regexp.MustCompile(`^[\s\-:]+$`)

	// rePanelDiv matches the opening tag of a panel div
	// Group 1: panel type (info, warning, error, etc.)
	rePanelDiv = regexp.MustCompile(`<div data-panel="([^"]+)">`)

	// reSummaryTag matches HTML summary tags
	// Group 1: summary text
	reSummaryTag = regexp.MustCompile(`<summary>([^<]+)</summary>`)
)

// ADFNode represents a node in the Atlassian Document Format tree
type ADFNode struct {
	Type    string                 `json:"type"`
	Content []ADFNode              `json:"content,omitempty"`
	Text    string                 `json:"text,omitempty"`
	Marks   []ADFMark              `json:"marks,omitempty"`
	Attrs   map[string]interface{} `json:"attrs,omitempty"`
}

// ADFMark represents formatting applied to text nodes
type ADFMark struct {
	Type  string                 `json:"type"`
	Attrs map[string]interface{} `json:"attrs,omitempty"`
}

// ADFDocument represents a complete ADF document
type ADFDocument struct {
	Type    string    `json:"type"`
	Version int       `json:"version"`
	Content []ADFNode `json:"content"`
}

// ADFToMarkdown converts an ADF JSON document to Markdown
func ADFToMarkdown(adfJSON json.RawMessage) (string, error) {
	if len(adfJSON) == 0 {
		return "", nil
	}

	// Handle null ADF fields (Jira returns null for empty descriptions/comments)
	if string(adfJSON) == "null" {
		return "", nil
	}

	var doc ADFDocument
	if err := json.Unmarshal(adfJSON, &doc); err != nil {
		return "", fmt.Errorf("failed to parse ADF: %w", err)
	}

	var md strings.Builder
	for i, node := range doc.Content {
		if i > 0 {
			md.WriteString("\n\n")
		}
		md.WriteString(nodeToMarkdown(node, 0))
	}

	return strings.TrimSpace(md.String()), nil
}

// nodeToMarkdown converts a single ADF node to Markdown
func nodeToMarkdown(node ADFNode, depth int) string {
	switch node.Type {
	case "paragraph":
		return paragraphToMarkdown(node)

	case "heading":
		return headingToMarkdown(node)

	case "bulletList":
		return strings.TrimSuffix(listToMarkdown(node, depth, false), "\n")

	case "orderedList":
		return strings.TrimSuffix(listToMarkdown(node, depth, true), "\n")

	case "taskList":
		return taskListToMarkdown(node, depth)

	case "listItem":
		var content strings.Builder
		for i, child := range node.Content {
			if i > 0 {
				// Add newline before nested lists/blocks (but not after paragraph)
				if child.Type == "bulletList" || child.Type == "orderedList" {
					content.WriteString("\n")
				}
			}
			content.WriteString(nodeToMarkdown(child, depth+1))
		}
		return content.String()

	case "codeBlock":
		return codeBlockToMarkdown(node)

	case "blockquote":
		return blockquoteToMarkdown(node)

	case "rule":
		return "---"

	case "hardBreak":
		return "  \n"

	case "table":
		return tableToMarkdown(node)

	case "panel":
		return panelToMarkdown(node)

	case "expand":
		return expandToMarkdown(node, false)

	case "nestedExpand":
		return expandToMarkdown(node, true)

	case "mediaGroup":
		return mediaGroupToMarkdown(node)

	case "mediaSingle":
		return mediaSingleToMarkdown(node)

	case "mediaInline":
		return mediaInlineToMarkdown(node)

	case "emoji":
		return emojiToMarkdown(node)

	case "mention":
		return mentionToMarkdown(node)

	case "date":
		return dateToMarkdown(node)

	case "status":
		return statusToMarkdown(node)

	case "inlineCard":
		return inlineCardToMarkdown(node)

	case "text":
		return applyMarks(node.Text, node.Marks)

	default:
		// Unknown node type - try to extract text from children
		var content strings.Builder
		for _, child := range node.Content {
			content.WriteString(nodeToMarkdown(child, depth))
		}
		return content.String()
	}
}

// paragraphToMarkdown converts a paragraph node to Markdown
func paragraphToMarkdown(node ADFNode) string {
	var content strings.Builder
	for _, child := range node.Content {
		content.WriteString(nodeToMarkdown(child, 0))
	}
	return content.String()
}

// headingToMarkdown converts a heading node to Markdown
func headingToMarkdown(node ADFNode) string {
	level := 1
	if node.Attrs != nil {
		if l, ok := node.Attrs["level"].(float64); ok {
			level = int(l)
		}
	}

	var content strings.Builder
	for _, child := range node.Content {
		content.WriteString(nodeToMarkdown(child, 0))
	}

	return strings.Repeat("#", level) + " " + content.String()
}

// listToMarkdown converts a bullet or ordered list to Markdown
func listToMarkdown(node ADFNode, depth int, ordered bool) string {
	var md strings.Builder
	indent := strings.Repeat("  ", depth)

	for i, item := range node.Content {
		if item.Type == "listItem" {
			prefix := "- "
			if ordered {
				prefix = fmt.Sprintf("%d. ", i+1)
			}

			md.WriteString(indent + prefix)
			md.WriteString(nodeToMarkdown(item, depth))
			md.WriteString("\n")
		}
	}

	return md.String()
}

// taskListToMarkdown converts a task list (checkboxes) to Markdown
func taskListToMarkdown(node ADFNode, depth int) string {
	var md strings.Builder
	indent := strings.Repeat("  ", depth)

	for i, item := range node.Content {
		if item.Type != "taskItem" {
			continue
		}

		// Get checkbox state (TODO or DONE)
		state := "TODO" // default
		if item.Attrs != nil {
			if s, ok := item.Attrs["state"].(string); ok {
				state = s
			}
		}

		// Convert to markdown checkbox
		checkbox := "- [ ] " // unchecked
		if state == "DONE" {
			checkbox = "- [x] " // checked
		}

		md.WriteString(indent + checkbox)

		// Process item content (usually a paragraph)
		for _, child := range item.Content {
			md.WriteString(nodeToMarkdown(child, depth))
		}

		if i < len(node.Content)-1 {
			md.WriteString("\n")
		}
	}

	return md.String()
}

// codeBlockToMarkdown converts a code block to Markdown
func codeBlockToMarkdown(node ADFNode) string {
	language := ""
	if node.Attrs != nil {
		if lang, ok := node.Attrs["language"].(string); ok {
			language = lang
		}
	}

	var code strings.Builder
	for _, child := range node.Content {
		if child.Type == "text" {
			code.WriteString(child.Text)
		}
	}

	return "```" + language + "\n" + code.String() + "\n```"
}

// blockquoteToMarkdown converts a blockquote to Markdown
func blockquoteToMarkdown(node ADFNode) string {
	var content strings.Builder
	for _, child := range node.Content {
		content.WriteString(nodeToMarkdown(child, 0))
	}

	// Prefix each line with "> "
	lines := strings.Split(strings.TrimSpace(content.String()), "\n")
	for i, line := range lines {
		lines[i] = "> " + line
	}
	return strings.Join(lines, "\n")
}

// applyMarks applies formatting marks to text
func applyMarks(text string, marks []ADFMark) string {
	if len(marks) == 0 {
		return text
	}

	// Track which marks to apply (order matters for nesting)
	var strong, em, code, strike, underline bool
	var link, subsup, textColor, border *ADFMark

	for _, mark := range marks {
		switch mark.Type {
		case "strong":
			strong = true
		case "em":
			em = true
		case "code":
			code = true
		case "strike":
			strike = true
		case "underline":
			underline = true
		case "link":
			link = &mark
		case "subsup":
			subsup = &mark
		case "textColor":
			textColor = &mark
		case "border":
			border = &mark
		}
	}

	// Apply marks in order (innermost to outermost)
	result := text

	if code {
		result = "`" + result + "`"
	}

	// Subscript/superscript (HTML format)
	if subsup != nil {
		if typ, ok := subsup.Attrs["type"].(string); ok {
			if typ == "sub" {
				result = "<sub>" + result + "</sub>"
			} else if typ == "sup" {
				result = "<sup>" + result + "</sup>"
			}
		}
	}

	if strike {
		result = "~~" + result + "~~"
	}
	if underline {
		// Markdown doesn't have native underline, use HTML
		result = "<u>" + result + "</u>"
	}

	// Text color (HTML inline style)
	if textColor != nil {
		if color, ok := textColor.Attrs["color"].(string); ok {
			result = fmt.Sprintf("<span style=\"color: %s\">%s</span>", color, result)
		}
	}

	// Border (HTML inline style)
	if border != nil {
		if color, ok := border.Attrs["color"].(string); ok {
			result = fmt.Sprintf("<span style=\"border: 1px solid %s\">%s</span>", color, result)
		}
	}

	if em {
		result = "*" + result + "*"
	}
	if strong {
		result = "**" + result + "**"
	}
	if link != nil {
		if href, ok := link.Attrs["href"].(string); ok {
			result = "[" + result + "](" + href + ")"
		}
	}

	return result
}

// tableToMarkdown converts a table to Markdown
func tableToMarkdown(node ADFNode) string {
	var rows [][]string
	var headerRow []string
	hasHeader := false

	// Extract all rows
	for _, rowNode := range node.Content {
		if rowNode.Type != "tableRow" {
			continue
		}

		var cells []string
		for _, cellNode := range rowNode.Content {
			if cellNode.Type == "tableHeader" {
				hasHeader = true
			}

			var cellContent strings.Builder
			for _, contentNode := range cellNode.Content {
				cellContent.WriteString(nodeToMarkdown(contentNode, 0))
			}

			// Replace newlines with spaces in table cells
			text := strings.ReplaceAll(strings.TrimSpace(cellContent.String()), "\n", " ")
			cells = append(cells, text)
		}

		if len(cells) > 0 {
			if hasHeader && len(headerRow) == 0 {
				headerRow = cells
				hasHeader = false // Reset for subsequent rows
			} else {
				rows = append(rows, cells)
			}
		}
	}

	// Build markdown table
	var md strings.Builder

	// Write header if we have one
	if len(headerRow) > 0 {
		md.WriteString("| ")
		md.WriteString(strings.Join(headerRow, " | "))
		md.WriteString(" |\n")

		// Write separator
		md.WriteString("|")
		for range headerRow {
			md.WriteString(" --- |")
		}
		md.WriteString("\n")
	}

	// Write data rows
	for _, row := range rows {
		md.WriteString("| ")
		md.WriteString(strings.Join(row, " | "))
		md.WriteString(" |\n")
	}

	return strings.TrimSuffix(md.String(), "\n")
}

// panelToMarkdown converts a panel (colored info box) to Markdown
func panelToMarkdown(node ADFNode) string {
	panelType := "info"
	if node.Attrs != nil {
		if pt, ok := node.Attrs["panelType"].(string); ok {
			panelType = pt
		}
	}

	var content strings.Builder
	for _, child := range node.Content {
		content.WriteString(nodeToMarkdown(child, 0))
	}

	// Use HTML div with data attribute
	return fmt.Sprintf("<div data-panel=\"%s\">\n\n%s\n\n</div>", panelType, strings.TrimSpace(content.String()))
}

// expandToMarkdown converts expand/nestedExpand to Markdown
func expandToMarkdown(node ADFNode, nested bool) string {
	title := "Details"
	if node.Attrs != nil {
		if t, ok := node.Attrs["title"].(string); ok {
			title = t
		}
	}

	var content strings.Builder
	for _, child := range node.Content {
		content.WriteString(nodeToMarkdown(child, 0))
	}

	// Use HTML details/summary tags (widely supported in Markdown)
	openTag := "details"
	if nested {
		openTag = `details class="nested"`
	}
	return fmt.Sprintf("<%s>\n<summary>%s</summary>\n\n%s\n</details>",
		openTag, title, strings.TrimSpace(content.String()))
}

// mediaGroupToMarkdown converts a media group to Markdown
func mediaGroupToMarkdown(node ADFNode) string {
	var images []string
	for _, child := range node.Content {
		if child.Type == "media" {
			images = append(images, mediaToMarkdown(child))
		}
	}
	return strings.Join(images, " ")
}

// mediaSingleToMarkdown converts a single media element to Markdown
func mediaSingleToMarkdown(node ADFNode) string {
	for _, child := range node.Content {
		if child.Type == "media" {
			return mediaToMarkdown(child)
		}
	}
	return ""
}

// mediaToMarkdown converts a media node to Markdown image syntax
func mediaToMarkdown(node ADFNode) string {
	if node.Attrs == nil {
		return ""
	}

	alt := ""
	if a, ok := node.Attrs["alt"].(string); ok {
		alt = a
	}

	// Try different URL attributes
	var url string
	if u, ok := node.Attrs["url"].(string); ok {
		url = u
	} else if u, ok := node.Attrs["src"].(string); ok {
		url = u
	} else if id, ok := node.Attrs["id"].(string); ok {
		// For attachments, use attachment ID
		url = "attachment:" + id
	}

	if url == "" {
		return ""
	}

	return fmt.Sprintf("![%s](%s)", alt, url)
}

// mediaInline handles inline media similarly
func mediaInlineToMarkdown(node ADFNode) string {
	return mediaToMarkdown(node)
}

// emojiToMarkdown converts emoji node to text
func emojiToMarkdown(node ADFNode) string {
	if node.Attrs == nil {
		return ""
	}

	// Try to get the emoji shortcode
	if shortName, ok := node.Attrs["shortName"].(string); ok {
		return shortName
	}

	// Or the emoji text itself
	if text, ok := node.Attrs["text"].(string); ok {
		return text
	}

	return ""
}

// mentionToMarkdown converts a user mention to @username format
func mentionToMarkdown(node ADFNode) string {
	if node.Attrs == nil {
		return ""
	}

	// Try to get display name or username
	if text, ok := node.Attrs["text"].(string); ok {
		return "@" + text
	}

	if id, ok := node.Attrs["id"].(string); ok {
		return "@" + id
	}

	return "@user"
}

// dateToMarkdown converts a date node to ISO format
func dateToMarkdown(node ADFNode) string {
	if node.Attrs == nil {
		return ""
	}

	if timestamp, ok := node.Attrs["timestamp"].(string); ok {
		return timestamp
	}

	return ""
}

// statusToMarkdown converts a status badge to HTML format
func statusToMarkdown(node ADFNode) string {
	if node.Attrs == nil {
		return ""
	}

	text := "Status"
	if t, ok := node.Attrs["text"].(string); ok {
		text = t
	}

	color := "neutral"
	if c, ok := node.Attrs["color"].(string); ok {
		color = c
	}

	return fmt.Sprintf("<span data-status=\"%s\">%s</span>", color, text)
}

// inlineCardToMarkdown converts smart links to regular links
func inlineCardToMarkdown(node ADFNode) string {
	if node.Attrs == nil {
		return ""
	}

	url := ""
	if u, ok := node.Attrs["url"].(string); ok {
		url = u
	}

	if url == "" {
		return ""
	}

	// Try to get title if available
	title := url
	if data, ok := node.Attrs["data"].(map[string]interface{}); ok {
		if t, ok := data["title"].(string); ok {
			title = t
		}
	}

	return fmt.Sprintf("[%s](%s)", title, url)
}

// MarkdownToADF converts Markdown text to ADF JSON
func MarkdownToADF(markdown string) (json.RawMessage, error) {
	if markdown == "" {
		return json.RawMessage(`{"type":"doc","version":1,"content":[]}`), nil
	}

	doc := ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{},
	}

	lines := strings.Split(markdown, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		// Code block
		if strings.HasPrefix(line, "```") {
			node, consumed := parseCodeBlock(lines, i)
			doc.Content = append(doc.Content, node)
			i += consumed
			continue
		}

		// Heading
		if match := reHeading.FindStringSubmatch(line); match != nil {
			doc.Content = append(doc.Content, ADFNode{
				Type: "heading",
				Attrs: map[string]interface{}{
					"level": len(match[1]),
				},
				Content: parseInlineContent(match[2]),
			})
			i++
			continue
		}

		// Horizontal rule
		if reHorizontalRule.MatchString(strings.TrimSpace(line)) {
			doc.Content = append(doc.Content, ADFNode{
				Type: "rule",
			})
			i++
			continue
		}

		// Table (starts with |)
		if strings.HasPrefix(strings.TrimSpace(line), "|") {
			node, consumed := parseTable(lines, i)
			if node.Type != "" {
				doc.Content = append(doc.Content, node)
				i += consumed
				continue
			}
		}

		// Panel (HTML format)
		if strings.HasPrefix(line, "<div data-panel=") {
			node, consumed := parsePanel(lines, i)
			doc.Content = append(doc.Content, node)
			i += consumed
			continue
		}

		// Expand (HTML details tag)
		if strings.HasPrefix(line, "<details") {
			node, consumed := parseExpand(lines, i)
			doc.Content = append(doc.Content, node)
			i += consumed
			continue
		}

		// Blockquote
		if strings.HasPrefix(line, "> ") {
			node, consumed := parseBlockquote(lines, i)
			doc.Content = append(doc.Content, node)
			i += consumed
			continue
		}

		// Task list (must check BEFORE unordered list since both start with -)
		if reTaskListItem.MatchString(line) {
			node, consumed := parseTaskList(lines, i)
			doc.Content = append(doc.Content, node)
			i += consumed
			continue
		}

		// Unordered list
		if reUnorderedListStart.MatchString(line) {
			node, consumed := parseList(lines, i, false)
			doc.Content = append(doc.Content, node)
			i += consumed
			continue
		}

		// Ordered list
		if reOrderedListStart.MatchString(line) {
			node, consumed := parseList(lines, i, true)
			doc.Content = append(doc.Content, node)
			i += consumed
			continue
		}

		// Regular paragraph
		node, consumed := parseParagraph(lines, i)
		doc.Content = append(doc.Content, node)
		i += consumed
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ADF: %w", err)
	}

	return json.RawMessage(data), nil
}

// parseCodeBlock parses a fenced code block
func parseCodeBlock(lines []string, start int) (ADFNode, int) {
	firstLine := lines[start]
	language := strings.TrimSpace(strings.TrimPrefix(firstLine, "```"))

	var code strings.Builder
	i := start + 1
	closed := false

	for i < len(lines) {
		if strings.HasPrefix(lines[i], "```") {
			closed = true
			break
		}
		if i > start+1 {
			code.WriteString("\n")
		}
		code.WriteString(lines[i])
		i++
	}

	node := ADFNode{
		Type: "codeBlock",
		Content: []ADFNode{
			{Type: "text", Text: code.String()},
		},
	}

	if language != "" {
		node.Attrs = map[string]interface{}{
			"language": language,
		}
	}

	// If closed, consume the closing fence line; otherwise don't over-consume
	if closed {
		return node, i - start + 1
	}
	return node, i - start
}

// parseBlockquote parses a blockquote
func parseBlockquote(lines []string, start int) (ADFNode, int) {
	var content strings.Builder
	i := start

	for i < len(lines) && strings.HasPrefix(lines[i], "> ") {
		if content.Len() > 0 {
			content.WriteString("\n")
		}
		content.WriteString(strings.TrimPrefix(lines[i], "> "))
		i++
	}

	return ADFNode{
		Type: "blockquote",
		Content: []ADFNode{
			{
				Type:    "paragraph",
				Content: parseInlineContent(content.String()),
			},
		},
	}, i - start
}

// parseList parses an ordered or unordered list with support for nested lists
func parseList(lines []string, start int, ordered bool) (ADFNode, int) {
	listType := "bulletList"
	attrs := map[string]interface{}{}
	if ordered {
		listType = "orderedList"
		attrs["order"] = 1
	}

	var items []ADFNode
	i := start
	baseIndent := getIndent(lines[start])

	for i < len(lines) {
		line := lines[i]
		currentIndent := getIndent(line)

		// Stop if we hit a line with less indent
		if currentIndent < baseIndent {
			break
		}

		// Check if this is a list item at the current level
		var pattern *regexp.Regexp
		if ordered {
			pattern = reOrderedListItem
		} else {
			pattern = reUnorderedListItem
		}

		match := pattern.FindStringSubmatch(line)
		if match == nil {
			// Not a list item, might be continuation or end of list
			break
		}

		if currentIndent == baseIndent {
			// Same level list item - create it
			listItem := ADFNode{
				Type: "listItem",
				Content: []ADFNode{
					{
						Type:    "paragraph",
						Content: parseInlineContent(match[2]),
					},
				},
			}
			items = append(items, listItem)
			i++

			// Check for nested lists
			if i < len(lines) {
				nextIndent := getIndent(lines[i])
				if nextIndent > baseIndent {
					// Detect if it's an ordered or unordered sublist
					isOrderedSublist := reOrderedListStart.MatchString(lines[i])
					isUnorderedSublist := reUnorderedListStart.MatchString(lines[i])

					if isOrderedSublist || isUnorderedSublist {
						// Parse nested list recursively
						sublist, consumed := parseList(lines, i, isOrderedSublist)
						i += consumed

						// Append sublist to the last list item's content
						if len(items) > 0 {
							items[len(items)-1].Content = append(items[len(items)-1].Content, sublist)
						}
					}
				}
			}
		} else if currentIndent > baseIndent {
			// This is a nested item but we're at the wrong level - break to let recursion handle it
			break
		} else {
			i++
		}
	}

	node := ADFNode{
		Type:    listType,
		Content: items,
	}
	if len(attrs) > 0 {
		node.Attrs = attrs
	}
	return node, i - start
}

// parseTaskList parses a task list (checkboxes) from Markdown
func parseTaskList(lines []string, start int) (ADFNode, int) {
	taskList := ADFNode{
		Type:    "taskList",
		Content: []ADFNode{},
	}

	i := start
	baseIndent := getIndent(lines[start])

	for i < len(lines) {
		line := lines[i]
		currentIndent := getIndent(line)

		// Stop if we hit a line with less indent
		if currentIndent < baseIndent {
			break
		}

		// Check if still a task list item
		matches := reTaskListItem.FindStringSubmatch(line)
		if matches == nil {
			break
		}

		if currentIndent == baseIndent {
			// Same level task item
			checkbox := matches[2]
			itemText := matches[3]

			// Determine state
			state := "TODO"
			if checkbox == "x" || checkbox == "X" {
				state = "DONE"
			}

			// Create task item with paragraph content
			taskItem := ADFNode{
				Type: "taskItem",
				Content: []ADFNode{
					{
						Type:    "paragraph",
						Content: parseInlineContent(itemText),
					},
				},
				Attrs: map[string]interface{}{
					"state": state,
				},
			}

			taskList.Content = append(taskList.Content, taskItem)
			i++
		} else {
			// Different indentation level, stop here
			break
		}
	}

	return taskList, i - start
}

// parseParagraph parses a regular paragraph
func parseParagraph(lines []string, start int) (ADFNode, int) {
	var text strings.Builder
	i := start

	text.WriteString(lines[i])
	i++

	// Continue until we hit an empty line or special formatting
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			break
		}
		// Stop at headings, lists, code blocks, etc.
		if reBlockElement.MatchString(line) {
			break
		}

		// Handle hard breaks (two spaces at end of line)
		if strings.HasSuffix(text.String(), "  ") {
			text.WriteString("\n")
		} else {
			text.WriteString(" ")
		}
		text.WriteString(line)
		i++
	}

	return ADFNode{
		Type:    "paragraph",
		Content: parseInlineContent(text.String()),
	}, i - start
}

// parseInlineContent parses inline formatting (bold, italic, links, etc.)
func parseInlineContent(text string) []ADFNode {
	if text == "" {
		return []ADFNode{}
	}

	var nodes []ADFNode
	remaining := text

	// Order matters: process more specific patterns first
	// Use precompiled regexes for performance
	patterns := []struct {
		regex *regexp.Regexp
		type_ string
		mark  string
	}{
		// Images: ![alt](url) - must come before links
		{reImage, "image", ""},
		// Links: [text](url) - must come before other bracket patterns
		{reLink, "link", ""},
		// Inline code: `code`
		{reInlineCode, "code", "code"},
		// HTML status: <span data-status="color">text</span>
		{reStatus, "status", ""},
		// HTML colored text: <span style="color: #xxx">text</span>
		{reTextColor, "textColor", ""},
		// HTML bordered text: <span style="border: 1px solid #xxx">text</span>
		{reBorder, "border", ""},
		// HTML subscript: <sub>text</sub>
		{reSubscript, "subsup", "sub"},
		// HTML superscript: <sup>text</sup>
		{reSuperscript, "subsup", "sup"},
		// HTML underline: <u>text</u>
		{reUnderline, "underline", "underline"},
		// Mentions: @username (word boundary at end)
		{reMention, "mention", ""},
		// Emoji: :emoji_name:
		{reEmoji, "emoji", ""},
		// Bold: **text** or __text__
		{reBold1, "strong", "strong"},
		{reBold2, "strong", "strong"},
		// Italic: *text* or _text_
		{reItalic1, "em", "em"},
		{reItalic2, "em", "em"},
		// Strikethrough: ~~text~~
		{reStrike, "strike", "strike"},
	}

	for remaining != "" {
		// Find the earliest match
		earliestPos := len(remaining)
		var earliestMatch []string
		var earliestPattern *struct {
			regex *regexp.Regexp
			type_ string
			mark  string
		}

		for i := range patterns {
			match := patterns[i].regex.FindStringSubmatchIndex(remaining)
			if match != nil && match[0] < earliestPos {
				earliestPos = match[0]
				earliestMatch = []string{
					remaining[match[0]:match[1]], // Full match
					remaining[match[2]:match[3]], // Captured group 1
				}
				if len(match) > 4 {
					earliestMatch = append(earliestMatch, remaining[match[4]:match[5]]) // Captured group 2 (for links)
				}
				earliestPattern = &patterns[i]
			}
		}

		if earliestPattern == nil {
			// No more matches, add remaining text
			if remaining != "" {
				nodes = append(nodes, ADFNode{
					Type: "text",
					Text: remaining,
				})
			}
			break
		}

		// Add text before match
		if earliestPos > 0 {
			nodes = append(nodes, ADFNode{
				Type: "text",
				Text: remaining[:earliestPos],
			})
		}

		// Add formatted text
		if earliestPattern.type_ == "image" {
			// Images become media nodes
			alt := earliestMatch[1]
			url := earliestMatch[2]

			// Determine if it's an attachment or external URL
			if strings.HasPrefix(url, "attachment:") {
				attachmentID := strings.TrimPrefix(url, "attachment:")
				nodes = append(nodes, ADFNode{
					Type: "media",
					Attrs: map[string]interface{}{
						"id":   attachmentID,
						"type": "file",
						"alt":  alt,
					},
				})
			} else {
				nodes = append(nodes, ADFNode{
					Type: "media",
					Attrs: map[string]interface{}{
						"url":  url,
						"type": "external",
						"alt":  alt,
					},
				})
			}
		} else if earliestPattern.type_ == "link" {
			// Special handling for links
			linkText := earliestMatch[1]
			linkURL := earliestMatch[2]
			nodes = append(nodes, ADFNode{
				Type: "text",
				Text: linkText,
				Marks: []ADFMark{
					{
						Type: "link",
						Attrs: map[string]interface{}{
							"href": linkURL,
						},
					},
				},
			})
		} else if earliestPattern.type_ == "status" {
			// Status badge
			color := earliestMatch[1]
			text := earliestMatch[2]
			nodes = append(nodes, ADFNode{
				Type: "status",
				Attrs: map[string]interface{}{
					"text":  text,
					"color": color,
				},
			})
		} else if earliestPattern.type_ == "mention" {
			// User mention
			username := earliestMatch[1]
			nodes = append(nodes, ADFNode{
				Type: "mention",
				Attrs: map[string]interface{}{
					"text": username,
					"id":   username,
				},
			})
		} else if earliestPattern.type_ == "emoji" {
			// Emoji
			shortName := earliestMatch[1]
			nodes = append(nodes, ADFNode{
				Type: "emoji",
				Attrs: map[string]interface{}{
					"shortName": ":" + shortName + ":",
					"text":      ":" + shortName + ":",
				},
			})
		} else if earliestPattern.type_ == "textColor" {
			// Text color: group 1 is color, group 2 is text
			nodes = append(nodes, ADFNode{
				Type: "text",
				Text: earliestMatch[2],
				Marks: []ADFMark{
					{
						Type: "textColor",
						Attrs: map[string]interface{}{
							"color": earliestMatch[1],
						},
					},
				},
			})
		} else if earliestPattern.type_ == "border" {
			// Border: group 1 is color, group 2 is text
			nodes = append(nodes, ADFNode{
				Type: "text",
				Text: earliestMatch[2],
				Marks: []ADFMark{
					{
						Type: "border",
						Attrs: map[string]interface{}{
							"color": earliestMatch[1],
						},
					},
				},
			})
		} else if earliestPattern.type_ == "subsup" {
			// Subsup: mark contains "sub" or "sup" type
			nodes = append(nodes, ADFNode{
				Type: "text",
				Text: earliestMatch[1],
				Marks: []ADFMark{
					{
						Type: "subsup",
						Attrs: map[string]interface{}{
							"type": earliestPattern.mark,
						},
					},
				},
			})
		} else {
			// Regular mark
			nodes = append(nodes, ADFNode{
				Type: "text",
				Text: earliestMatch[1],
				Marks: []ADFMark{
					{Type: earliestPattern.mark},
				},
			})
		}

		// Continue with remaining text
		remaining = remaining[earliestPos+len(earliestMatch[0]):]
	}

	return nodes
}

// parseTable parses a Markdown table
func parseTable(lines []string, start int) (ADFNode, int) {
	var rows [][]string
	i := start

	// Read all table rows
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "|") {
			break
		}

		// Skip separator line (|---|---|)
		// Check if this is a separator line by looking for dashes between pipes
		isSeparator := true
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for _, cell := range cells {
			trimmed := strings.TrimSpace(cell)
			// Separator cells should only contain dashes, colons, and spaces
			if trimmed != "" && !reTableSeparator.MatchString(trimmed) {
				isSeparator = false
				break
			}
		}

		if isSeparator && len(cells) > 0 {
			i++
			continue
		}

		// Parse cells
		parsedCells := []string{}
		for _, cell := range cells {
			parsedCells = append(parsedCells, strings.TrimSpace(cell))
		}

		rows = append(rows, parsedCells)
		i++
	}

	if len(rows) == 0 {
		return ADFNode{}, 0
	}

	// Build ADF table
	table := ADFNode{
		Type:    "table",
		Content: []ADFNode{},
	}

	// First row is header
	if len(rows) > 0 {
		headerCells := []ADFNode{}
		for _, cell := range rows[0] {
			headerCells = append(headerCells, ADFNode{
				Type: "tableHeader",
				Content: []ADFNode{
					{
						Type:    "paragraph",
						Content: parseInlineContent(cell),
					},
				},
			})
		}
		table.Content = append(table.Content, ADFNode{
			Type:    "tableRow",
			Content: headerCells,
		})
	}

	// Remaining rows are data
	for _, row := range rows[1:] {
		dataCells := []ADFNode{}
		for _, cell := range row {
			dataCells = append(dataCells, ADFNode{
				Type: "tableCell",
				Content: []ADFNode{
					{
						Type:    "paragraph",
						Content: parseInlineContent(cell),
					},
				},
			})
		}
		table.Content = append(table.Content, ADFNode{
			Type:    "tableRow",
			Content: dataCells,
		})
	}

	return table, i - start
}

// parsePanel parses an HTML panel
func parsePanel(lines []string, start int) (ADFNode, int) {
	firstLine := lines[start]
	match := rePanelDiv.FindStringSubmatch(firstLine)
	if match == nil {
		return ADFNode{}, 0
	}

	panelType := match[1]
	var content strings.Builder
	i := start + 1

	for i < len(lines) {
		if strings.HasPrefix(lines[i], "</div>") {
			break
		}
		if content.Len() > 0 {
			content.WriteString("\n")
		}
		content.WriteString(lines[i])
		i++
	}

	// Parse the content as markdown
	contentADF, err := MarkdownToADF(strings.TrimSpace(content.String()))
	if err != nil {
		// If markdown parsing fails, return empty panel
		return ADFNode{
			Type: "panel",
			Attrs: map[string]interface{}{
				"panelType": panelType,
			},
			Content: []ADFNode{},
		}, i - start + 1
	}

	var doc ADFDocument
	if err := json.Unmarshal(contentADF, &doc); err != nil {
		// If ADF parsing fails, return empty panel
		return ADFNode{
			Type: "panel",
			Attrs: map[string]interface{}{
				"panelType": panelType,
			},
			Content: []ADFNode{},
		}, i - start + 1
	}

	return ADFNode{
		Type: "panel",
		Attrs: map[string]interface{}{
			"panelType": panelType,
		},
		Content: doc.Content,
	}, i - start + 1
}

// parseExpand parses HTML details/summary tags
func parseExpand(lines []string, start int) (ADFNode, int) {
	firstLine := lines[start]
	nested := strings.Contains(firstLine, "nested")

	// Extract title from summary tag
	title := "Details"
	i := start + 1
	for i < len(lines) && !strings.Contains(lines[i], "<summary>") {
		i++
	}

	if i < len(lines) {
		summaryLine := lines[i]
		match := reSummaryTag.FindStringSubmatch(summaryLine)
		if match != nil {
			title = match[1]
		}
		i++
	}

	// Read content until closing tag
	var content strings.Builder
	for i < len(lines) {
		if strings.HasPrefix(lines[i], "</details>") {
			break
		}
		if content.Len() > 0 {
			content.WriteString("\n")
		}
		content.WriteString(lines[i])
		i++
	}

	// Parse the content as markdown
	contentADF, err := MarkdownToADF(strings.TrimSpace(content.String()))
	nodeType := "expand"
	if nested {
		nodeType = "nestedExpand"
	}

	if err != nil {
		// If markdown parsing fails, return empty expand
		return ADFNode{
			Type: nodeType,
			Attrs: map[string]interface{}{
				"title": title,
			},
			Content: []ADFNode{},
		}, i - start + 1
	}

	var doc ADFDocument
	if err := json.Unmarshal(contentADF, &doc); err != nil {
		// If ADF parsing fails, return empty expand
		return ADFNode{
			Type: nodeType,
			Attrs: map[string]interface{}{
				"title": title,
			},
			Content: []ADFNode{},
		}, i - start + 1
	}

	return ADFNode{
		Type: nodeType,
		Attrs: map[string]interface{}{
			"title": title,
		},
		Content: doc.Content,
	}, i - start + 1
}

// getIndent returns the number of spaces at the start of a line
func getIndent(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}
