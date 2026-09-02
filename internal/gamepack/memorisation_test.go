package gamepack

import (
	"os"
	"path/filepath"
	"testing"
)

// 十個預設施法者的記憶陣列都要落在自己的可記憶數之內，一個例外都不能有。
//
// 這一條同時釘住四件事：記憶陣列的位置與 1-based 編號（spec 070）、
// 參數表的職業與等級欄位（spec 074）、可記憶數表加睿智加成（spec 072），
// 以及可記憶數在記錄裡的排法（`+0B1h + 組 × 3 + 等級`）。任何一個讀錯，
// 就會有人記得比上限多。
func TestDefaultCastersFitTheirSpellSlots(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	directory := filepath.Join("..", "..", "workplace", "oracle", "dos")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Skipf("original character files are intentionally not tracked: %v", err)
	}
	casters := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".sav" {
			continue
		}
		record, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if len(record) < MemorisedSpellOffset+MemorisedSpellSlots {
			continue
		}
		memorised := record[MemorisedSpellOffset : MemorisedSpellOffset+MemorisedSpellSlots]
		used := MemorisedCounts(memorised, parameters)
		if used == (SpellSlotCounts{}) {
			continue
		}
		casters++
		maxima, err := RecordSpellSlotMaxima(record)
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		for group := range used {
			for level := range used[group] {
				if used[group][level] > maxima[group][level] {
					t.Errorf("%s 的第 %d 組第 %d 級記了 %d 個，上限只有 %d",
						entry.Name(), group, level+1, used[group][level], maxima[group][level])
				}
			}
		}
		// 記憶陣列裡的每一個編號都要查得到參數表，而且不是物品效果。
		for _, value := range memorised {
			if id := value & memorisedSpellIDMask; id != 0 {
				if int(id) >= len(parameters) {
					t.Errorf("%s 記了表外的編號 %d", entry.Name(), id)
					continue
				}
				if source := parameters[id].Source(); source != SpellSourceCleric &&
					source != SpellSourceMagicUser {
					t.Errorf("%s 記了不是法術的編號 %d（來源 %d）", entry.Name(), id, source)
				}
			}
		}
	}
	if casters < 10 {
		t.Fatalf("只檢查到 %d 個施法者，預設人物裡有十個——掃描面漏了", casters)
	}
}

// 記滿了就記不進去；空格清掉之後又記得進去。
func TestMemoriseRespectsTheSlotLimit(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 編號 15 是 Magic Missile：法師第 1 級。
	const magicMissile = 15
	if parameters[magicMissile].Source() != SpellSourceMagicUser ||
		parameters[magicMissile].Level() != 1 {
		t.Fatalf("編號 %d 不是法師第 1 級，參數是 %+v", magicMissile, parameters[magicMissile])
	}
	var maxima SpellSlotCounts
	maxima[SpellSlotGroupMagicUser][0] = 2
	memorised := make([]uint8, MemorisedSpellSlots)
	for count := 0; count < 2; count++ {
		if err := Memorise(memorised, magicMissile, parameters, maxima); err != nil {
			t.Fatalf("第 %d 個就記不進去：%v", count+1, err)
		}
	}
	if err := Memorise(memorised, magicMissile, parameters, maxima); err == nil {
		t.Fatal("上限是 2，第 3 個卻記進去了")
	}
	if err := ForgetMemorised(memorised, 0); err != nil {
		t.Fatal(err)
	}
	if err := Memorise(memorised, magicMissile, parameters, maxima); err != nil {
		t.Fatalf("清掉一格之後應該記得進去：%v", err)
	}
	// 牧師的法術不能佔法師的格子。
	const bless = 1
	if parameters[bless].Source() != SpellSourceCleric {
		t.Fatalf("編號 %d 不是牧師法術", bless)
	}
	if err := Memorise(memorised, bless, parameters, maxima); err == nil {
		t.Fatal("法師的格子不該收得下牧師法術")
	}
}

// 從職業等級與睿智算出來的可記憶數，要和角色記錄裡已經算好的六格一模一樣。
// 這是把 spec 072 的兩張表加睿智加成拿去對原版寫下的結果。
func TestComputedSpellSlotsMatchTheRecords(t *testing.T) {
	tables, err := ReadDOSSpellSlotTableSet(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	directory := filepath.Join("..", "..", "workplace", "oracle", "dos")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Skipf("original character files are intentionally not tracked: %v", err)
	}
	checked := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".sav" {
			continue
		}
		record, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if len(record) < 285 {
			continue
		}
		stored, err := RecordSpellSlotMaxima(record)
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		if stored == (SpellSlotCounts{}) {
			continue
		}
		checked++
		levels, err := ClassLevels(record)
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		computed := tables.SpellSlotMaxima(int(levels[ClassSlotCleric]),
			int(levels[ClassSlotMagicUser]), int(record[wisdomOffset]))
		if computed != stored {
			t.Errorf("%s 算出來是 %v，記錄裡是 %v（牧師 %d 級、法師 %d 級、睿智 %d）",
				entry.Name(), computed, stored, levels[ClassSlotCleric],
				levels[ClassSlotMagicUser], record[wisdomOffset])
		}
	}
	if checked < 10 {
		t.Fatalf("只對到 %d 個施法者，預設人物裡有十個——掃描面漏了", checked)
	}
}

// 選好的法術要休息過才施得出來：Memorise 設第 7 位，休息把它清掉。
func TestMemorisationNeedsRest(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	var maxima SpellSlotCounts
	maxima[SpellSlotGroupMagicUser][0] = 1 // 魔法飛彈：法師第 1 級
	maxima[SpellSlotGroupMagicUser][2] = 1 // 火球術：法師第 3 級
	memorised := make([]uint8, MemorisedSpellSlots)
	if err := Memorise(memorised, SpellIDMagicMissile, parameters, maxima); err != nil {
		t.Fatal(err)
	}
	if err := Memorise(memorised, SpellIDFireball, parameters, maxima); err != nil {
		t.Fatal(err)
	}
	for _, value := range memorised {
		if value != 0 && MemorisedSpellIsReady(value) {
			t.Fatalf("剛選好的 %#02x 不該是可施展的狀態", value)
		}
	}
	// 時間是各法術等級的總和：第 1 級加第 3 級。
	if got := PendingMemorisationTime(memorised, parameters); got != 4 {
		t.Errorf("待記時間應該是 1 加 3 ＝ 4，算出 %d", got)
	}
	if done := CompletePendingMemorisation(memorised); done != 2 {
		t.Errorf("應該記完兩條，記完 %d 條", done)
	}
	ready := 0
	for _, value := range memorised {
		if MemorisedSpellIsReady(value) {
			ready++
		}
	}
	if ready != 2 {
		t.Errorf("休息完應該有兩條可以施，只有 %d 條", ready)
	}
	if got := PendingMemorisationTime(memorised, parameters); got != 0 {
		t.Errorf("記完之後不該還有待記時間，算出 %d", got)
	}
	// 記完之後編號要還原得回去，不能被旗標污染。
	if SearchMemorisedSpell(memorised, SpellIDFireball) == SpellSearchNotFound {
		t.Error("記完之後查不到火球術")
	}
}
