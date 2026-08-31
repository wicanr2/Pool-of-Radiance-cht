// Command pool-initial-cell-sweep executes the original initial-map cell
// lifecycle entry against isolated copies of the post-Rolf VM state.
//
// This is an entry sweep, not proof that a normal player can reach every cell.
// Geometry-only reachability is reported separately for that reason.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

const codeBase = 0x9900

type report struct {
	Schema       string   `json:"schema"`
	Input        string   `json:"input"`
	InputSHA256  string   `json:"input_sha256"`
	Map          mapKey   `json:"map"`
	EntryAddress string   `json:"entry_address"`
	Start        point    `json:"geometry_reachability_start"`
	Scope        string   `json:"scope"`
	Samples      []sample `json:"samples"`
}

type mapKey struct {
	Archive uint8 `json:"archive"`
	BlockID uint8 `json:"block_id"`
}

type point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type sample struct {
	X                 int          `json:"x"`
	Y                 int          `json:"y"`
	Facing            int          `json:"facing"`
	Terrain           uint8        `json:"terrain"`
	FacingWall        uint8        `json:"facing_wall"`
	GeometryReachable bool         `json:"geometry_reachable"`
	GeometryDistance  int          `json:"geometry_distance,omitempty"`
	Boundary          string       `json:"boundary"`
	StopPC            string       `json:"stop_pc"`
	Steps             int          `json:"steps"`
	Events            []eventEntry `json:"events,omitempty"`
	Menus             int          `json:"menus,omitempty"`
	Error             string       `json:"error,omitempty"`
	PerTurn           phase        `json:"per_turn"`
	Search            *phase       `json:"search,omitempty"`
}

type phase struct {
	Entry    string       `json:"entry"`
	Boundary string       `json:"boundary"`
	StopPC   string       `json:"stop_pc"`
	Steps    int          `json:"steps"`
	Observed []eventEntry `json:"observed_presentation_events,omitempty"`
	Events   []eventEntry `json:"events,omitempty"`
	Menus    int          `json:"menus,omitempty"`
	Error    string       `json:"error,omitempty"`
}

