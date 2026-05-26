// Package paginate provides a generic cursor pager.
package paginate

import "context"

type Page[T any] struct {
	Items      []T
	NextCursor string // empty when no more pages
}

type Fetcher[T any] func(ctx context.Context, cursor string) (Page[T], error)

// All fetches every page until NextCursor is empty.
func All[T any](ctx context.Context, fetch Fetcher[T]) ([]T, error) {
	return AllN(ctx, fetch, -1)
}

// AllN fetches pages until either no more pages remain or `limit` items are collected.
// A negative limit means unbounded.
func AllN[T any](ctx context.Context, fetch Fetcher[T], limit int) ([]T, error) {
	var out []T
	cursor := ""
	for {
		p, err := fetch(ctx, cursor)
		if err != nil {
			return nil, err
		}
		out = append(out, p.Items...)
		if limit >= 0 && len(out) >= limit {
			return out[:limit], nil
		}
		if p.NextCursor == "" {
			return out, nil
		}
		cursor = p.NextCursor
	}
}
