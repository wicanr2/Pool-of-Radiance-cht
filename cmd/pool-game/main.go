package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

const (
	logicalWidth  = 640
	logicalHeight = 400
)

type screenMode uint8

const (
	modeTitle screenMode = iota
	modeMenu
	modeCreation
)

type diceRoller struct{ random *rand.Rand }

func (roller diceRoller) Roll(count, sides int) int {
	total := 0
	for index := 0; index < count; index++ {
		total += roller.random.Intn(sides) + 1
	}
	return total
}

type keySource interface{ JustPressed(ebiten.Key) bool }
type ebitenKeys struct{}

func (ebitenKeys) JustPressed(key ebiten.Key) bool { return inpututil.IsKeyJustPressed(key) }
func (ebitenKeys) Chars() []rune                   { return ebiten.AppendInputChars(nil) }

type app struct {
	mode         screenMode
	title        *ebiten.Image
	flow         creation.Flow
	cursor       int
	rolled       *creation.RolledCharacter
	roller       diceRoller
	help         bool
	modern       bool
	statusLine   string
	keys         keySource
	nameInput    string
	portrait     *ebiten.Image
	loadPortrait func(head, body uint8) (*ebiten.Image, error)
}

func newApp(zipPath string) (*app, error) {
	pictures, err := assets.ReadTitlePictures(zipPath)
	if err != nil {
		return nil, err
	}
	rendered, err := pictures[1].RGBA(0, graphics.EGA16)
	if err != nil {
		return nil, err
	}
	application := &app{
		mode:   modeTitle,
		title:  ebiten.NewImageFromImage(rendered),
		flow:   creation.NewFlow(),
		roller: diceRoller{random: rand.New(rand.NewSource(time.Now().UnixNano()))},
		keys:   ebitenKeys{},
	}
	application.loadPortrait = func(head, body uint8) (*ebiten.Image, error) {
		parts, err := assets.ReadCreationPortraitParts(zipPath, head, body)
		if err != nil {
			return nil, err
		}
		composed, err := assets.ComposeCreationPortrait(parts)
		if err != nil {
			return nil, err
		}
		rendered, err := composed.RGBA(0, graphics.EGA16)
		if err != nil {
			return nil, err
		}
		return ebiten.NewImageFromImage(rendered), nil
	}
	return application, nil
}

func (a *app) justPressed(key ebiten.Key) bool {
	return a.keys != nil && a.keys.JustPressed(key)
}

func (a *app) inputChars() []rune {
	if source, ok := a.keys.(interface{ Chars() []rune }); ok {
		return source.Chars()
	}
	return nil
}

func (a *app) reloadPortrait() error {
	if a.loadPortrait == nil {
		return fmt.Errorf("portrait loader is not configured")
	}
	portrait, err := a.loadPortrait(a.flow.PortraitHead, a.flow.PortraitBody)
	if err != nil {
		return err
	}
	a.portrait = portrait
	return nil
}

func (a *app) Update() error {
	if a.justPressed(ebiten.KeyF10) {
		return ebiten.Termination
	}
	if a.justPressed(ebiten.KeyF1) {
		a.help = !a.help
	}
	if a.justPressed(ebiten.KeyF2) {
		a.modern = !a.modern
	}
	if a.help {
		if a.justPressed(ebiten.KeyEscape) {
			a.help = false
		}
		return nil
	}
	switch a.mode {
	case modeTitle:
		if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
			a.mode = modeMenu
		}
	case modeMenu:
		if a.justPressed(ebiten.KeyC) || a.justPressed(ebiten.KeyEnter) {
			a.flow, a.cursor, a.rolled = creation.NewFlow(), 0, nil
			a.mode = modeCreation
		}
	case modeCreation:
		return a.updateCreation()
	}
	return nil
}