type eventEntry struct {
	PC      string `json:"pc"`
	Opcode  string `json:"opcode"`
	Text    string `json:"text,omitempty"`
	Value   uint16 `json:"value,omitempty"`
	Address string `json:"address,omitempty"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "original DOS ZIP")
	output := flag.String("output", "", "write JSON to this file instead of stdout")
	flag.Parse()

	report, err := buildReport(*zipPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if *output == "" {
		_, err = os.Stdout.Write(data)
	} else {
		err = os.WriteFile(*output, data, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildReport(zipPath string) (report, error) {
	digest, err := fileSHA256(zipPath)
	if err != nil {
		return report{}, err
	}
	event, err := gamepack.ReadDOSInitialEvent(zipPath)
	if err != nil {
		return report{}, err
	}
	base, err := gamepack.NewInitialEventMachine(event)
	if err != nil {
		return report{}, err
	}
	if err := finishRolf(base); err != nil {
		return report{}, err
	}
	catalog, err := gamepack.ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		return report{}, err
	}
	key := gamepack.MapKey{Archive: 3, BlockID: 0}
	initial, ok := catalog.Map(key)
	if !ok {
		return report{}, fmt.Errorf("initial GEO3/block0 is absent")
	}
	distances := geometryDistances(initial.Grid, 0, 4)

	result := report{
		Schema:       "pool-initial-cell-sweep/v2",
		Input:        zipPath,
		InputSHA256:  digest,
		Map:          mapKey{Archive: key.Archive, BlockID: key.BlockID},
		EntryAddress: "9914",
		Start:        point{X: 0, Y: 4},
		Scope:        "isolated post-Rolf execution of entry 0 (per-turn) followed by entry 1 (SearchLocation); five-entry roles are cross-title strong inference pending Pool executable confirmation; geometry reachability is separate and does not execute intervening ECL events",
		Samples:      make([]sample, 0, geometry.Width*geometry.Height*4),
	}
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
			cell := initial.Grid.CellWrapped(x, y)
			distance, reachable := distances[point{X: x, Y: y}]
			for _, facing := range []int{0, 2, 4, 6} {
				wall, _ := initial.Grid.WallWrapped(x, y, facing)
				entry := sample{X: x, Y: y, Facing: facing, Terrain: cell.Terrain, FacingWall: wall, GeometryReachable: reachable, GeometryDistance: distance}
				machine := base.Clone()
				run, runErr := gamepack.RunInitialCellEntry(machine, initial.Grid, gamepack.Spawn{Map: key, X: uint8(x), Y: uint8(y), Facing: uint8(facing)})
				entry.PerTurn = summarizePhase("9914", run, runErr)
				final := entry.PerTurn
				if runErr == nil && run.Exited {
					if err := machine.SetPC(0x99EB - codeBase); err != nil {
						return report{}, err
					}
					search := runSearchAudit(machine)
					entry.Search = &search
					final = search
				}
				entry.Boundary, entry.StopPC, entry.Steps = final.Boundary, final.StopPC, entry.PerTurn.Steps
				if entry.Search != nil {
					entry.Steps += entry.Search.Steps
				}
				entry.Events, entry.Menus, entry.Error = final.Events, final.Menus, final.Error
				result.Samples = append(result.Samples, entry)
			}
		}
	}
	return result, nil
}

func summarizePhase(entry string, run eclvm.Result, runErr error) phase {
	result := phase{Entry: entry, StopPC: fmt.Sprintf("%04X", run.PC+codeBase), Steps: run.Steps, Menus: len(run.Menus)}
	for _, observed := range run.Events {
		item := eventEntry{PC: fmt.Sprintf("%04X", observed.PC+codeBase), Opcode: fmt.Sprintf("%02X", observed.Opcode), Text: observed.Text, Value: observed.Value}
		if observed.Address != 0 {
			item.Address = fmt.Sprintf("%04X", observed.Address)
		}
		result.Events = append(result.Events, item)
	}
	switch {
	case runErr != nil:
		result.Boundary = "error"
		result.Error = runErr.Error()
	case run.WaitingForMenu:
		result.Boundary = "menu"
	case len(run.Events) != 0:
		result.Boundary = "event"
	case run.Exited:
		result.Boundary = "exit"
	default:
		result.Boundary = "step_limit"
	}
	return result
}

func runSearchAudit(machine *eclvm.Machine) phase {
	result := phase{Entry: "99EB"}
	for boundaries := 0; boundaries < 64; boundaries++ {
		run, err := machine.RunUntilEvent(4096, nil, true)
		result.Steps += run.Steps
		current := summarizePhase("99EB", run, err)
		if err == nil && len(run.Events) == 1 && isPresentationOnly(run.Events[0]) {
			result.Observed = append(result.Observed, current.Events...)
			continue
		}
		result.Boundary = current.Boundary
		result.StopPC = current.StopPC
		result.Events = current.Events
		result.Menus = current.Menus
		result.Error = current.Error
		return result
	}
	result.Boundary = "boundary_limit"
	result.StopPC = fmt.Sprintf("%04X", machine.PC+codeBase)
	result.Error = "SearchLocation exceeded presentation boundary limit"
	return result
}

func isPresentationOnly(event eclvm.Event) bool {
	return (event.Opcode == 0x12 && event.Text == "") || event.Opcode == 0x0E
}

func finishRolf(machine *eclvm.Machine) error {
	run, err := machine.Run(2000, nil, true)
	if err != nil {
		return err
	}
	for menus := 0; !run.Exited; menus++ {
		if !run.WaitingForMenu {
			return fmt.Errorf("Rolf session stopped without menu or EXIT at %04X", run.PC+codeBase)
		}
		if menus > 12 {
			return fmt.Errorf("Rolf session exceeded menu limit")
		}
		run, err = machine.Run(4000, []uint16{0}, true)
		if err != nil {
			return err
		}
	}
	if machine.Memory[0xC04B] != 0 || machine.Memory[0xC04C] != 4 || machine.Memory[0xC04D] != 3 {
		return fmt.Errorf("Rolf final position is (%d,%d,%d), want (0,4,3)", machine.Memory[0xC04B], machine.Memory[0xC04C], machine.Memory[0xC04D])
	}
	return nil
}

func geometryDistances(grid geometry.Grid, startX, startY int) map[point]int {
	start := point{X: geometry.WrapCoordinate(startX, geometry.Width), Y: geometry.WrapCoordinate(startY, geometry.Height)}
	distances := map[point]int{start: 0}
	queue := []point{start}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		for _, direction := range []int{0, 2, 4, 6} {
			if !grid.CanMoveDungeonWrapped(current.X, current.Y, direction) {
				continue
			}
			next := current
			switch direction {
			case 0:
				next.Y--
			case 2:
				next.X++
			case 4:
				next.Y++
			case 6:
				next.X--
			}
			next.X = geometry.WrapCoordinate(next.X, geometry.Width)
			next.Y = geometry.WrapCoordinate(next.Y, geometry.Height)
			if _, seen := distances[next]; seen {
				continue
			}
			distances[next] = distances[current] + 1
			queue = append(queue, next)
		}
	}
	return distances
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read input ZIP: %w", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
