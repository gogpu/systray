// Example multi-tray demonstrates running two independent system tray icons
// simultaneously within the same application.
//
// Each tray has its own icon color, tooltip, menu, and click handlers,
// showing that multiple SystemTray instances work independently.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/gogpu/systray"
)

func main() {
	// First tray: red icon.
	iconRed := generateCircleIcon(22, color.RGBA{R: 220, G: 40, B: 40, A: 255})
	// Second tray: blue icon.
	iconBlue := generateCircleIcon(22, color.RGBA{R: 40, G: 80, B: 220, A: 255})

	tray1 := systray.New()
	tray2 := systray.New()

	// Menu for tray 1.
	menu1 := systray.NewMenu()
	menu1.Add("Red: Say Hello", func() {
		fmt.Printf("[Tray %d] Hello from RED tray!\n", tray1.ID())
	})
	menu1.Add("Red: Notify", func() {
		tray1.ShowNotification("Red Tray", "Notification from the red tray icon.")
	})
	menu1.AddSeparator()
	menu1.Add("Quit All", func() {
		fmt.Println("Quit All selected from red tray.")
		tray1.Remove()
		tray2.Remove()
		os.Exit(0)
	})

	// Menu for tray 2.
	menu2 := systray.NewMenu()
	menu2.Add("Blue: Say Hello", func() {
		fmt.Printf("[Tray %d] Hello from BLUE tray!\n", tray2.ID())
	})
	menu2.Add("Blue: Notify", func() {
		tray2.ShowNotification("Blue Tray", "Notification from the blue tray icon.")
	})
	menu2.AddSeparator()
	menu2.Add("Quit All", func() {
		fmt.Println("Quit All selected from blue tray.")
		tray1.Remove()
		tray2.Remove()
		os.Exit(0)
	})

	tray1.SetIcon(iconRed).SetTooltip("Red Tray (GoGPU)").SetMenu(menu1)
	tray1.OnClick(func() { fmt.Println("[Red] Left click!") })
	tray1.Show()

	tray2.SetIcon(iconBlue).SetTooltip("Blue Tray (GoGPU)").SetMenu(menu2)
	tray2.OnClick(func() { fmt.Println("[Blue] Left click!") })
	tray2.Show()

	fmt.Printf("Multi-tray example running. Red tray ID=%d, Blue tray ID=%d\n", tray1.ID(), tray2.ID())
	fmt.Println("Right-click either tray icon for its menu.")
	fmt.Println("Select 'Quit All' from either menu to exit.")

	// Only one Run() call is needed; the message loop serves all tray icons.
	if err := tray1.Run(); err != nil {
		fmt.Println("Run error:", err)
	}
}

// generateCircleIcon creates a filled circle icon with the given color.
func generateCircleIcon(size int, c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	mid := float64(size) / 2
	radius := mid - 1

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - mid + 0.5
			dy := float64(y) - mid + 0.5
			if dx*dx+dy*dy <= radius*radius {
				img.SetRGBA(x, y, c)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
