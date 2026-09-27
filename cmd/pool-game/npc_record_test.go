package main

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// npcCombatReceipt 是 tools/dosgolem-npc-combat-runtime.py 的產物（spec 154）。
type npcCombatReceipt struct {
	Generator string         `json:"generator"`
	Offer     string         `json:"offer"`
	Joined    npcRuntimeCard `json:"joined"`
	Combat    []npcRuntimeCard
}

type npcRuntimeCard struct {
	Name          string `json:"name"`
	Record        string `json:"record_hex"`
	RuntimeLimit  *uint8 `json:"runtime_sweep_limit_5"`
	SweepLimit6Bh uint8  `json:"sweep_limit_6Bh"`
}

func readNPCCombatReceipt(t *testing.T) npcCombatReceipt {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-npc-combat-runtime.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt npcCombatReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Generator != "dosgolem" || receipt.Joined.Name != "WARRIOR" {
		t.Fatalf("receipt from %q hired %q", receipt.Generator, receipt.Joined.Name)
	}
	return receipt
}

// hireMercenary 跑訓練所競技場那一條 `ADD NPC @6E79 @6E7A`（ecl3/11 `9F1Ch`）：
// 雇傭兵的 MONnCHA 區塊與已經算好的士氣（`等級 × 5 + 50`）放進兩個變數。
func hireMercenary(t *testing.T, block uint8, morale uint16) *app {
	t.Helper()
	application, err := newApp(dosZIPForTests, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(3)
	if !ok {
		t.Fatal("ECL3 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 11, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 3
	application.spawn = gamepack.Spawn{Map: gamepack.MapKey{Archive: 3, BlockID: 11}, X: 7, Y: 2, Facing: 2}
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.state.Party = []poolsave.Character{{Name: "HERO", RaceID: "dwarf", ClassID: "fighter"}}
	session.Machine().Memory[0x6E79] = uint16(block)
	session.Machine().Memory[0x6E7A] = morale
	const pc = 0x9F1C - 0x9900
	instruction, err := application.eclInstruction(pc)
	if err != nil || instruction.Command.Opcode != gamepack.AddNPCOpcode {
		t.Fatalf("ECL3/b11 @9F1Ch is %+v (%v), want ADD NPC", instruction, err)
	}
	if err := application.applyAddNPC(eclvm.Event{PC: pc, Opcode: gamepack.AddNPCOpcode}); err != nil {
		t.Fatal(err)
	}
	if len(application.state.Party) != 2 || !application.state.Party[1].NPC {
		t.Fatalf("party after ADD NPC: %+v", application.state.Party)
	}
	return application
}

// ADD NPC 當下的記錄逐位元組對原版（dosgolem 雇 WARRIOR 那一刻，spec 154）。
// 不比的只有原版執行期才有的東西：遠指標、亂數（`+ABh`、戰鬥造形 `+BBh`／`+BCh`）、
// 隊伍順序 `+BFh` 與造形配色 `+C0h..+C7h`。
func TestAddNPCRecordMatchesTheHireRuntimeReceipt(t *testing.T) {
	receipt := readNPCCombatReceipt(t)
	original, err := hex.DecodeString(receipt.Joined.Record)
	if err != nil || len(original) != poolsave.NPCRecordSize {
		t.Fatalf("receipt record: %d bytes, %v", len(original), err)
	}
	// 開價是 1 份（等級 1）：士氣 1 × 5 + 50。
	a := hireMercenary(t, 0x67, 55)
	member := a.state.Party[1]
	runtimeOnly := func(offset int) bool {
		switch {
		case offset == 0xAB, offset == 0xBB, offset == 0xBC:
			return true // 1250h 的骰(1,100h)、138Ah／1363h 的造形
		case offset >= 0xBF && offset <= 0xC7:
			return true // 13EDh 的隊伍順序與 0477h 的造形配色
		case offset >= 0xC8 && offset <= 0xFF, offset >= 0x104 && offset <= 0x10B:
			return true // +C8h 物品、+CCh.. 槽、+104h 串列、+108h runtime 的遠指標
		}
		return false
	}
	compared := 0
	for offset := range original {
		if runtimeOnly(offset) {
			continue
		}
		compared++
		if member.Record[offset] != original[offset] {
			t.Errorf("+%03Xh: remake %02X, original %02X", offset, member.Record[offset], original[offset])
		}
	}
	if compared != 209 {
		t.Fatalf("compared only %d bytes", compared)
	}
	// 負對照：樣板的 `+2Dh`（40）與 `+110h`（149）都不是原版加入之後的值。
	template, err := a.loadMonster(3, 0x67)
	if err != nil {
		t.Fatal(err)
	}
	if template.Raw[gamepack.BaseThac0Offset] == original[gamepack.BaseThac0Offset] ||
		template.Raw[gamepack.CurrentThac0Offset] == original[gamepack.CurrentThac0Offset] {
		t.Fatal("the template already matches the runtime record; the receipt proves nothing")
	}
	// `1Dh PARTYSTRENGTH` 讀的就是這兩格。
	strength, err := a.partyStrengthRecord(member)
	if err != nil {
		t.Fatal(err)
	}
	if strength.Field110 != original[0x110] || strength.Field111 != original[0x111] {
		t.Errorf("party strength reads %d/%d, original record %d/%d",
			strength.Field110, strength.Field111, original[0x110], original[0x111])
	}
}

// #107 之前的存檔：NPC 記錄還是樣板。讀檔補成加入時的值，隊伍強度也照同一支現算。
func TestOldNPCRecordIsBroughtUpToTheJoinValues(t *testing.T) {
	receipt := readNPCCombatReceipt(t)
	original, _ := hex.DecodeString(receipt.Joined.Record)
	a := hireMercenary(t, 0x67, 55)
	joined := a.state.Party[1]
	template, err := a.loadMonster(3, 0x67)
	if err != nil {
		t.Fatal(err)
	}
	old := joined
	old.Record = append([]byte(nil), template.Raw[:]...)
	old.Record[gamepack.MoraleOffset] = joined.Record[gamepack.MoraleOffset]
	strength, err := a.partyStrengthRecord(old)
	if err != nil {
		t.Fatal(err)
	}
	// `+110h` 由 entry 7 現算，但 `+2Dh` 還是樣板的 40：這一格要等讀檔補上。
	if strength.Field111 != original[0x111] {
		t.Errorf("old record strength AC %d, want %d", strength.Field111, original[0x111])
	}
	a.state.Party[1] = old
	a.migrateNPCMorale()
	for _, offset := range []int{gamepack.BaseThac0Offset, 0x110, 0x111, 0x112, 0x11C, 0x102, 0x103, gamepack.SweepLimitOffset, 0x10E} {
		if got := a.state.Party[1].Record[offset]; got != original[offset] {
			t.Errorf("migrated +%03Xh = %02X, original %02X", offset, got, original[offset])
		}
	}
}

// 原版開打那一幀的每一筆 combatant（隊員 HERO、NPC WARRIOR、衛兵與他們的幫手）：
// `+6Bh` 都是 entry 7 尾段那一條，runtime `+5` 要不是它、就是出過手之後的 0。
func TestSweepLimitMatchesEveryCombatantInTheRuntimeReceipt(t *testing.T) {
	receipt := readNPCCombatReceipt(t)
	if len(receipt.Combat) < 10 {
		t.Fatalf("receipt has %d combatants", len(receipt.Combat))
	}
	struck := 0
	for index, card := range receipt.Combat {
		raw, err := hex.DecodeString(card.Record)
		if err != nil {
			t.Fatal(err)
		}
		want := gamepack.SweepLimit(raw[gamepack.ClassLevelOffset+gamepack.ClassSlotFighter], raw[gamepack.RaceOffset])
		if raw[gamepack.SweepLimitOffset] != want {
			t.Errorf("combatant %d %s: +6Bh %d, rule %d", index, card.Name, raw[gamepack.SweepLimitOffset], want)
		}
		if card.RuntimeLimit == nil {
			t.Fatalf("combatant %d %s has no runtime", index, card.Name)
		}
		switch *card.RuntimeLimit {
		case want:
		case 0:
			struck++
		default:
			t.Errorf("combatant %d %s: runtime +5 = %d, +6Bh %d", index, card.Name, *card.RuntimeLimit, want)
		}
	}
	// 規則要分得出來：這一場有戰士 6 級、3 級，也有上限 1 的。
	seen := map[uint8]bool{}
	for _, card := range receipt.Combat {
		seen[card.SweepLimit6Bh] = true
	}
	if !seen[1] || !seen[3] || !seen[6] {
		t.Fatalf("receipt limits %v do not exercise the rule", seen)
	}
	if struck > 1 {
		t.Fatalf("%d combatants already struck at the first frame", struck)
	}
}

// MONnSPC 的效果串列跟著 ADD NPC 進來（`0E90h` 的 `1051h`）。八個呼叫點的樣板在資料裡
// 都沒有效果節點，所以這裡用換過的載入器驗接線。
func TestAddNPCCarriesTheMonsterEffectList(t *testing.T) {
	a := hireMercenary(t, 0x67, 55)
	if len(a.state.Party[1].Effects) != 0 {
		t.Fatalf("WARRIOR joined with effects %+v; MON3SPC.DAX does not exist", a.state.Party[1].Effects)
	}
	for _, source := range gamepack.NPCMoraleSources() {
		list, err := a.loadMonsterEffects(source.Archive, source.Block)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 0 {
			t.Errorf("NPC source %d/%02X has %d effect nodes; the test above no longer covers the data", source.Archive, source.Block, len(list))
		}
	}
	node := gamepack.NewEffectNode(0x3B, 0, 0, false)
	a.state.Party = a.state.Party[:1]
	a.loadMonsterEffects = func(archive, block uint8) (gamepack.EffectList, error) {
		return gamepack.EffectList{node}, nil
	}
	const pc = 0x9F1C - 0x9900
	if err := a.applyAddNPC(eclvm.Event{PC: pc, Opcode: gamepack.AddNPCOpcode}); err != nil {
		t.Fatal(err)
	}
	effects := a.state.Party[1].Effects
	if len(effects) != 1 || effects[0].Code != node.Code || effects[0].Payload != node.Payload {
		t.Fatalf("NPC effects %+v, want the MONnSPC node", effects)
	}
}
