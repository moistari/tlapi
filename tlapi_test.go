package tlapi

import (
	"bytes"
	"context"
	"testing"
)

func TestSearch(t *testing.T) {
	cl := buildClient(t)
	req := Search().
		WithCategories(CategoryForeignMovies).
		WithFacets("size", Size15GBPlus).
		WithOrder(OrderDesc).
		WithOrderBy(OrderBySize).
		WithPage(2)
	res, err := req.Do(context.Background(), cl)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for i, r := range res.TorrentList {
		t.Logf("%03d: %07d %q %d", i+1, r.ID, r.Name, r.Size)
	}
	t.Logf("numFound: %d", res.NumFound)
}

func TestTorrent(t *testing.T) {
	cl := buildClient(t)
	buf, err := cl.Torrent(context.Background(), 1319660)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !bytes.Contains(buf, []byte("Fight.Club.1999.1080p.BluRay.REMUX.AVC.DTS-HD.MA5.1-HDH")) {
		t.Errorf("expected buf to contain torrent name")
	}
}
