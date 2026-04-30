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
	// Light mode icon: green (visible on light taskbar backgrounds).
	iconLight := generateIcon(22, color.RGBA{R: 0, G: 180, B: 80, A: 255})
	// Dark mode icon: bright cyan (visible on dark taskbar backgrounds).
	iconDark := generateIcon(22, color.RGBA{R: 0, G: 230, B: 230, A: 255})

	tray := systray.New()

	menu := systray.NewMenu()
	menu.Add("Hello", func() { fmt.Println("Hello clicked!") })
	menu.Add("Show Notification", func() {
		fmt.Println("Sending notification...")
		tray.ShowNotification("GoGPU", "Hello from systray!")
	})
	menu.AddSeparator()

	sub := systray.NewMenu()
	sub.Add("Sub Item 1", func() { fmt.Println("Sub 1") })
	sub.Add("Sub Item 2", func() { fmt.Println("Sub 2") })
	menu.AddSubmenu("More...", sub)

	menu.AddCheckbox("Check me", false, func() { fmt.Println("Checkbox toggled") })
	menu.AddSeparator()
	menu.Add("Quit", func() {
		fmt.Println("Quit clicked, removing tray...")
		tray.Remove()
		os.Exit(0)
	})

	tray.SetIcon(iconLight).
		SetDarkModeIcon(iconDark).
		SetTooltip("GoGPU Systray Test").
		SetMenu(menu)
	tray.OnClick(func() { fmt.Println("Left click!") })
	tray.OnDoubleClick(func() { fmt.Println("Double click!") })
	tray.OnRightClick(func() { fmt.Println("Right click!") })
	tray.Show()

	fmt.Println("Systray running. Right-click the tray icon for menu.")
	fmt.Println("Toggle dark/light mode in Windows Settings to see the icon change.")
	fmt.Println("Click Quit to exit.")
	if err := tray.Run(); err != nil {
		fmt.Println("Run error:", err)
	}
}

func generateIcon(size int, c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if x == 0 || x == size-1 || y == 0 || y == size-1 {
				img.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			} else {
				img.SetRGBA(x, y, c)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
