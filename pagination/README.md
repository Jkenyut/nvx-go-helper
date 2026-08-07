# Keyset / Cursor Pagination Helper

This package provides an industry-standard implementation of **Bidirectional Dynamic Keyset Pagination** (also known as cursor pagination). It allows clients to sort API tables dynamically (e.g. by `name`, `status`, `created_at`) while maintaining extreme performance (no `OFFSET` bottleneck) and strict consistency (no duplicate or skipped records).

## Features
- **Dynamic Keyset Support:** Can parse and build cursors that contain 1, 2, or 3+ column values at once. Simply send comma-separated strings (e.g. `sort_by=status,name` and `sort_type=asc,desc`) from the frontend.
- **Tie-Breaker Safety:** Automatically appends a unique column (e.g., `id`) to ensure deterministic sorting, preventing infinite loops.
- **Bidirectional Navigation:** Safely handles `<` and `>` SQL operator inversions when the user clicks "Previous Page", and reverses the output arrays seamlessly.
- **Base64 Cursors:** Hides implementation details from the client.
- **DTO Binding:** Offers a plug-and-play struct (`DynamicCursorRequest`) to avoid boilerplate code in HTTP handlers.

## Getting Started

### 1. The DTO & HTTP Handler
Instead of defining cursor variables in every Request struct, embed the `DynamicCursorRequest`:

```go
import "github.com/Jkenyut/nvx-go-helper/pagination"

type UserGetRequest struct {
	Search string `json:"search"`
	
	// Magic field that instantly injects all pagination params
	pagination.DynamicCursorRequest 
}
```

In your handler, extract it easily:
```go
req := UserGetRequest{
	Search:               request.GetQueryString(r, "search", ""),
	DynamicCursorRequest: pagination.BindDynamicCursorRequest(r), // Automates extraction & default values
}
```

### 2. The Repository (Database / SQL)
You must prepare the SQL strings based on the user's requested sort options. We provide `PrepareDynamicSort` to handle the heavy lifting (anti-SQL-Injection mapped columns, appending unique keys, and handling `direction=prev`).

```go
allowedSort := map[string]string{
	"name":       "user_name",
	"created_at": "user_created_at",
}

// 1. Prepare SQL
sortRes := pagination.PrepareDynamicSort(pagination.DynamicSortParams{
	SortBy:         req.SortBy,
	SortType:       req.SortType,
	Direction:      req.Direction,
	AllowedColumns: allowedSort,
	UniqueColumn:   "user_id", // Essential to prevent skipped rows
	UniqueSortType: "DESC",
})

// sortRes.Columns -> ["user_name", "user_id"]
// sortRes.Operators -> [">", "<"]
// sortRes.OrderStrs -> ["user_name ASC", "user_id DESC"]

// 2. Decode the cursor sent by user
var cursorVals []interface{}
if req.Cursor != "" {
	cursorVals, _ = pagination.DecodeDynamicCursor(req.Cursor)
}

// 3. Generate SQL string (for Squirrel / GORM)
var sqlWhere string
var sqlArgs []interface{}
if len(cursorVals) > 0 {
	sqlWhere, sqlArgs = pagination.BuildDynamicKeyset(sortRes.Columns, sortRes.Operators, cursorVals)
}

// Resulting sqlWhere: "(user_name > ?) OR (user_name = ? AND user_id < ?)"
```

### 3. The Service Layer (Business Logic)
Once you receive the `users` array from your database, pass it into the generator. It will construct the `next_cursor` and `prev_cursor` correctly based on the items in the slice.

```go
import "github.com/Jkenyut/nvx-go-helper/sliceutil"

// If the user requested the 'prev' page, the DB returned it backwards. You MUST reverse it:
if req.Direction == "prev" {
	users = sliceutil.Reverse(users)
}

// Generate the Cursor output:
pageRes := pagination.GenerateBidirectionalCursor(
	users, 
	req.Limit, 
	req.Direction, 
	req.Cursor, 
	func(u User) []interface{} {
		// Define how to extract the struct values matching the sort request!
		var vals []interface{}
		
		sortBys := strings.Split(req.SortBy, ",")
		for _, col := range sortBys {
			col = strings.ToLower(strings.TrimSpace(col))
			switch col {
			case "name":
				vals = append(vals, u.Name)
			case "created_at":
				vals = append(vals, u.CreatedAt)
			}
		}
		
		// The tie-breaker must always be manually appended at the end
		vals = append(vals, u.ID)
		return vals
	},
)

// Now pageRes contains:
// - NextCursor (Base64)
// - PrevCursor (Base64)
// - HasNext (bool)
```

## How It Works Under The Hood

If a user sorts by `status ASC` but 10 users have the same status, traditional 1-column cursors fail (because `status > 'active'` skips the other active users). 
This package automatically generates SQL equivalent to:
```sql
WHERE (status > ?) OR (status = ? AND id < ?)
ORDER BY status ASC, id DESC
```
This guarantees mathematically perfect pagination at all scales.
