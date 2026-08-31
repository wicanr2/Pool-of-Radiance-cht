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
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
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
	modeAdventure
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
	iconReady    *ebiten.Image
	iconAction   *ebiten.Image
	loadPortrait func(head, body uint8) (*ebiten.Image, error)
	loadIcon     func(head, body, size uint8, action bool, colors [6][2]uint8) (*ebiten.Image, error)
	state        poolsave.State
	saveState    func(poolsave.State) error
	loadState    func() (poolsave.State, error)
	initialMap   *gamepack.GeometryMap
	spawn        gamepack.Spawn
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
		state:  poolsave.NewState(),
	}
	catalog, err := gamepack.ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		return nil, err
	}
	application.spawn = gamepack.DOSInitialSpawn()
	initialMap, ok := catalog.Map(application.spawn.Map)
	if !ok {
		return nil, fmt.Errorf("DOS initial map GEO%d block %d is absent", application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}
	application.initialMap = &initialMap
	const statePath = "saves/pool-remake-state.json"
	application.saveState = func(state poolsave.State) error { return poolsave.WriteAtomic(statePath, state) }
	application.loadState = func() (poolsave.State, error) { return poolsave.Read(statePath) }
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
	application.loadIcon = func(head, body, size uint8, action bool, colors [6][2]uint8) (*ebiten.Image, error) {
		picture, err := assets.ReadCustomizedCombatIcon(zipPath, assets.CombatIconSelection{Head: head, Body: body, Size: size}, action, colors)
		if err != nil {
			return nil, err
		}
		rendered, err := picture.RGBA(0, graphics.EGA16)
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

func (a *app) reloadIcons() error {
	if a.loadIcon == nil {
		return fmt.Errorf("combat icon loader is not configured")
	}
	ready, err := a.loadIcon(a.flow.IconHead, a.flow.IconWeapon, a.flow.IconSize, false, a.flow.IconColors)
	if err != nil {
		return err
	}
	action, err := a.loadIcon(a.flow.IconHead, a.flow.IconWeapon, a.flow.IconSize, true, a.flow.IconColors)
	if err != nil {
		return err
	}
	a.iconReady, a.iconAction = ready, action
	return nil
}

func (a *app) Update() error {
	if a.justPressed(ebiten.KeyF10) {
		if a.saveState != nil {
			if err := a.saveState(a.state); err != nil {
				return err
			}
		}
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
		if a.justPressed(ebiten.KeyB) {
			if len(a.state.Party) == 0 {
				a.statusLine = "Add at least one character before beginning adventure."
				return nil
			}
			if a.initialMap == nil {
				return fmt.Errorf("Pool initial geometry is not configured")
			}
			a.mode = modeAdventure
			a.statusLine = "GEO preview only; press ESC to return."
			return nil
		}
		if a.justPressed(ebiten.KeyC) || a.justPressed(ebiten.KeyEnter) {
			a.flow, a.cursor, a.rolled = creation.NewFlow(), 0, nil
			a.mode = modeCreation
			return nil
		}
		if a.justPressed(ebiten.KeyA) {
			return a.addFirstLibraryCharacter()
		}
		if a.justPressed(ebiten.KeyL) {
			if a.loadState == nil {
				a.statusLine = "No save loader is configured."
				return nil
			}
			loaded, err := a.loadState()
			if err != nil {
				a.statusLine = err.Error()
				return nil
			}
			a.state = loaded
			a.statusLine = fmt.Sprintf("Loaded %d library / %d party characters.", len(loaded.CharacterLibrary), len(loaded.Party))
		}
	case modeCreation:
		return a.updateCreation()
	case modeAdventure:
		if a.justPressed(ebiten.KeyEscape) {
			a.mode = modeMenu
			a.statusLine = "Returned from the geometry preview."
		}
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
		a.cursor = 0
		if a.flow.Stage <= creation.StageAlignment {
			a.rolled = nil
		}
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
			a.statusLine = "Portrait accepted."
			return a.reloadIcons()
		}
		return nil
	}
	if a.flow.Stage == creation.StageIcon {
		changed := false
		if a.justPressed(ebiten.KeyH) {
			if err := a.flow.NextIconHead(); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyW) {
			if err := a.flow.NextIconWeapon(); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyP) {
			if err := a.flow.SelectNextIconPart(); err != nil {
				return err
			}
		}
		if a.justPressed(ebiten.KeyDigit1) {
			if err := a.flow.NextIconColor(0); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyDigit2) {
			if err := a.flow.NextIconColor(1); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyS) {
			if err := a.flow.ToggleIconSize(); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeyE) {
			return a.flow.RequestIconConfirmation()
		}
		if changed {
			return a.reloadIcons()
		}
		return nil
	}
	if a.flow.Stage == creation.StageIconConfirm {
		if a.justPressed(ebiten.KeyN) {
			return a.flow.RejectIconConfirmation()
		}
		if a.justPressed(ebiten.KeyY) || a.justPressed(ebiten.KeyEnter) {
			return a.finishCharacter()
		}
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

func (a *app) finishCharacter() error {
	if a.rolled == nil {
		return fmt.Errorf("Pool character confirmation has no rolled character")
	}
	for _, existing := range a.state.CharacterLibrary {
		if existing.Name == a.flow.Name {
			a.statusLine = "A character with that name already exists."
			return nil
		}
	}
	rolled := a.rolled
	character := poolsave.Character{
		Name: a.flow.Name, RaceID: a.flow.SelectedRace().ID, GenderID: a.flow.SelectedGender().ID,
		ClassID: a.flow.SelectedClass().ID, AlignmentID: a.flow.SelectedAlignment().ID,
		Age: rolled.Age, Abilities: rolled.Abilities, ExceptionalStrength: rolled.ExceptionalStrength,
		Gold: rolled.Gold, HP: rolled.HP, RawHP: rolled.RawHP,
		PortraitHead: a.flow.PortraitHead, PortraitBody: a.flow.PortraitBody,
		IconHead: a.flow.IconHead, IconWeapon: a.flow.IconWeapon, IconSize: a.flow.IconSize, IconColors: a.flow.IconColors,
	}
	a.state.CharacterLibrary = append(a.state.CharacterLibrary, character)
	if a.saveState != nil {
		if err := a.saveState(a.state); err != nil {
			a.state.CharacterLibrary = a.state.CharacterLibrary[:len(a.state.CharacterLibrary)-1]
			a.statusLine = err.Error()
			return nil
		}
	}
	a.mode, a.flow, a.cursor, a.rolled = modeMenu, creation.NewFlow(), 0, nil
	a.portrait, a.iconReady, a.iconAction = nil, nil, nil
	a.statusLine = character.Name + " saved to the character library."
	return nil
}

func (a *app) addFirstLibraryCharacter() error {
	if len(a.state.Party) >= 6 {
		a.statusLine = "The party already has six characters."
		return nil
	}
	for _, candidate := range a.state.CharacterLibrary {
		present := false
		for _, member := range a.state.Party {
			if member.Name == candidate.Name {
				present = true
				break
			}
		}
		if present {
			continue
		}
		a.state.Party = append(a.state.Party, candidate)
		if a.saveState != nil {
			if err := a.saveState(a.state); err != nil {
				a.state.Party = a.state.Party[:len(a.state.Party)-1]
				a.statusLine = err.Error()
				return nil
			}
		}
		a.statusLine = candidate.Name + " added to the party."
		return nil
	}
	a.statusLine = "No unassigned character is available."
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
		drawText(screen, "PARTY CREATION MENU", 224, 54, accent)
		drawText(screen, "C  CREATE NEW CHARACTER", 176, 112, foreground)
		drawText(screen, "A  ADD CHARACTER TO PARTY", 176, 140, foreground)
		drawText(screen, "L  LOAD SAVED GAME", 176, 168, foreground)
		drawText(screen, "B  BEGIN ADVENTURING", 176, 196, foreground)
		drawText(screen, fmt.Sprintf("LIBRARY %d   PARTY %d/6", len(a.state.CharacterLibrary), len(a.state.Party)), 176, 230, accent)
		for index, member := range a.state.Party {
			drawText(screen, fmt.Sprintf("%d  %s", index+1, member.Name), 176, 254+index*18, foreground)
		}
		if a.statusLine != "" {
			drawText(screen, a.statusLine, 72, 350, foreground)
		}
	} else if a.mode == modeCreation {
		drawCreation(screen, a, foreground, accent)
	} else {
		drawAdventure(screen, a, foreground, accent)
	}
	drawText(screen, "F1 Help  F2 Theme  ESC Back  F10 Quit", 16, 390, foreground)
	if a.help {
		drawHelp(screen, background, foreground, accent)
	}
}

func drawAdventure(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	drawFrame(screen, foreground, accent)
	drawText(screen, "INITIAL DOS GEOMETRY PREVIEW", 190, 52, accent)
	if a.initialMap == nil {
		drawText(screen, "INITIAL MAP IS NOT LOADED", 184, 190, foreground)
		return
	}
	const cellSize = 14
	const originX = 42
	const originY = 82
	grid := a.initialMap.Grid
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			cell := grid.Cells[y][x]
			shade := uint8(18 + (int(cell.Terrain)&7)*8)
			for py := 1; py < cellSize; py++ {
				for px := 1; px < cellSize; px++ {
					screen.Set(originX+x*cellSize+px, originY+y*cellSize+py, color.RGBA{shade, shade, shade, 255})
				}
			}
			left, top := originX+x*cellSize, originY+y*cellSize
			if cell.WallDirections[0] != 0 {
				for px := 0; px <= cellSize; px++ {
					screen.Set(left+px, top, accent)
				}
			}
			if cell.WallDirections[1] != 0 {
				for py := 0; py <= cellSize; py++ {
					screen.Set(left+cellSize, top+py, accent)
				}
			}
			if cell.WallDirections[2] != 0 {
				for px := 0; px <= cellSize; px++ {
					screen.Set(left+px, top+cellSize, accent)
				}
			}
			if cell.WallDirections[3] != 0 {
				for py := 0; py <= cellSize; py++ {
					screen.Set(left, top+py, accent)
				}
			}
		}
	}
	markerX := originX + int(a.spawn.X)*cellSize + cellSize/2
	markerY := originY + int(a.spawn.Y)*cellSize + cellSize/2
	for delta := -3; delta <= 3; delta++ {
		screen.Set(markerX+delta, markerY, color.White)
		screen.Set(markerX, markerY+delta, color.White)
	}
	drawText(screen, fmt.Sprintf("GEO%d BLOCK %d", a.spawn.Map.Archive, a.spawn.Map.BlockID), 310, 106, foreground)
	drawText(screen, fmt.Sprintf("X %d  Y %d  FACING %d", a.spawn.X, a.spawn.Y, a.spawn.Facing), 310, 136, foreground)
	drawText(screen, "DATA IDENTITY: EXACT", 310, 184, accent)
	drawText(screen, "VIEW: DIAGNOSTIC TOP-DOWN", 310, 210, accent)
	drawText(screen, "WALL ART / MOVE POLICY: PENDING", 310, 246, foreground)
	drawText(screen, "FIRST EVENT: PENDING", 310, 272, foreground)
	drawText(screen, "ESC: PARTY CREATION MENU", 310, 308, foreground)
	if a.statusLine != "" {
		drawText(screen, a.statusLine, 42, 342, foreground)
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
		parts := []string{"BODY", "ARM", "LEG", "HAIR/FACE", "SHIELD", "WEAPON"}
		drawText(screen, fmt.Sprintf("H HEAD %02d/13   W WEAPON %02d/31", a.flow.IconHead, a.flow.IconWeapon), 48, 116, foreground)
		drawText(screen, fmt.Sprintf("P PART %-9s  1 COLOR-1 %X  2 COLOR-2 %X", parts[a.flow.IconPart], a.flow.IconColors[a.flow.IconPart][0], a.flow.IconColors[a.flow.IconPart][1]), 48, 146, foreground)
		size := "LARGE"
		if a.flow.IconSize == 1 {
			size = "SMALL"
		}
		drawText(screen, "S SIZE "+size, 48, 176, foreground)
		drawText(screen, "READY", 356, 118, accent)
		drawText(screen, "ACTION", 472, 118, accent)
		for index, icon := range []*ebiten.Image{a.iconReady, a.iconAction} {
			if icon == nil {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(4, 4)
			op.GeoM.Translate(float64(340+index*116), 142)
			screen.DrawImage(icon, op)
		}
		drawText(screen, "ALL DOS OPTIONS ARE KEPT; DIRECT KEYS GUIDE THIS FIRST SLICE.", 48, 298, foreground)
		drawText(screen, creation.HintFor("icon"), 48, 332, foreground)
		return
	}
	if a.flow.Stage == creation.StageIconConfirm {
		drawText(screen, "IS THIS ICON OK?", 230, 138, accent)
		drawText(screen, "Y / ENTER  YES", 230, 190, foreground)
		drawText(screen, "N          NO", 230, 222, foreground)
		if a.statusLine != "" {
			drawText(screen, a.statusLine, 72, 310, foreground)
		}
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
		"B: begin adventure after adding a party member",
		"F10: save the remake state and quit",
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