func (a *app) updateCreation() error {
	if a.justPressed(ebiten.KeyEscape) {
		if a.flow.Stage == creation.StageRace {
			a.mode, a.cursor, a.rolled = modeMenu, 0, nil
			return nil
		}
		a.flow.Back()
		if a.flow.Stage == creation.StageName {
			a.nameInput = a.flow.Name
		}
		if a.flow.Stage != creation.StagePortrait {
			a.portrait = nil
		}
		a.cursor, a.rolled = 0, nil
		return nil
	}
	if a.flow.Stage == creation.StageRoll {
		if a.rolled == nil || a.justPressed(ebiten.KeyR) {
			rolled, err := a.flow.Roll(a.roller)
			if err != nil {
				return err
			}
			a.rolled = &rolled
		}
		if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeyY) {
			if err := a.flow.AcceptRoll(); err != nil {
				return err
			}
			a.nameInput, a.statusLine = "", ""
		}
		return nil
	}
	if a.flow.Stage == creation.StageName {
		if a.justPressed(ebiten.KeyBackspace) && len(a.nameInput) > 0 {
			a.nameInput = a.nameInput[:len(a.nameInput)-1]
		}
		for _, entered := range a.inputChars() {
			if entered >= 0x20 && entered <= 0x7E && len(a.nameInput) < 15 {
				a.nameInput += strings.ToUpper(string(entered))
			}
		}
		if a.justPressed(ebiten.KeyEnter) {
			if err := a.flow.SetName(a.nameInput); err != nil {
				a.statusLine = err.Error()
				return nil
			}
			a.statusLine = ""
			return a.reloadPortrait()
		}
		return nil
	}
	if a.flow.Stage == creation.StagePortrait {
		changed := false
		if a.justPressed(ebiten.KeyH) {
			if err := a.flow.NextPortraitHead(); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyB) {
			if err := a.flow.NextPortraitBody(); err != nil {
				return err
			}
			changed = true
		}
		if changed {
			return a.reloadPortrait()
		}
		if a.justPressed(ebiten.KeyK) {
			if err := a.flow.KeepPortrait(); err != nil {
				return err
			}
			a.statusLine = "Portrait accepted; original combat icon editor is next."
		}
		return nil
	}
	if a.flow.Stage == creation.StageIcon {
		return nil
	}
	options := a.flow.Options()
	if a.justPressed(ebiten.KeyArrowUp) || a.justPressed(ebiten.KeyHome) {
		a.cursor = (a.cursor + len(options) - 1) % len(options)
	}
	if a.justPressed(ebiten.KeyArrowDown) || a.justPressed(ebiten.KeyEnd) {
		a.cursor = (a.cursor + 1) % len(options)
	}
	if a.justPressed(ebiten.KeyEnter) {
		if err := a.flow.Select(a.cursor); err != nil {
			return err
		}
		a.cursor = 0
	}
	return nil
}

func (a *app) Draw(screen *ebiten.Image) {
	background, foreground, accent := color.RGBA{0, 0, 0, 255}, color.RGBA{170, 255, 255, 255}, color.RGBA{255, 255, 85, 255}
	if a.modern {
		background, foreground, accent = color.RGBA{16, 20, 30, 255}, color.RGBA{238, 232, 207, 255}, color.RGBA{255, 202, 72, 255}
	}
	screen.Fill(background)
	if a.mode == modeTitle {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(a.title, op)
		drawText(screen, "ENTER / SPACE", 264, 382, accent)
	} else if a.mode == modeMenu {
		drawFrame(screen, foreground, accent)
		drawText(screen, "POOL OF RADIANCE", 224, 54, accent)
		drawText(screen, "C  CREATE NEW CHARACTER", 176, 122, foreground)
		drawText(screen, "   ADD CHARACTER TO PARTY   [pending]", 176, 150, color.RGBA{110, 120, 125, 255})
		drawText(screen, "   LOAD SAVED GAME          [pending]", 176, 178, color.RGBA{110, 120, 125, 255})
		drawText(screen, "ENTER also starts character creation", 176, 226, foreground)
	} else {
		drawCreation(screen, a, foreground, accent)
	}
	drawText(screen, "F1 Help  F2 Theme  ESC Back  F10 Quit", 16, 390, foreground)
	if a.help {
		drawHelp(screen, background, foreground, accent)
	}
}

