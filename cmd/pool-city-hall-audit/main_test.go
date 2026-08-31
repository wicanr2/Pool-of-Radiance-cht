package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func loadTrace(t *testing.T) trace {
	t.Helper()
	b, err := os.ReadFile("../../docs/audit/dos-ecl3-block8-trace.json")
	if err != nil {
		t.Fatal(err)
	}
	var tr trace
	if err := json.Unmarshal(b, &tr); err != nil {
		t.Fatal(err)
	}
	return tr
}
func TestAuditPinsCityHallStructure(t *testing.T) {
	r, err := audit(loadTrace(t))
	if err != nil {
		t.Fatal(err)
	}
	wantReward := []string{"0xB5C4", "0xB5CC", "0xB5CC", "0xB5D7", "0xB5D7", "0xB5E2", "0xB5E2"}
	if !reflect.DeepEqual(r.RewardDispatch.Targets, wantReward) {
		t.Fatalf("reward targets=%v", r.RewardDispatch.Targets)
	}
	wantCommission := []string{"0xA8AB", "0xA8EC", "0xA91C", "0xA9B0", "0xAA31", "0xAAA8", "0xAB10", "0xAB7E", "0xABC0", "0xAC1D", "0xAC21", "0xAC58", "0xAC9A", "0xAD01", "0xAE25", "0xAEA5"}
	if !reflect.DeepEqual(r.CommissionDispatch.Targets, wantCommission) {
		t.Fatalf("commission targets=%v", r.CommissionDispatch.Targets)
	}
	wantProducers := []string{"0x9FAE", "0x9FF3", "0xA179", "0xA1BB", "0xA201", "0xA24C", "0xA304", "0xA342", "0xA37F", "0xA4D1"}
	if !reflect.DeepEqual(r.ProgressProducers, wantProducers) {
		t.Fatalf("4AC1 producers=%v", r.ProgressProducers)
	}
	if len(r.CompletionTable) != 26 || r.CompletionTable[0].StateAddress != "0x4AA6" || r.CompletionTable[0].Target != "0x9F77" || !r.CompletionTable[0].IncrementsProgress || r.CompletionTable[0].IncrementAddress != "0x9FAE" || r.CompletionTable[21].StateAddress != "0x4ABB" || r.CompletionTable[21].IncrementAddress != "0xA4D1" {
		t.Fatalf("completion table=%+v", r.CompletionTable)
	}
	increments := 0
	for _, row := range r.CompletionTable {
		if row.IncrementsProgress {
			increments++
		}
	}
	if increments != 10 {
		t.Fatalf("completion increments=%d", increments)
	}
	if len(r.ExternalCalls) < 3 {
		t.Fatalf("external calls=%v", r.ExternalCalls)
	}
}
func TestAuditRejectsMissingEvidence(t *testing.T) {
	tr := loadTrace(t)
	tr.BlockSHA256 = "changed"
	if _, err := audit(tr); err == nil {
		t.Fatal("changed source identity accepted")
	}
	tr = loadTrace(t)
	for i := range tr.Edges {
		if tr.Edges[i].From == 0xA85B-codeBase && tr.Edges[i].Kind == "ON GOSUB" {
			tr.Edges = append(tr.Edges[:i], tr.Edges[i+1:]...)
			break
		}
	}
	if _, err := audit(tr); err == nil {
		t.Fatal("missing commission route accepted")
	}
	tr = loadTrace(t)
	for i := range tr.Instructions {
		if tr.Instructions[i].Address == "0x9FAE" {
			tr.Instructions[i].Operands[0].Low = 2
			break
		}
	}
	if _, err := audit(tr); err == nil {
		t.Fatal("changed 4AC1 producer accepted")
	}
	tr = loadTrace(t)
	for i := range tr.Edges {
		if tr.Edges[i].From == 0x9D63-codeBase && tr.Edges[i].Kind == "ON GOSUB" {
			tr.Edges = append(tr.Edges[:i], tr.Edges[i+1:]...)
			break
		}
	}
	if _, err := audit(tr); err == nil {
		t.Fatal("missing completion notification route accepted")
	}
}
