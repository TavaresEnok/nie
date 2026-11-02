package main

import (
	"context"
	"fmt"
	"log"

	"nie/internal/discovery"
)

func main() {
	m := discovery.NewMwmbl("https://api.mwmbl.org")
	res, err := m.Search(context.Background(), "comandos olt fiberhome", 5)
	if err != nil {
		log.Fatalf("Search error: %v", err)
	}
	fmt.Printf("Retornou %d resultados:\n", len(res))
	for i, r := range res {
		fmt.Printf("[%d] URL: %s | Title: %s\n", i+1, r.URL, r.Title)
	}
}
