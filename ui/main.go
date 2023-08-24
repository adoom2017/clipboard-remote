package main

import (
    "log"

    "github.com/hajimehoshi/ebiten/v2"
    "golang.design/x/hotkey"
)

type Game struct{}

func (g *Game) Update() error {
    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
    return 320, 240
}

func main() {
    game := &Game{}
    ebiten.SetWindowSize(640, 480)
    ebiten.SetWindowTitle("Ebiten Game")
    go reghk()

    if err := ebiten.RunGame(game); err != nil {
        log.Fatal(err)
    }
}

func reghk() {
    // Register a desired hotkey.
    hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyS)

    if err := hk.Register(); err != nil {
        log.Printf("failed to register hotkey: %v", err)
        return
    }

    // Unregister the hotkey when keydown event is triggered
    <-hk.Keydown()
    log.Println("the registered hotkey is triggered.")

    hk.Unregister()
}
