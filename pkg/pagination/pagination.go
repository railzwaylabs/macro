package pagination

import (
	"encoding/base64"
	"encoding/json"
)

// Pagination constants
const (
	DefaultPageSize = 10  // Default number of items per page
	MaxPageSize     = 250 // Maximum allowed page size
	MinPageSize     = 1   // Minimum allowed page size
)

// Pagination represents query parameters for cursor-based pagination.
// Use page_token to navigate through pages and page_size to control items per page.
type Pagination struct {
	PageToken string `form:"page_token"`                                    // Opaque cursor for pagination
	PageSize  int    `form:"page_size,default=10" validate:"gte=1,lte=250"` // Items per page (1-250)
}

// Cursor represents the internal structure of a page token.
// It contains the last item's ID and timestamp for cursor-based pagination.
type Cursor struct {
	ID        string `json:"id,omitempty"`         // Last item ID
	CreatedAt string `json:"created_at,omitempty"` // Last item timestamp
	Name      string `json:"name,omitempty"`       // Last item name (for sorting by name)
}

// PageInfo contains pagination metadata returned in API responses.
type PageInfo struct {
	NextPageToken     string `json:"next_page_token"`     // Token for next page (empty if last page)
	PreviousPageToken string `json:"previous_page_token"` // Token for previous page (empty if first page)
	HasMore           bool   `json:"has_more"`            // Whether more pages exist
}

// EncodeCursor encodes a Cursor into a base64-encoded page token.
func EncodeCursor(data Cursor) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", nil
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

// DecodeCursor decodes a base64-encoded page token into a Cursor.
func DecodeCursor(data string) (*Cursor, error) {
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	var cursor Cursor
	if err := json.Unmarshal(b, &cursor); err != nil {
		return nil, err
	}

	return &cursor, nil
}

// BuildCursorPageInfo builds PageInfo from a list of items.
// It automatically determines if there are more pages and extracts the cursor from the last item.
// The data slice should contain limit+1 items to detect if there are more pages.
func BuildCursorPageInfo[T any](data []*T, limit int32, extractCursor func(*T) string) *PageInfo {
	if len(data) == 0 {
		return &PageInfo{HasMore: false}
	}

	hasMore := false
	if len(data) > int(limit) {
		hasMore = true
		data = data[:limit]
	}

	pageInfo := &PageInfo{
		HasMore:       hasMore,
		NextPageToken: extractCursor(data[len(data)-1]),
	}

	return pageInfo
}

// ValidatePageSize validates and normalizes page size to be within allowed bounds.
func ValidatePageSize(pageSize int) int {
	if pageSize < MinPageSize {
		return DefaultPageSize
	}
	if pageSize > MaxPageSize {
		return MaxPageSize
	}
	return pageSize
}
