package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/theprimeagen/apollo-11/exec-tui/sourceserve"
)

func main() {
	dir, err := sourceserve.DocsDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "source-serve: %v\n", err)
		os.Exit(1)
	}
	addr := sourceserve.Addr()
	fmt.Printf("serving %s on http://127.0.0.1%s/\n", dir, addr)
	log.Fatal(http.ListenAndServe(addr, sourceserve.Handler(dir)))
}
