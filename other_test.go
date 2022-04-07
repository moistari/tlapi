//go:build !real

package tlapi

import "testing"

func buildClient(t *testing.T) *Client {
	t.Fatal("bad test tag")
	return nil
}
