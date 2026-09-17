// Package mobile exposes MegaTwist to ebitenmobile.
package mobile

import (
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"

	"megatwist"
)

func init() {
	enginemobile.SetGame(megatwist.NewGame())
}

// Dummy forces gomobile to include this package in the Android binding.
func Dummy() {}