func drawCreation(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	drawFrame(screen, foreground, accent)
	if a.flow.Stage == creation.StageRoll {
		drawText(screen, "CHARACTER SHEET", 230, 42, accent)
		if a.rolled == nil {
			drawText(screen, "Rolling...", 40, 82, foreground)
			return
		}
		value := a.rolled
		drawText(screen, fmt.Sprintf("%s  %s  %s", a.flow.SelectedGender().Label, a.flow.SelectedRace().Label, a.flow.SelectedClass().Label), 48, 82, foreground)
		drawText(screen, fmt.Sprintf("AGE %d", value.Age), 48, 110, foreground)
		for index, name := range []string{"STR", "INT", "WIS", "DEX", "CON", "CHA"} {
			extra := ""
			if index == 0 && value.ExceptionalStrength != 0 {
				extra = fmt.Sprintf("/%02d", value.ExceptionalStrength)
			}
			drawText(screen, fmt.Sprintf("%-3s %2d%s", name, value.Abilities[index], extra), 48+(index/3)*180, 150+(index%3)*28, foreground)
		}
		drawText(screen, fmt.Sprintf("GOLD %d     HP %d", value.Gold, value.HP), 48, 252, foreground)
		drawText(screen, "KEEP THIS CHARACTER?  ENTER/Y = YES   R = REROLL", 48, 302, accent)
		if a.statusLine != "" {
			drawText(screen, a.statusLine, 48, 334, foreground)
		}
		return
	}
	if a.flow.Stage == creation.StageName {
		drawText(screen, "CHARACTER NAME:", 160, 128, accent)
		drawText(screen, a.nameInput+"_", 160, 164, foreground)
		drawText(screen, "1-15 CHARACTERS; ENTER ACCEPTS", 160, 214, foreground)
		if a.statusLine != "" {
			drawText(screen, a.statusLine, 48, 334, foreground)
		}
		return
	}
	if a.flow.Stage == creation.StagePortrait {
		drawText(screen, "HEAD / BODY / KEEP", 48, 72, accent)
		drawText(screen, fmt.Sprintf("H HEAD %02d/14", a.flow.PortraitHead), 48, 118, foreground)
		drawText(screen, fmt.Sprintf("B BODY %02d/12", a.flow.PortraitBody), 48, 150, foreground)
		drawText(screen, "K KEEP", 48, 182, foreground)
		if a.portrait != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(2, 2)
			op.GeoM.Translate(448, 16)
			screen.DrawImage(a.portrait, op)
		}
		return
	}
	if a.flow.Stage == creation.StageIcon {
		drawText(screen, "COMBAT ICON EDITOR", 216, 72, accent)
		drawText(screen, "READY / ACTION customization is the next slice.", 96, 132, foreground)
		drawText(screen, a.statusLine, 96, 176, foreground)
		return
	}
	title := map[creation.Stage]string{creation.StageRace: "PICK RACE", creation.StageGender: "PICK GENDER", creation.StageClass: "PICK CLASS", creation.StageAlignment: "PICK ALIGNMENT"}[a.flow.Stage]
	drawText(screen, title, 250, 42, accent)
	for index, option := range a.flow.Options() {
		prefix := "  "
		ink := foreground
		if index == a.cursor {
			prefix, ink = "> ", accent
		}
		drawText(screen, prefix+option, 128, 82+index*22, ink)
	}
	drawText(screen, creation.HintFor(stageName(a.flow.Stage)), 32, 346, foreground)
}

func stageName(stage creation.Stage) string {
	switch stage {
	case creation.StageRace:
		return "race"
	case creation.StageClass:
		return "class"
	case creation.StageAlignment:
		return "alignment"
	default:
		return ""
	}
}

func drawFrame(screen *ebiten.Image, foreground, accent color.Color) {
	for inset := 8; inset < 12; inset++ {
		for x := inset; x < logicalWidth-inset; x++ {
			screen.Set(x, inset, accent)
			screen.Set(x, logicalHeight-inset-1, accent)
		}
		for y := inset; y < logicalHeight-inset; y++ {
			screen.Set(inset, y, accent)
			screen.Set(logicalWidth-inset-1, y, accent)
		}
	}
	drawText(screen, "SSI GOLD BOX / POOL REMAKE", 18, 26, foreground)
}

func drawHelp(screen *ebiten.Image, background, foreground, accent color.Color) {
	for y := 54; y < 340; y++ {
		for x := 72; x < 568; x++ {
			screen.Set(x, y, background)
		}
	}
	drawText(screen, "HELP", 292, 80, accent)
	lines := []string{
		"UP/DOWN or HOME/END: choose an item",
		"ENTER: accept the selected item",
		"ESC: return to the previous screen",
		"R: reroll on the character sheet",
		"F2: switch original/modern presentation",
		"F10: quit the current prototype",
	}
	for index, line := range lines {
		drawText(screen, line, 104, 120+index*30, foreground)
	}
}

func drawText(screen *ebiten.Image, value string, x, y int, ink color.Color) {
	text.Draw(screen, strings.ToUpper(value), basicfont.Face7x13, x, y, ink)
}

func (a *app) Layout(_, _ int) (int, int) { return logicalWidth, logicalHeight }

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP used as local asset source")
	flag.Parse()
	game, err := newApp(*zipPath)
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(960, 600)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Pool of Radiance Remake")
	if err := ebiten.RunGame(game); err != nil && err != ebiten.Termination {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
