package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"fyne.io/systray"
	"github.com/hajimehoshi/ebiten/v2"
)

type EbiSnow struct {
	snow          []*Snow
	width, height int
	firstRun      bool
	piledSnow     *ebiten.Image
	trayFuncStart func()
	trayFuncEnd   func()
	windDir       float64
	windPower     float64
}

func (e *EbiSnow) Update() error {
	if e.firstRun {
		e.trayFuncStart()
		e.firstRun = false

		for i := 0; i < 200; i++ {
			e.AddSnow()
		}

	}
	for _, s := range e.snow {
		s.lifetime++
		//s.y += s.speed

		x := s.x
		y := s.y

		if e.windPower > 0 {
			x += math.Cos(e.windDir) * e.windPower
			y += math.Sin(e.windDir) * e.windPower
		}

		y += s.speed

		_, _, _, a := e.piledSnow.At(int(s.x), int(math.Floor(s.y+1))).RGBA()

		if a > 0 {
			y = s.y
		}

		if a > 0 || y >= float64(e.height)-1 {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(s.x, y)
			e.piledSnow.DrawImage(s.image, op)
			s.y = 0
			s.x = rand.Float64() * float64(e.width)
		} else {
			s.y = y
			//s.x += math.Cos(float64(s.lifetime)) * 0.5 * rand.Float64()
		}
		/*if s.y > float64(e.height) {
			s.y = 0
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(s.x, s.y)
			e.piledSnow.DrawImage(s.image, op)
		}*/
		//s.x += math.Cos(float64(s.lifetime))
	}
	return nil
}

func (e *EbiSnow) Draw(screen *ebiten.Image) {
	screen.Fill(color.Transparent)
	for _, s := range e.snow {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(s.x, s.y)
		screen.DrawImage(s.image, op)
	}
	screen.DrawImage(e.piledSnow, nil)
}

func (e *EbiSnow) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	if e.width != outsideWidth || e.height != outsideHeight {
		e.piledSnow = ebiten.NewImage(outsideWidth, outsideHeight)
		e.width, e.height = outsideWidth, outsideHeight
	}
	return outsideWidth, outsideHeight
}

func (e *EbiSnow) AddSnow() {
	e.snow = append(e.snow, &Snow{
		image: snowImage,
		x:     rand.Float64() * float64(e.width),
		y:     -rand.Float64() * float64(e.height),
		speed: math.Max(0.5, rand.Float64()*2),
	})
}

var snowImage *ebiten.Image

func init() {
	snowImage = ebiten.NewImage(1, 1)
	snowImage.Fill(color.White)
}

type Snow struct {
	image    *ebiten.Image
	x, y     float64
	speed    float64
	lifetime int
}

func main() {
	e := &EbiSnow{
		firstRun: true,
		//windDir:   math.Pi * rand.Float64(),
		//windPower: 20,
	}

	e.trayFuncStart, e.trayFuncEnd = systray.RunWithExternalLoop(func() {
		fmt.Println("apparently ready")
		//systray.SetTitle("EbiSnow")
		//systray.SetTooltip("EbiSnow")
		//mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
		systray.AddMenuItem("Quit", "Quit the whole app")
		//mQuit.ClickedCh
	}, func() {
		fmt.Println("quit?")
	})

	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowFloating(true)
	ebiten.SetWindowMousePassthrough(true)

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.MaximizeWindow()
	if err := ebiten.RunGameWithOptions(e, &ebiten.RunGameOptions{
		ScreenTransparent: true,
		SkipTaskbar:       true,
	}); err != nil {
		panic(err)
	}
	e.trayFuncEnd()
}
