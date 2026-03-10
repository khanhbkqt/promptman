---
name: vibepm-content-format
description: |
  How to write content (documents, comments, issue descriptions) using Markdown
  so it renders correctly in VibePM. Use when writing any content fields via MCP tools.
---

# Content Formatting for VibePM

VibePM renders content differently depending on the content type:

| Field | Renderer | Format |
|-------|----------|--------|
| Issue `description` | ReactMarkdown | Markdown string |
| Comment `content` | Tiptap editor | Markdown → converted to Tiptap JSON |
| Document `content` | Tiptap editor | Markdown → converted to Tiptap JSON |

**Write all content fields using Markdown.** Full Markdown syntax is supported
across all three renderers.

---

## Supported Markdown Syntax

### Headings
```markdown
# Heading 1
## Heading 2
### Heading 3
```
> Note: Documents support H1–H3 in Tiptap. Issue descriptions support H1–H6 via ReactMarkdown.

### Inline formatting
```markdown
**bold text**
*italic text*
`inline code`
~~strikethrough~~
```

### Lists
```markdown
- Bullet item one
- Bullet item two
  - Nested bullet

1. Ordered item one
2. Ordered item two
```

### Task lists (documents and comments only)
```markdown
- [ ] Unchecked task
- [x] Completed task
```

### Blockquotes
```markdown
> This is a blockquote.
> Multiple lines continue the same block.
```

### Code blocks (with optional language)
````markdown
```typescript
const msg = "Hello world";
console.log(msg);
```
````

### Horizontal rule
```markdown
---
```

---

## Example: Issue Description (ReactMarkdown)

```markdown
## Overview

Implement full-text search for issues and documents.

## Acceptance Criteria

- Search box visible in the top navigation bar
- Results include issues and documents, linked directly
- Empty state shown when no results found

## Out of Scope

- Real-time search-as-you-type
- Cross-project search
```

## Example: Document Content (Tiptap)

```markdown
# Search API Design

## Endpoint

`POST /api/v1/search`

Requires Bearer token in the `Authorization` header.

## Request Body

- `q` — search query string (required)
- `projectId` — limit to one project (required)
- `type` — `issue | document | all` (optional)
- `limit` — max results, default 20, max 50 (optional)

## Response

Returns results ordered by relevance. Each result includes:

- `id` — resource UUID
- `type` — `issue` or `document`
- `title` — display title
- `excerpt` — matching text snippet (up to 200 chars)
```

## Example: Comment (Tiptap)

```markdown
Reviewed the implementation. A few issues to fix:

- The SQL index on `ts_vector` is missing — add it in the migration
- Empty query string returns **500** instead of **400**
- Pagination tested and working correctly ✅

Please fix the two issues above and re-request review.
```
