// Example dynamic-submenu demonstrates dynamic updates on submenu items
// and submenu containers — the fix from PRs #24 (macOS) and #27 (Windows).
//
// Every second, the example updates:
//   - A counter label inside a submenu
//   - The submenu container label itself (shows item count)
//   - A checkbox toggle inside a submenu
//
// Right-click the tray icon to observe live updates.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"

	"github.com/gogpu/systray"
)

func main() {
	tray := systray.New()

	// Build submenu with dynamic items.
	sub := systray.NewMenu()
	counter := sub.Add("Elapsed: 0s", nil)
	autoRefresh := sub.AddCheckbox("Auto-refresh", true, nil)
	sub.AddSeparator()
	sub.Add("Reset", func() {
		counter.SetLabel("Elapsed: 0s")
		autoRefresh.SetChecked(true)
	})

	// Build root menu.
	root := systray.NewMenu()
	root.Add("Status: running", nil)
	root.AddSeparator()

	// Submenu container — its label will update dynamically.
	container := root.AddSubmenu("Options (4 items)", sub)

	root.AddSeparator()
	root.Add("Quit", func() {
		tray.Remove()
		os.Exit(0)
	})

	icon := generateIcon(22, color.RGBA{R: 40, G: 160, B: 80, A: 255})
	tray.SetIcon(icon).SetTooltip("Dynamic Submenu Demo").SetMenu(root).Show()

	// Background goroutine updates submenu items every second.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		start := time.Now()
		tick := 0
		for range ticker.C {
			tick++
			elapsed := int(time.Since(start).Seconds())

			// Update item inside submenu.
			counter.SetLabel(fmt.Sprintf("Elapsed: %ds", elapsed))

			// Toggle checkbox every 5 seconds.
			if tick%5 == 0 {
				autoRefresh.SetChecked(!autoRefresh.IsChecked())
			}

			// Update submenu container label.
			container.SetLabel(fmt.Sprintf("Options (%ds elapsed)", elapsed))
		}
	}()

	fmt.Println("Dynamic submenu demo running.")
	fmt.Println("Right-click tray icon → observe 'Options' label and counter updating live.")
	fmt.Println("Click Quit to exit.")

	if err := tray.Run(); err != nil {
		fmt.Println("Run error:", err)
	}
}

func generateIcon(size int, c color.RGBA) []byte {
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
