// example/example.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/moistari/tlapi"
)

func main() {
	// read PHPSESSID, tluid, and tlpass cookies from browser ... hint: use github.com/zellyn/kooky !
	cl := tlapi.New(tlapi.WithCreds("<PHPSESSID>", "<tluid>", "<tlpass>"))
	req := tlapi.Search("framestor", "2019")
	for req.Next(context.Background(), cl) {
		torrent := req.Cur()
		fmt.Printf("%d: %s\n", torrent.ID, torrent.Name)
	}
	if err := req.Err(); err != nil {
		log.Fatal(err)
	}
}
