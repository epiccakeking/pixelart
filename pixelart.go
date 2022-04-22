/*
This file is part of epiccakeking/pixelart.

epiccakeking/pixelart is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

epiccakeking/pixelart is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with epiccakeking/pixelart. If not, see <https://www.gnu.org/licenses/>.
*/
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
)

type ImageBuffer struct {
	Canvas        *image.RGBA
	Overlay       *image.RGBA
	canvasWidget  *canvas.Image
	overlayWidget *canvas.Image
	Widget        *fyne.Container
	CursorColor   *color.RGBA
	CursorPos     image.Point
}

func (i *ImageBuffer) MoveCursor(x, y int) {
	i.Overlay.Set(i.CursorPos.X*3, i.CursorPos.Y*3, color.Transparent)
	i.Overlay.Set(i.CursorPos.X*3+1, i.CursorPos.Y*3+1, color.Transparent)
	i.CursorPos = image.Point{x, y}
	i.DrawCursor()
}

func (i ImageBuffer) DrawCursor() {
	c := i.Canvas.RGBAAt(i.CursorPos.X, i.CursorPos.Y)
	if uint16(c.R)+uint16(c.G)+uint16(c.B) < 384 {
		c = color.RGBA{255, 255, 255, 255}
	} else {
		c = color.RGBA{0, 0, 0, 255}
	}
	i.Overlay.Set(i.CursorPos.X*3, i.CursorPos.Y*3, c)
	i.Overlay.Set(i.CursorPos.X*3+1, i.CursorPos.Y*3+1, i.CursorColor)
	i.Widget.Refresh()
}

func NewImageBuffer(im *image.RGBA, cursorColor *color.RGBA) ImageBuffer {
	bounds := im.Bounds()
	overlayImage := image.NewRGBA(image.Rect(0, 0, bounds.Dx()*3, bounds.Dy()*3))
	overlay := canvas.Image{
		Image:     overlayImage,
		ScaleMode: canvas.ImageScalePixels,
		FillMode:  canvas.ImageFillContain,
	}

	// Display
	myCanvas := canvas.Image{
		Image:     im,
		ScaleMode: canvas.ImageScalePixels,
		FillMode:  canvas.ImageFillContain,
	}
	return ImageBuffer{im, overlayImage, &myCanvas, &overlay, container.New(layout.NewMaxLayout(), &myCanvas, &overlay), cursorColor, bounds.Min}
}

type appState struct {
	app          fyne.App
	currentColor color.RGBA
	buffers      map[*ImageBuffer]bool
}

func (a *appState) redrawCursors() {
	for k := range a.buffers {
		k.DrawCursor()
	}
}

func openWindow(state *appState, imBuf *ImageBuffer) {
	w := state.app.NewWindow("pixelart")
	w.SetCloseIntercept(func() {
		delete(state.buffers, imBuf)
		w.Close()
	})
	state.buffers[imBuf] = true
	imBuf.DrawCursor()
	w.SetContent(imBuf.Widget)
	w.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		switch e.Name {
		case "Up":
			if imBuf.CursorPos.Y > imBuf.Canvas.Bounds().Min.Y {
				imBuf.MoveCursor(imBuf.CursorPos.X, imBuf.CursorPos.Y-1)
			}
		case "Down":
			if imBuf.CursorPos.Y+1 < imBuf.Canvas.Bounds().Max.Y {
				imBuf.MoveCursor(imBuf.CursorPos.X, imBuf.CursorPos.Y+1)
			}
		case "Left":
			if imBuf.CursorPos.X > imBuf.Canvas.Bounds().Min.X {
				imBuf.MoveCursor(imBuf.CursorPos.X-1, imBuf.CursorPos.Y)
			}
		case "Right":
			if imBuf.CursorPos.X+1 < imBuf.Canvas.Bounds().Max.X {
				imBuf.MoveCursor(imBuf.CursorPos.X+1, imBuf.CursorPos.Y)
			}
		case "Y":
			state.currentColor = imBuf.Canvas.RGBAAt(imBuf.CursorPos.X, imBuf.CursorPos.Y)
			state.redrawCursors()
		case "Space":
			imBuf.Canvas.Set(imBuf.CursorPos.X, imBuf.CursorPos.Y, state.currentColor)
			imBuf.DrawCursor()
			imBuf.Widget.Refresh()
		case "C":
			picker := dialog.NewColorPicker("Color picker", "", func(c color.Color) {
				r, g, b, a := c.RGBA()
				state.currentColor = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
				state.redrawCursors()
			}, w)
			picker.Advanced = true
			picker.Show()
		case "O":
			dialog.NewFileOpen(func(closer fyne.URIReadCloser, err error) {
				if err != nil {
					return
				}
				defer closer.Close()
				loadReader(state, closer)
			}, w).Show()
		case "S":
			dialog.NewFileSave(func(closer fyne.URIWriteCloser, err error) {
				if err != nil {
					return
				}
				defer closer.Close()
				png.Encode(closer, imBuf.Canvas)
			}, w).Show()
		case "N":
			buf := NewImageBuffer(image.NewRGBA(image.Rect(0, 0, 50, 50)), &state.currentColor)
			openWindow(state, &buf)
		}
	})
	w.Show()
}

func loadReader(state *appState, f io.Reader) {
	// Load the image
	m, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	// Image needs to be converted to RGBA
	bounds := m.Bounds()
	converted := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(converted, converted.Bounds(), m, bounds.Min, draw.Src)

	// Create buffer
	imBuf := NewImageBuffer(converted, &state.currentColor)

	openWindow(state, &imBuf)
}

func openFile(state *appState, name string) {
	// Load the image
	f, err := os.Open(name)
	if err != nil {
		panic(err)
	}
	m, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		panic(err)
	}
	// Image needs to be converted to RGBA
	bounds := m.Bounds()
	converted := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(converted, converted.Bounds(), m, bounds.Min, draw.Src)

	// Create buffer
	imBuf := NewImageBuffer(converted, &state.currentColor)

	go openWindow(state, &imBuf)
}

func main() {
	state := appState{
		app.New(),
		color.RGBA{0, 0, 0, 255},
		make(map[*ImageBuffer]bool),
	}
	loadReader(&state, bytes.NewReader(resourceHelloworldPng.StaticContent))
	state.app.Run()
}
