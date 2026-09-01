# Pagination Helper (`/pagination`)

Production-ready, bidirectional keyset (cursor) and traditional offset pagination for Go REST APIs. Zero dependencies, type-safe generic responses, whitelisted column filtering, and protection against large query DoS attacks.

## 🚀 Key Capabilities

- **Bidirectional Dynamic Keyset (Cursor):** Supports dynamic multi-column sorting (e.g., `sort_by=status,created_at`) with automatic SQL operator inversion (`<` / `>`) and tie-breaker safety.
- **Traditional Offset-Based:** Sanitized `page` and `limit`, safe SQL `Offset()` calculation, and zero ghost-page bugs.
- **Unified Mode:** Supports both pagination styles in a single endpoint with automatic detection.
- **Smart Filtering & Search:** Parses grouped filters (`?filter=status:active,pending`) and direct query parameters (`?status=active`) with SQL injection protection via `allowedFilters` whitelisting.
- **Generic Responses:** Standardized `ListResponse[T]` and `CursorListResponse[T]` with RFC-compliant metadata.

---

## 📖 Quickstart & Patterns

### 1. Traditional Offset Pagination

Ideal for admin dashboards and tables requiring jump-to-page navigation.

```go
import "github.com/Jkenyut/nvx-go-helper/pagination"

// 1. In Handler: Extract pagination + filters safely
allowedFilters := map[string]string{
	"status": "users.status",
	"role":   "roles.name",
}
req := pagination.BindOffsetFilterRequest(r, allowedFilters)

// 2. In Repository: Calculate safe SQL offset and limit
totalCount := 150
pageData := pagination.NewFromInt(req.Page, req.Limit, totalCount)

// db.Limit(pageData.Limit).Offset(pageData.Offset())
// if req.HasFilter("users.status") { ... }

// 3. In Response: Wrap into standardized generic response
resp := pagination.NewListResponse(users, pageData)
response.OK(ctx, "success", resp)
```

**JSON Output:**
```json
{
  "items": [...],
  "pagination": {
    "page": 2,
    "limit": 10,
    "total": 150,
    "total_pages": 15,
    "has_next": true,
    "has_prev": true,
    "next_page": 3,
    "prev_page": 1
  }
}
```

---

### 2. Dynamic Keyset Cursor Pagination

Ideal for infinite scrolling, mobile feeds, and high-performance, high-volume datasets (eliminates `OFFSET` bottlenecks).

```go
import (
	"github.com/Jkenyut/nvx-go-helper/pagination"
	"github.com/Jkenyut/nvx-go-helper/sliceutil"
)

// 1. In Handler: Binds query params and automatically decodes cursor
allowedColumns := map[string]string{
	"name":       "user_name",
	"created_at": "user_created_at",
}
req := pagination.BindCursorFilterRequest(r, allowedColumns)

// 2. In Repository: Prepare SQL sort and keyset WHERE conditions
sortRes := pagination.PrepareDynamicSort(pagination.DynamicSortParams{
	SortBy:         req.SortBy,
	SortType:       req.SortType,
	Direction:      req.Direction,
	AllowedColumns: allowedColumns,
	UniqueColumn:   "user_id", // Mandatory tie-breaker to prevent skipped rows
	UniqueSortType: "DESC",
})

var sqlWhere string
var sqlArgs []any
if len(req.CursorValues) > 0 {
	sqlWhere, sqlArgs = pagination.BuildDynamicKeyset(sortRes.Columns, sortRes.Operators, req.CursorValues)
}

// Execute query:
// q = q.Where(sqlWhere, sqlArgs...).Limit(req.Limit)
// for _, order := range sortRes.OrderStrs { q = q.OrderBy(order) }

// 3. In Service: Handle backward reversal & generate next/prev cursors
if req.Direction == "prev" {
	users = sliceutil.Reverse(users)
}

cursorMeta := pagination.GenerateBidirectionalCursor(
	users,
	req.Limit,
	req.Direction,
	req.Cursor,
	func(u User) []any {
		// Extract values matching dynamic sort columns + unique column
		return []any{u.Name, u.CreatedAt, u.ID}
	},
)

// 4. In Response: Return standardized cursor response
resp := pagination.NewCursorListResponse(users, cursorMeta)
response.OK(ctx, "success", resp)
```

**JSON Output:**
```json
{
  "items": [...],
  "pagination": {
    "limit": 10,
    "next_cursor": "eyJhbGciOiJ...",
    "prev_cursor": "eyJhbGciOiJ...",
    "has_next": true
  }
}
```

---

### 3. Unified Mode (Hybrid Endpoint)

Allow API consumers to choose either offset (`?page=2&limit=20`) or cursor (`?cursor=xyz&direction=next`) within the same endpoint:

```go
req := pagination.BindUnifiedFilterRequest(r, allowedFilters)

if req.IsCursor {
	// Execute cursor keyset logic (using req.DynamicCursorRequest and req.CursorValues)
} else {
	// Execute traditional offset logic (using req.OffsetRequest)
}
```

---

## 🛠️ Filter & Query Parameter Syntax

The binder automatically processes and normalizes two common query parameter styles:

1. **Grouped Syntax:**
   ```
   GET /users?filter=status:active,pending&filter=role:admin
   ```
2. **Direct Syntax:**
   ```
   GET /users?status=active,pending&role=admin
   ```

### Safety & Features:
- **SQL Injection Safe:** Only parameters listed in `allowedFilters` are parsed.
- **Automatic Deduplication:** Duplicate values in a column are collapsed into unique slices.
- **Search Support:** Reads `?search=term` via `req.Search`.
- **Max Limit Protection:** Caps client-requested limits to prevent database strain (e.g. `BindOffsetFilterRequest(r, allowedFilters, 50)`).
- **Convenient Filter Helpers:**
  ```go
  if req.HasFilter("users.status") {
  	statuses := req.GetFilter("users.status")           // []string{"active", "pending"}
  	primaryStatus := req.GetFirstFilter("users.status") // "active"
  }
  ```

---

## ⚙️ Under The Hood: Dynamic Keyset Pagination

When multiple rows share the same column value (e.g., duplicate timestamps or statuses), traditional single-column keyset conditions fail.

`pagination.BuildDynamicKeyset` generates mathematically sound nested composite SQL conditions:

```sql
-- For 2 columns (status, id):
(status > ?) OR (status = ? AND id < ?)

-- For 3 columns (status, created_at, id):
(status > ?) OR (status = ? AND created_at > ?) OR (status = ? AND created_at = ? AND id < ?)
```

Combined with automatic operator flipping when navigating backwards (`direction=prev`), this guarantees deterministic, zero-duplicate, and zero-skip pagination at enterprise scale.
