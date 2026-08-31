package main

import (
	"path/filepath"
	"testing"
)

func TestBuildReportAnchorsEmptyMovedCell(t *testing.T) {
	report, err := buildReport(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if report.Schema != "pool-initial-cell-sweep/v2" || len(report.Samples) != 16*16*4 {
		t.Fatalf("schema=%q samples=%d", report.Schema, len(report.Samples))
	}
	var found bool
	for _, sample := range report.Samples {
		if sample.X == 1 && sample.Y == 4 && sample.Facing == 2 {
			found = true
			if sample.PerTurn.Boundary != "exit" || sample.PerTurn.StopPC != "997E" || sample.PerTurn.Steps != 15 || len(sample.PerTurn.Events) != 0 || sample.Search == nil {
				t.Fatalf("moved-cell anchor: %+v", sample)
			}
		}
	}
	if !found {
		t.Fatal("moved-cell anchor is absent")
	}
}

func TestGeometryDistancesWrapAndRespectWalls(t *testing.T) {
	report, err := buildReport(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	var start, east *sample
	for index := range report.Samples {
		item := &report.Samples[index]
		if item.X == 0 && item.Y == 4 && item.Facing == 0 {
			start = item
		}
		if item.X == 1 && item.Y == 4 && item.Facing == 2 {
			east = item
		}
	}
	if start == nil || !start.GeometryReachable || start.GeometryDistance != 0 {
		t.Fatalf("start=%+v", start)
	}
	if east == nil || !east.GeometryReachable || east.GeometryDistance != 1 {
		t.Fatalf("east=%+v", east)
	}
}

func TestSweepSeparatesPresentationAndPlayerBoundaries(t *testing.T) {
	report, err := buildReport(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	counts := map[string]int{}
	var sune *sample
	for index := range report.Samples {
		item := &report.Samples[index]
		counts[item.Boundary]++
		if item.X == 1 && item.Y == 3 && item.Facing == 0 {
			sune = item
		}
	}
	if counts["exit"] != 840 || counts["event"] != 156 || counts["error"] != 28 {
		t.Fatalf("boundary counts=%v", counts)
	}
	if sune == nil || !sune.GeometryReachable || sune.GeometryDistance != 2 || sune.Boundary != "event" || len(sune.Events) != 1 || sune.Events[0].Text != "YOU ARE WELCOMED BY PRIESTESS JOY OF SUNE." {
		t.Fatalf("Sune anchor=%+v", sune)
	}
	if sune.Search == nil || len(sune.Search.Observed) != 3 {
		t.Fatalf("Sune presentation sequence=%+v", sune.Search)
	}
}
