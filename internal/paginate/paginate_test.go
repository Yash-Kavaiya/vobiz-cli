package paginate

import (
	"context"
	"testing"
)

func TestAll_StopsWhenHasMoreFalse(t *testing.T) {
	pages := [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7},
	}
	idx := 0
	fetch := func(ctx context.Context, cursor string) (Page[int], error) {
		p := Page[int]{Items: pages[idx], NextCursor: ""}
		idx++
		if idx < len(pages) {
			p.NextCursor = "more"
		}
		return p, nil
	}

	got, err := All(context.Background(), fetch)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{1, 2, 3, 4, 5, 6, 7}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestAll_RespectsLimit(t *testing.T) {
	fetch := func(ctx context.Context, cursor string) (Page[int], error) {
		return Page[int]{Items: []int{1, 2, 3, 4, 5}, NextCursor: "more"}, nil
	}
	got, err := AllN(context.Background(), fetch, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got len %d want 3", len(got))
	}
}
