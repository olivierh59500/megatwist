package main

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type fakeGame struct {
	updates int
	draws   int
	err     error
}

func (g *fakeGame) Update() error {
	g.updates++
	return g.err
}

func (g *fakeGame) Draw(*ebiten.Image) {
	g.draws++
}

func (g *fakeGame) Layout(int, int) (int, int) {
	return 832, 552
}

func TestDrawOnUpdateGame(t *testing.T) {
	inner := &fakeGame{}
	game := newDrawOnUpdateGame(inner)
	game.Draw(nil)
	game.Draw(nil)
	if inner.draws != 1 {
		t.Fatalf("initial draws = %d, want 1", inner.draws)
	}

	if err := game.Update(); err != nil {
		t.Fatal(err)
	}
	game.Draw(nil)
	game.Draw(nil)
	if inner.updates != 1 || inner.draws != 2 {
		t.Fatalf("updates, draws = %d, %d; want 1, 2", inner.updates, inner.draws)
	}
}

func TestDrawOnUpdateGameDoesNotDirtyAfterFailedUpdate(t *testing.T) {
	updateErr := errors.New("update failed")
	inner := &fakeGame{err: updateErr}
	game := newDrawOnUpdateGame(inner)
	game.Draw(nil)
	if err := game.Update(); !errors.Is(err, updateErr) {
		t.Fatalf("Update error = %v, want %v", err, updateErr)
	}
	game.Draw(nil)
	if inner.draws != 1 {
		t.Fatalf("draws = %d, want 1", inner.draws)
	}
}
