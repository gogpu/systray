// Example notification demonstrates OS-level notifications from a system tray icon.
//
// The tray menu offers several notification types to test the platform's
// notification subsystem (Win32 balloon tips, macOS Notification Center,
// Linux org.freedesktop.Notifications via D-Bus).
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
	icon := generateBellIcon(22)

	tray := systray.New()

	menu := systray.NewMenu()
	menu.Add("Info Notification", func() {
		fmt.Printf("[%s] Sending info notification...\n", timestamp())
		tray.ShowNotification("Information", "This is an informational message from GoGPU systray.")
	})
	menu.Add("Reminder Notification", func() {
		fmt.Printf("[%s] Sending reminder notification...\n", timestamp())
		tray.ShowNotification("Reminder", "Don't forget to check system tray features!")
	})
	menu.AddSeparator()
	menu.Add("Quit", func() {
		fmt.Println("Exiting notification example...")
		tray.Remove()
		os.Exit(0)
	})

	tray.SetIcon(icon).
		SetTooltip("GoGPU Notification Example").
		SetMenu(menu).
		Show()

	fmt.Println("Notification example running.")
	fmt.Println("Right-click the tray icon and select a notification type.")
	fmt.Println("Select Quit to exit.")

	if err := tray.Run(); err != nil {
		fmt.Println("Run error:", err)
	}
}

func timestamp() string {
	return time.Now().Format("15:04:05")
}

// generateBellIcon creates a simple bell-shaped icon programmatically.
func generateBellIcon(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	mid := size / 2
	bellColor := color.RGBA{R: 255, G: 180, B: 0, A: 255}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Bell body: a centered trapezoid shape.
			topY := size / 4
			botY := size * 3 / 4

			if y >= topY && y <= botY {
				// Width expands from top to bottom.
				progress := float64(y-topY) / float64(botY-topY)
				halfWidth := int(float64(size/6) + progress*float64(size/4))
				if x >= mid-halfWidth && x <= mid+halfWidth {
					img.SetRGBA(x, y, bellColor)
				}
			}
			// Bell clapper: small dot at the bottom center.
			if y > botY && y <= botY+2 && x >= mid-1 && x <= mid+1 {
				img.SetRGBA(x, y, bellColor)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
