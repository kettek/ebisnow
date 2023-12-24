package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"math/rand"
	"os"

	_ "embed"

	"fyne.io/systray"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sqweek/dialog"
)

//go:embed icon.ico
var icoBytes []byte

//go:embed snowplow.png
var snowplowBytes []byte

type EbiSnow struct {
	snow          []*Snow
	width, height int
	firstRun      bool
	piledSnow     *ebiten.Image
	pileSnow      bool
	//
	trayFuncStart     func()
	trayFuncEnd       func()
	trayQuitItem      *systray.MenuItem
	trayPileSnowItem  *systray.MenuItem
	trayWindItem      *systray.MenuItem
	trayClearSnowItem *systray.MenuItem
	//
	windDir        float64
	windPower      float64
	lastWindDir    float64
	lastWindPower  float64
	lastWindChange int
	windIntensity  float64
	//
	gravity float64
	//
	snowPlowX          float64
	snowPlowIterator   int
	snowPlowing        bool
	snowPlowIndex      int
	snowPlowClearImage *ebiten.Image
	//
	stopped bool
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

	if e.stopped {
		return nil
	}

	e.lastWindChange++
	if e.lastWindChange > 2000 {
		e.RandomizeWind()
	}

	wd, wp := e.Wind()

	if e.snowPlowing {
		e.snowPlowIterator++
		if e.snowPlowIterator > 5 {
			e.snowPlowIndex = (e.snowPlowIndex + 1) % 4
			e.snowPlowIterator = 0
		}
		e.snowPlowX++
		if e.snowPlowX > float64(e.width+snowImage.Bounds().Dx()) {
			e.snowPlowX = 0
			e.snowPlowing = false
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(e.snowPlowX+float64(snowplowImages[e.snowPlowIndex].Bounds().Dx())*2, 0)
		op.Blend = ebiten.BlendClear
		e.piledSnow.DrawImage(e.snowPlowClearImage, op)
	}

	for _, s := range e.snow {
		s.lifetime++

		x := s.x
		y := s.y

		if e.windPower > 0 {
			x += math.Cos(wd) * wp
			y += math.Sin(wd) * wp
		}

		y += s.speed * e.gravity

		if !e.pileSnow {
			s.x = x
			s.y = y
			continue
		}
		_, _, _, a := e.piledSnow.At(int(x), int(math.Floor(s.y+1))).RGBA()

		if a > 0 {
			y = s.y
		}

		if a > 0 || y >= float64(e.height)-1 {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(x, y)
			op.GeoM.Scale(s.Size())
			e.piledSnow.DrawImage(snowImage, op)
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
	//screen.Fill(color.Transparent)

	if e.stopped {
		return
	}

	for _, s := range e.snow {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(s.x, s.y)
		op.GeoM.Scale(s.Size())
		screen.DrawImage(snowImage, op)
	}
	if e.pileSnow {
		screen.DrawImage(e.piledSnow, nil)
	}
	if e.snowPlowing {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2.0, 2.0)
		op.GeoM.Translate(e.snowPlowX, float64(e.height-snowplowImages[e.snowPlowIndex].Bounds().Dy()*2))
		screen.DrawImage(snowplowImages[e.snowPlowIndex], op)
	}
}

func (e *EbiSnow) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	if e.width != outsideWidth || e.height != outsideHeight {
		e.piledSnow = ebiten.NewImage(outsideWidth, outsideHeight)
		e.snowPlowClearImage = ebiten.NewImage(1, outsideHeight)
		e.width, e.height = outsideWidth, outsideHeight
	}
	return outsideWidth, outsideHeight
}

func (e *EbiSnow) AddSnow() {
	e.snow = append(e.snow, &Snow{
		x:     rand.Float64() * float64(e.width),
		y:     -rand.Float64() * float64(e.height),
		speed: math.Max(0.5, rand.Float64()*2),
	})
}

var snowImage *ebiten.Image
var snowplowImages []*ebiten.Image

func init() {
	snowImage = ebiten.NewImage(1, 1)
	snowImage.Fill(color.White)

	{
		img, _, _ := image.Decode(bytes.NewReader(snowplowBytes))
		snowplowImage := ebiten.NewImageFromImage(img)
		for i := 0; i < 4; i++ {
			snowplowImages = append(snowplowImages, snowplowImage.SubImage(image.Rect(i*snowplowImage.Bounds().Dx()/4, 0, (i+1)*snowplowImage.Bounds().Dx()/4, snowplowImage.Bounds().Dy())).(*ebiten.Image))
		}
	}
}

type Snow struct {
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
		pileSnow:      true,
		windIntensity: 3,
		gravity:       1.0,
	}
	e.RandomizeWind()

	e.trayFuncStart, e.trayFuncEnd = systray.RunWithExternalLoop(func() {
		systray.SetIcon(icoBytes)
		systray.SetTitle("EbiSnow")
		systray.SetTooltip("EbiSnow")
		e.trayWindItem = systray.AddMenuItem("Wind", "Change wind intensity")
		e.trayWindItem.Enable()
		wind1 := e.trayWindItem.AddSubMenuItemCheckbox("None", "Change wind intensity", false)
		wind2 := e.trayWindItem.AddSubMenuItemCheckbox("Low", "Change wind intensity", false)
		wind3 := e.trayWindItem.AddSubMenuItemCheckbox("Moderate", "Change wind intensity", true)
		wind4 := e.trayWindItem.AddSubMenuItemCheckbox("Strong", "Change wind intensity", false)
		wind5 := e.trayWindItem.AddSubMenuItemCheckbox("Extreme", "Change wind intensity", false)
		winds := []*systray.MenuItem{wind1, wind2, wind3, wind4, wind5}

		gravityItem := systray.AddMenuItem("Gravity", "Change gravity")
		gravity1 := gravityItem.AddSubMenuItemCheckbox("None", "Change gravity", false)
		gravity2 := gravityItem.AddSubMenuItemCheckbox("Low", "Change gravity", false)
		gravity3 := gravityItem.AddSubMenuItemCheckbox("Moderate", "Change gravity", true)
		gravity4 := gravityItem.AddSubMenuItemCheckbox("Strong", "Change gravity", false)
		gravity5 := gravityItem.AddSubMenuItemCheckbox("Extreme", "Change gravity", false)
		gravities := []*systray.MenuItem{gravity1, gravity2, gravity3, gravity4, gravity5}

		countItem := systray.AddMenuItem("Snow count", "Change snow count")
		count1 := countItem.AddSubMenuItemCheckbox("Low", "Change snow count", false)
		count2 := countItem.AddSubMenuItemCheckbox("Moderate", "Change snow count", true)
		count3 := countItem.AddSubMenuItemCheckbox("High", "Change snow count", false)
		count4 := countItem.AddSubMenuItemCheckbox("Extreme", "Change snow count", false)
		counts := []*systray.MenuItem{count1, count2, count3, count4}

		e.trayPileSnowItem = systray.AddMenuItem("Pile snow", "Pile snow")
		e.trayPileSnowItem.Check()
		e.trayClearSnowItem = systray.AddMenuItem("Snowplow", "Clear the snow")
		e.trayClearSnowItem.Enable()
		selectSnowImage := systray.AddMenuItem("Select image", "Select image to use for snow")
		paused := systray.AddMenuItemCheckbox("Pause", "Pause ebisnow", false)
		systray.AddSeparator()
		e.trayQuitItem = systray.AddMenuItem("Quit", "Quit ebisnow")
		e.trayQuitItem.Enable()
		go func() {
			for {
				select {
				case <-paused.ClickedCh:
					if paused.Checked() {
						e.stopped = false
						paused.Uncheck()
					} else {
						e.stopped = true
						paused.Check()
					}
				case <-selectSnowImage.ClickedCh:
					if src, err := dialog.File().Filter("Image files", "png", "jpg", "jpeg").Title("Select image").Load(); err == nil {
						if b, err := os.ReadFile(src); err == nil {
							if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
								snowImage = ebiten.NewImageFromImage(img)
							}
						}
					}
				case <-wind1.ClickedCh:
					e.windIntensity = 0
					e.RandomizeWind()
					for _, w := range winds {
						w.Uncheck()
					}
					wind1.Check()
				case <-wind2.ClickedCh:
					e.windIntensity = 1
					e.RandomizeWind()
					for _, w := range winds {
						w.Uncheck()
					}
					wind2.Check()
				case <-wind3.ClickedCh:
					e.windIntensity = 3
					e.RandomizeWind()
					for _, w := range winds {
						w.Uncheck()
					}
					wind3.Check()
				case <-wind4.ClickedCh:
					e.windIntensity = 5
					e.RandomizeWind()
					for _, w := range winds {
						w.Uncheck()
					}
					wind4.Check()
				case <-wind5.ClickedCh:
					e.windIntensity = 7
					e.RandomizeWind()
					for _, w := range winds {
						w.Uncheck()
					}
					wind5.Check()
				case <-gravity1.ClickedCh:
					e.gravity = 0
					for _, g := range gravities {
						g.Uncheck()
					}
					gravity1.Check()
				case <-gravity2.ClickedCh:
					e.gravity = 0.50
					for _, g := range gravities {
						g.Uncheck()
					}
					gravity2.Check()
				case <-gravity3.ClickedCh:
					e.gravity = 1.0
					for _, g := range gravities {
						g.Uncheck()
					}
					gravity3.Check()
				case <-gravity4.ClickedCh:
					e.gravity = 2.0
					for _, g := range gravities {
						g.Uncheck()
					}
					gravity4.Check()
				case <-gravity5.ClickedCh:
					e.gravity = 4.0
					for _, g := range gravities {
						g.Uncheck()
					}
					gravity5.Check()
				case <-count1.ClickedCh:
					e.snow = e.snow[:0]
					for i := 0; i < 100; i++ {
						e.AddSnow()
					}
					for _, c := range counts {
						c.Uncheck()
					}
					count1.Check()
				case <-count2.ClickedCh:
					e.snow = e.snow[:0]
					for i := 0; i < 200; i++ {
						e.AddSnow()
					}
					for _, c := range counts {
						c.Uncheck()
					}
					count2.Check()
				case <-count3.ClickedCh:
					e.snow = e.snow[:0]
					for i := 0; i < 400; i++ {
						e.AddSnow()
					}
					for _, c := range counts {
						c.Uncheck()
					}
					count3.Check()
				case <-count4.ClickedCh:
					e.snow = e.snow[:0]
					for i := 0; i < 800; i++ {
						e.AddSnow()
					}
					for _, c := range counts {
						c.Uncheck()
					}
					count4.Check()
				case <-e.trayClearSnowItem.ClickedCh:
					e.snowPlowing = true
					e.snowPlowX = -float64(snowplowImages[e.snowPlowIndex].Bounds().Dx()*2 - 1)
				case <-e.trayPileSnowItem.ClickedCh:
					e.pileSnow = !e.pileSnow
					if e.pileSnow {
						e.trayPileSnowItem.Check()
					} else {
						e.trayPileSnowItem.Uncheck()
					}
				case <-e.trayQuitItem.ClickedCh:
					os.Exit(0)
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
