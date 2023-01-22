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
		WithFacets(FacetSize, Size15GBPlus).
		WithOrder(OrderDesc).
		WithOrderBy(OrderBySize).
		WithPage(2)
	res, err := req.Do(context.Background(), cl)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for i, torrent := range res.TorrentList {
		t.Logf("%03d: %07d %q %d", i+1, torrent.ID, torrent.Name, torrent.Size)
	}
	t.Logf("numFound: %d", res.NumFound)
}

func TestNext(t *testing.T) {
	cl := buildClient(t)
	req := Search("framestor", "2019")
	var torrents []Torrent
	for req.Next(context.Background(), cl) {
		torrent := req.Cur()
		torrents = append(torrents, torrent)
		t.Logf("%d %03d: %07d %q %d", req.p, req.i, torrent.ID, torrent.Name, torrent.Size)
	}
	if err := req.Err(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if n, exp := len(torrents), 100; n < exp {
		t.Errorf("expected at least %d torrents, got: %d", exp, n)
	}
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
