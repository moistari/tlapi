# About

TL API. See: [Go reference](https://pkg.go.dev/github.com/moistari/tlapi)

Using:

```sh
go get github.com/moistari/tlapi
```

Example:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/moistari/tlapi"
)

func main() {
	// read PHPSESSID, tluid, and tlpass cookies from browser and build a cookiejar ... hint: use github.com/zellyn/kooky !
	jar := buildJar()
	cl := tlapi.New(tlapi.WithJar(jar))
	res, err := cl.Search(context.Background(), "Fight Club")
	if err != nil {
		log.Fatal(err)
	}
	for i, t := range res.TorrentList {
		fmt.Printf("%02d: %s\n", i, t.Name)
	}
}
```
