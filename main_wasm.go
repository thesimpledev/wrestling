//go:build js

package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/loader"
	"wrestling/internal/storage"
	"wrestling/internal/ui"
)

func main() {
	store := storage.NewWASMStore(DefaultCards)

	roster, problems, err := loader.LoadAllCards(store)
	if err != nil {
		log.Fatalf("Error loading wrestler cards: %v", err)
	}

	if len(roster) < 2 {
		log.Fatalf("Need at least 2 readable wrestler cards (problems: %v)", problems)
	}

	game := ui.NewGame(roster, store)
	game.ReportCardProblems(problems)

	ebiten.SetWindowSize(ui.WindowWidth, ui.WindowHeight)
	ebiten.SetWindowTitle("Ring Wars")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
