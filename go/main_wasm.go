//go:build js

package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/loader"
	"wrestling/storage"
	"wrestling/ui"
)

func main() {
	store := storage.NewWASMStore(DefaultCards)

	roster, err := loader.LoadAllCards(store)
	if err != nil {
		log.Fatalf("Error loading wrestler cards: %v", err)
	}

	if len(roster) < 2 {
		log.Fatal("Need at least 2 wrestler cards")
	}

	game := ui.NewGame(roster, store)

	ebiten.SetWindowSize(ui.WindowWidth, ui.WindowHeight)
	ebiten.SetWindowTitle("Ring Wars")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
