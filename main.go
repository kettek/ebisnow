package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	_ "embed"

	"fyne.io/systray"
	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed icon.ico
var icoBytes []byte

type EbiSnow struct {
	snow          []*Snow
	width, height int
	firstRun      bool
	piledSnow     *ebiten.Image
	//
	trayFuncStart     func()
	trayFuncEnd       func()
	trayQuitItem      *systray.MenuItem
	trayWindItem      *systray.MenuItem
	trayClearSnowItem *systray.MenuItem
	//
	windDir        float64
	windPower      float64
	lastWindDir    float64
	lastWindPower  float64
	lastWindChange int
	windIntensity  float64
}

func (e *EbiSnow) Wind() (dir, power float64) {
	r := float64(e.lastWindChange) / 2000.0
	return e.lastWindDir + (e.windDir-e.lastWindDir)*r, e.lastWindPower + (e.windPower-e.lastWindPower)*r
}

func (e *EbiSnow) RandomizeWind() {
	e.lastWindDir = e.windDir
	e.lastWindPower = e.windPower
	e.lastWindChange = 0
	e.windDir = math.Pi * rand.Float64()
	e.windPower = e.windIntensity * rand.Float64()
}

func (e *EbiSnow) Update() error {
	if e.firstRun {
		e.trayFuncStart()
		e.firstRun = false

		for i := 0; i < 200; i++ {
			e.AddSnow()
		}
	}

	e.lastWindChange++
	if e.lastWindChange > 2000 {
		e.RandomizeWind()
	}

	wd, wp := e.Wind()

	for _, s := range e.snow {
		s.lifetime++

		x := s.x
		y := s.y

		if e.windPower > 0 {
			x += math.Cos(wd) * wp
			y += math.Sin(wd) * wp
		}

		y += s.speed

		_, _, _, a := e.piledSnow.At(int(x), int(math.Floor(s.y+1))).RGBA()

		if a > 0 {
			y = s.y
		}

		if a > 0 || y >= float64(e.height)-1 {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(x, y)
			op.GeoM.Scale(s.Size())
			e.piledSnow.DrawImage(s.image, op)
			s.y = 0
			s.x = rand.Float64() * float64(e.width)
		} else {
			s.y = y
			if x > float64(e.width) {
				s.x = 0
			} else if x < 0 {
				s.x = float64(e.width)
			} else {
				s.x = x
			}
		}
	}
	return nil
}

func (e *EbiSnow) Draw(screen *ebiten.Image) {
	screen.Fill(color.Transparent)
	for _, s := range e.snow {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(s.x, s.y)
		op.GeoM.Scale(s.Size())
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

func (s *Snow) Size() (w, h float64) {
	return math.Round(0.5 + s.speed), math.Round(0.5 + s.speed)
}

func main() {
	e := &EbiSnow{
		firstRun:      true,
		windIntensity: 3,
	}
	e.RandomizeWind()

	e.trayFuncStart, e.trayFuncEnd = systray.RunWithExternalLoop(func() {
		systray.SetIcon(icoBytes)
		fmt.Println("apparently ready")
		systray.SetTitle("EbiSnow")
		systray.SetTooltip("EbiSnow")
		e.trayWindItem = systray.AddMenuItem("Wind - 3", "Change wind intensity")
		e.trayWindItem.Enable()
		e.trayClearSnowItem = systray.AddMenuItem("Snowplow", "Clear the snow")
		e.trayClearSnowItem.Enable()
		systray.AddSeparator()
		e.trayQuitItem = systray.AddMenuItem("Quit", "Quit ebisnow")
		e.trayQuitItem.Enable()
		go func() {
			for {
				select {
				case <-e.trayWindItem.ClickedCh:
					e.windIntensity++
					if e.windIntensity == 6 {
						e.windIntensity = 0
					}
					e.RandomizeWind()
					e.trayWindItem.SetTitle(fmt.Sprintf("Wind - %d", int(e.windIntensity)))
				case <-e.trayClearSnowItem.ClickedCh:
					e.piledSnow.Fill(color.Transparent)
				case <-e.trayQuitItem.ClickedCh:
					fmt.Println("quit")
				}
			}
		}()
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
		InitUnfocused:     true,
	}); err != nil {
		e.trayFuncEnd()
		panic(err)
	}
	e.trayFuncEnd()
}
