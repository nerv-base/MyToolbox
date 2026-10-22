// Command icon renders the application icon with the same native vector API
// used by the app. Run with go generate from the repository root.
package main

import (
	"image/png"
	"log"
	"os"

	"github.com/egoist/mygo/ui"
	"github.com/nerv-base/MyToolbox/internal/design"
)

func main() {
	img := ui.Render(func(c *ui.Context) {
		c.Root().Background(ui.Transparent)
		ui.Box(c).Fill().Padding(36).Children(func() {
			ui.Box(c).Fill().Radius(220).Background(ui.Hex("#286d53")).Center().
				Border(3, ui.Hex("#43866b")).Children(func() {
				ui.Icon(c, design.Logo).Size(600, 600).TextColor(ui.Hex("#ffffff"))
			})
		})
	}, 1024, 1024, 1)
	if err := os.MkdirAll("resources", 0755); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create("resources/icon.png")
	if err != nil {
		log.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod("resources/icon.png", 0644); err != nil {
		log.Fatal(err)
	}
}
