package journal_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 附錄 4「昇級所需經驗表」印的是原版自己的表。
//
// 前面幾則檢查的是轉錄跟**掃描**、跟**遊戲內建語料**對不對得起來；那兩份都是
// 抄本，一起抄錯就一起錯。數字表不一樣：原版執行檔裡就有這張表
// （`START.EXE` 的 `DS:4013h`，spec 071），它是抄本以外的第三份來源。
// OCR 最容易在數字上出錯，而數字錯了讀起來一樣通順。
var experienceClassHeading = regexp.MustCompile(`(?m)^\*\*[A-D] .*（(Cleric|Fighter|Magic-user|Thief)）\*\*`)

// experienceRow 認得兩種列：戰士與小偷只有等級與經驗值，牧師與法師後面
// 還有三個法術等級欄。
var experienceRow = regexp.MustCompile(`(?m)^\| *(\d+) *\| *(\d+)～(\d+) *\|(.*)$`)

// manualClassIndex 把書上的職業對到角色記錄 `+96h` 起的職業索引。
var manualClassIndex = map[string]int{"Cleric": 0, "Fighter": 2, "Magic-user": 5, "Thief": 6}

type experienceBand struct {
	level      int
	from, to   uint32
	spellSlots []int
	hasSlots   bool
}

// parseExperienceAppendix 從上冊轉錄切出附錄 4 的四張表。
func parseExperienceAppendix(t *testing.T) map[string][]experienceBand {
	t.Helper()
	raw, err := os.ReadFile(manualBookPath("journal-vol1.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	start := strings.Index(text, "#### 4. 昇級所需經驗表")
	if start < 0 {
		t.Fatal("找不到附錄 4")
	}
	end := strings.Index(text[start:], "#### 5. ")
	if end < 0 {
		t.Fatal("找不到附錄 5，附錄 4 的範圍切不出來")
	}
	section := text[start : start+end]

	headings := experienceClassHeading.FindAllStringSubmatchIndex(section, -1)
	if len(headings) != len(manualClassIndex) {
		t.Fatalf("附錄 4 抓到 %d 張職業表，應該是 %d 張", len(headings), len(manualClassIndex))
	}
	tables := map[string][]experienceBand{}
	for index, heading := range headings {
		class := section[heading[2]:heading[3]]
		stop := len(section)
		if index+1 < len(headings) {
			stop = headings[index+1][0]
		}
		for _, row := range experienceRow.FindAllStringSubmatch(section[heading[1]:stop], -1) {
			level, _ := strconv.Atoi(row[1])
			from, _ := strconv.ParseUint(row[2], 10, 32)
			to, _ := strconv.ParseUint(row[3], 10, 32)
			band := experienceBand{level: level, from: uint32(from), to: uint32(to)}
			for _, cell := range strings.Split(row[4], "|") {
				cell = strings.TrimSpace(cell)
				if cell == "" {
					continue
				}
				band.hasSlots = true
				if cell == "—" {
					band.spellSlots = append(band.spellSlots, 0)
					continue
				}
				count, err := strconv.Atoi(cell)
				if err != nil {
					t.Fatalf("%s 第 %d 級的法術欄是 %q，讀不成數字", class, level, cell)
				}
				band.spellSlots = append(band.spellSlots, count)
			}
			tables[class] = append(tables[class], band)
		}
	}
	return tables
}

// 附錄 4 的經驗值門檻要跟原版執行檔裡的表一致。
func TestExperienceAppendixMatchesTheGameTable(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	table, err := gamepack.ReadDOSExperienceTable(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	tables := parseExperienceAppendix(t)
	for class, bands := range tables {
		index := manualClassIndex[class]
		if got := table.MaxLevel(index); got != len(bands) {
			t.Errorf("%s：書上列了 %d 級，原版的上限是 %d 級", class, len(bands), got)
		}
		for position, band := range bands {
			if band.level != position+1 {
				t.Errorf("%s：第 %d 列印的是第 %d 級", class, position+1, band.level)
				continue
			}
			// 第 1 級從 0 開始；之後每一級的下界就是原版「升到這一級」的門檻。
			want := uint32(0)
			if band.level > 1 {
				need, ok := table.RequiredExperience(index, band.level)
				if !ok {
					t.Errorf("%s：原版沒有第 %d 級的門檻，書上卻有", class, band.level)
					continue
				}
				want = need
			}
			if band.from != want {
				t.Errorf("%s 第 %d 級：書上寫 %d 起，原版是 %d 起", class, band.level, band.from, want)
			}
			// 上界是下一級的門檻減一。最後一級沒有下一個門檻，原版也沒有
			// 印在書上的那個上限，所以那一列只驗下界。
			if next, ok := table.RequiredExperience(index, band.level+1); ok {
				if band.to != next-1 {
					t.Errorf("%s 第 %d 級：書上寫到 %d，原版的下一級門檻是 %d（應該寫 %d）",
						class, band.level, band.to, next, next-1)
				}
			}
		}
		t.Logf("%s：%d 級的經驗值門檻與原版一致", class, len(bands))
	}
}

// 附錄 4 牧師與法師那兩張表右邊的法術格數，要跟原版的格數表一致。
func TestSpellSlotAppendixMatchesTheGameTable(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	cleric, magicUser, err := gamepack.ReadDOSSpellSlotTables(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	tables := parseExperienceAppendix(t)
	for class, slots := range map[string][]gamepack.SpellSlots{"Cleric": cleric, "Magic-user": magicUser} {
		for _, band := range tables[class] {
			if !band.hasSlots {
				t.Errorf("%s 第 %d 級沒有法術欄", class, band.level)
				continue
			}
			// 第 1 級不在那張表裡：原版對等級 1 直接跳過，格數是建角時寫下的
			// `{1,0,0}`（spec 071）。書上印 1，兩邊一致。
			want := gamepack.FirstLevelSpellSlots()
			if band.level > 1 {
				if band.level >= len(slots) {
					t.Errorf("%s：原版的格數表沒有第 %d 級", class, band.level)
					continue
				}
				want = slots[band.level]
			}
			if len(band.spellSlots) != len(want) {
				t.Errorf("%s 第 %d 級：書上有 %d 個法術欄，原版是 %d 個",
					class, band.level, len(band.spellSlots), len(want))
				continue
			}
			for spellLevel, count := range band.spellSlots {
				if uint8(count) != want[spellLevel] {
					t.Errorf("%s 第 %d 級的第 %d 級法術：書上寫 %d，原版是 %d",
						class, band.level, spellLevel+1, count, want[spellLevel])
				}
			}
		}
		t.Logf("%s：法術格數與原版一致", class)
	}
}

// 附錄 7「武器一覽表」印的是每一種武器對中小型／大型目標的傷害。原版把同一
// 組數字放在 `ITEMS` 的物品型別表裡（spec 063），所以這張表也有第三份來源。
//
// 對照的是傷害骰加上固定加值：型別記錄 `+2`／`+3`／`+4` 是對大型目標的
// 骰數、面數與加值，`+9`／`+0Ah`／`+0Bh` 是對中小型目標的同三個欄位。
// 書上寫的是範圍（`2-7`），所以比對前換算成 `骰數..骰數×面數＋加值`。
const (
	itemTypeLargeCount  = 0x02
	itemTypeLargeSides  = 0x03
	itemTypeLargeBonus  = 0x04
	itemTypeSmallCount  = 0x09
	itemTypeSmallSides  = 0x0a
	itemTypeSmallBonus  = 0x0b
)

// manualWeaponNames 把書上的寫法對到原版 `ITEM` 記錄裡的名字。書照 AD&D 的
// 排法把修飾語擺後面（`Axe, Hand`），原版存的是直接的名字（`Hand Axe`）。
//
// 弓弩與投石索**不在這裡**：它們的傷害來自彈藥，型別表那一欄是佔位值
//（`ItemTypeFlagUsesAmmunition`／`NeedsLauncher`），拿來比會比到假的。
var manualWeaponNames = map[string]string{
	"Axe, Hand": "Hand Axe", "Bardiche+": "Bardiche", "Bastard Sword": "Bastard Sword",
	"Battleaxe": "Battle Axe", "Bec de Corbin+": "Bec De Corbin", "Bill-Guisarme+": "Bill-Guisarme",
	"Bo Stick": "Bo Stick", "Broad Sword": "Broad Sword", "Club": "Club", "Dagger": "Dagger",
	"Dart": "Dart", "Fauchard+": "Fauchard", "Fauchard-Fork+": "Fauchard-Fork", "Flail": "Flail",
	"Fork, Military+": "Military Fork", "Glaive+": "Glaive", "Glaive, Guisarme+": "Glaive-Guisarme",
	"Guisarme+": "Guisarme", "Guisarme-Voulge+": "Guisarme-Voulge", "Halberd+": "Halberd",
	"Lucern Hammer+": "Lucern Hammer", "Hammer": "Hammer", "Javelin": "Javelin", "Jo Stick": "Jo Stick",
	"Long Sword": "Long Sword", "Mace": "Mace", "Morning Star": "Morning Star", "Partisan+": "Partisan",
	"Pick, Military": "Military Pick", "Pike, Awl+": "Awl Pike", "Quarterstaff": "Quarter Staff",
	"Ranseur+": "Ranseur", "Scimitar": "Scimitar", "Short Sword": "Short Sword", "Spear": "Spear",
	"Spetum+": "Spetum", "Trident": "Trident", "Two-Handed Sword": "Two-Handed Sword", "Voulge+": "Voulge",
}

var weaponRow = regexp.MustCompile(`(?m)^\| ([^|]+?) \| (\d+)-(\d+) \| (\d+)-(\d+) \| (\d+) \| ([^|]+) \|$`)

// bookAgainstGame 是**書上與原版資料本身不一樣**的地方，不是轉錄錯字。
//
// 兩處都回對過掃描原頁（`Pic0032.jpg` 左半，上冊 p.54）：書印的就是這個數字。
// 差別出在書照抄 AD&D 規則書的表，而遊戲的 `ITEMS` 存的是自己那一份。
// 遊戲照自己的資料跑，所以 remake 取原版的值；這裡把差異釘住，免得日後
// 有人「照書修正」把遊戲行為改壞。
var bookAgainstGame = map[string]string{
	"Pick, Military": "書 2-5／1-4，原版型別 1Ah 是 2-7／2-8（AD&D 的軍用鎬正是 1d6+1／2d4，遊戲那一份才合規則書）",
	"Pike, Awl+":     "書對大型 2-12，原版型別 1Bh 是 1-12（1d12）",
}

// weaponTypeByName 掃過 `ITEM1..8` 的每一塊，收集「名字 → 型別索引」。
// 只收附錄 7 要用到的那幾個名字：`Potion`、`Ring`、`Wand` 這類同名不同物的
// 記錄本來就有好幾個型別，收進來只會製造雜訊。原版的名字前面可能帶數量
//（`4 Dart`、`2 Javelin`），比對前去掉。
func weaponTypeByName(t *testing.T, zipPath string) map[string]uint8 {
	t.Helper()
	wanted := map[string]bool{}
	for _, name := range manualWeaponNames {
		wanted[name] = true
	}
	quantity := regexp.MustCompile(`^\d+ `)
	types := map[string]uint8{}
	for archive := uint8(1); archive <= 8; archive++ {
		for block := 0; block < 256; block++ {
			records, err := gamepack.ReadDOSTreasureItemBlock(zipPath, archive, uint8(block))
			if err != nil {
				continue
			}
			for _, record := range records {
				name := quantity.ReplaceAllString(strings.TrimSpace(record.Name), "")
				if !wanted[name] {
					continue
				}
				typeID := record.Raw[gamepack.ItemTypeOffset]
				if existing, seen := types[name]; seen && existing != typeID {
					t.Errorf("%q 在原版裡有兩個型別 %02X／%02X", name, existing, typeID)
				}
				types[name] = typeID
			}
		}
	}
	return types
}

func TestWeaponAppendixMatchesTheGameItemTypes(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	table, err := gamepack.ReadDOSItemTypeTable(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	types := weaponTypeByName(t, zipPath)

	raw, err := os.ReadFile(manualBookPath("journal-vol1.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	start := strings.Index(text, "#### 7. 武器一覽表")
	if start < 0 {
		t.Fatal("找不到附錄 7")
	}
	rows := weaponRow.FindAllStringSubmatch(text[start:], -1)
	if len(rows) == 0 {
		t.Fatal("附錄 7 一列都沒抓到；表格格式是不是改了？")
	}
	span := func(entry gamepack.ItemTypeEntry, count, sides, bonus int) (int, int) {
		low := int(entry.Raw[count]) + int(entry.Raw[bonus])
		high := int(entry.Raw[count])*int(entry.Raw[sides]) + int(entry.Raw[bonus])
		return low, high
	}
	checked, divergent := 0, 0
	for _, row := range rows {
		manualName := strings.TrimSpace(row[1])
		gameName, mapped := manualWeaponNames[manualName]
		if !mapped {
			continue // 弓弩與投石索的傷害來自彈藥，不在這張表裡比
		}
		typeID, known := types[gameName]
		if !known {
			t.Errorf("原版的 ITEM 記錄裡找不到 %q（書上作 %q）", gameName, manualName)
			continue
		}
		entry, err := table.Entry(typeID)
		if err != nil {
			t.Errorf("%q 的型別 %02X 讀不出來：%v", gameName, typeID, err)
			continue
		}
		checked++
		bookSmallLow, _ := strconv.Atoi(row[2])
		bookSmallHigh, _ := strconv.Atoi(row[3])
		bookLargeLow, _ := strconv.Atoi(row[4])
		bookLargeHigh, _ := strconv.Atoi(row[5])
		smallLow, smallHigh := span(entry, itemTypeSmallCount, itemTypeSmallSides, itemTypeSmallBonus)
		largeLow, largeHigh := span(entry, itemTypeLargeCount, itemTypeLargeSides, itemTypeLargeBonus)
		agrees := smallLow == bookSmallLow && smallHigh == bookSmallHigh &&
			largeLow == bookLargeLow && largeHigh == bookLargeHigh
		if why, known := bookAgainstGame[manualName]; known {
			if agrees {
				t.Errorf("%s 現在書與原版一致了，bookAgainstGame 那一條該刪：%s", manualName, why)
			}
			divergent++
			continue
		}
		if smallLow != bookSmallLow || smallHigh != bookSmallHigh {
			t.Errorf("%s 對中小型：書上 %d-%d，原版型別 %02X 是 %d-%d",
				manualName, bookSmallLow, bookSmallHigh, typeID, smallLow, smallHigh)
		}
		if largeLow != bookLargeLow || largeHigh != bookLargeHigh {
			t.Errorf("%s 對大型：書上 %d-%d，原版型別 %02X 是 %d-%d",
				manualName, bookLargeLow, bookLargeHigh, typeID, largeLow, largeHigh)
		}
	}
	if checked != len(manualWeaponNames) {
		t.Errorf("書上對得到原版的武器只驗了 %d 種，名字表有 %d 種", checked, len(manualWeaponNames))
	}
	t.Logf("附錄 7：%d 種近戰武器，其中 %d 種與原版型別表逐格一致，%d 種是書與原版本來就不同（已記錄）",
		checked, checked-divergent, divergent)
}

// 附錄 3「裝備一覽表」的三欄，原版都有對應的資料。
//
// 最有用的發現是**書把欄名寫錯了**：那一欄印「價值（單位：黃金）」，可是十筆
// 數字裡有九筆等於原版物品記錄的重量（`+37h`），不是價目——皮革在店裡賣 5
// 枚金幣，書上那格寫 150，而 150 正是它的重量。轉錄照印，欄名的問題寫在譯註
// 裡，remake 取原版的值。
var manualArmourNames = map[string]string{
	"Shield": "Shield", "Leather": "Leather Armor", "Padded": "Padded Armor",
	"Studded": "Studded Leather Armor", "Ring": "Ring Mail", "Scale": "Scale Mail",
	"Chain": "Chain Mail", "Splint": "Splint Mail", "Banded": "Banded Mail", "Plate": "Plate Mail",
}

// armourBookAgainstGame 是書與原版本來就不一樣的三格，都回對過掃描原頁
//（`Pic0030.jpg` 左半，上冊 p.50）。
var armourBookAgainstGame = map[string]string{
	"Shield/重量": "書寫 50，原版沒加值的盾記錄是 100（AD&D 規則書的盾重量正是 50）",
	"Shield/防禦": "書寫 9，那是 AD&D「只拿盾」的檯面 AC；原版的盾是加值 1，不是一件 AC 為 9 的盔甲",
	"Padded/移動": "書寫 9，原版的護胸品重 100，落在「不壓速」那一段，算出來是 12",
}

var armourRow = regexp.MustCompile(`(?m)^\| [^（|]*（([A-Za-z]+)） \| (\d+) \| (\d+) \| ([^|]+) \|$`)

// plainArmourRecord 找出某件盔甲沒有魔法加值的那一筆記錄。
func plainArmourRecord(t *testing.T, zipPath string, want map[string]bool) map[string]gamepack.TreasureItemRecord {
	t.Helper()
	out := map[string]gamepack.TreasureItemRecord{}
	for archive := uint8(1); archive <= 8; archive++ {
		for block := 0; block < 256; block++ {
			records, err := gamepack.ReadDOSTreasureItemBlock(zipPath, archive, uint8(block))
			if err != nil {
				continue
			}
			for _, record := range records {
				name := strings.TrimSpace(record.Name)
				if !want[name] || record.Raw[gamepack.ItemPlusOffset] != 0 {
					continue
				}
				out[name] = record
			}
		}
	}
	return out
}

func TestArmourAppendixMatchesTheGameRecords(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	table, err := gamepack.ReadDOSItemTypeTable(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	want := map[string]bool{}
	for _, name := range manualArmourNames {
		want[name] = true
	}
	records := plainArmourRecord(t, zipPath, want)

	raw, err := os.ReadFile(manualBookPath("journal-vol1.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	start := strings.Index(text, "#### 3. 裝備一覽表")
	if start < 0 {
		t.Fatal("找不到附錄 3")
	}
	end := strings.Index(text[start:], "#### 4. ")
	if end < 0 {
		t.Fatal("找不到附錄 4，附錄 3 的範圍切不出來")
	}
	rows := armourRow.FindAllStringSubmatch(text[start:start+end], -1)
	if len(rows) != len(manualArmourNames) {
		t.Fatalf("附錄 3 抓到 %d 列，書上是 %d 列", len(rows), len(manualArmourNames))
	}
	used := map[string]bool{}
	check := func(key string, agrees bool, complain func()) {
		if why, known := armourBookAgainstGame[key]; known {
			used[key] = true
			if agrees {
				t.Errorf("%s 現在書與原版一致了，armourBookAgainstGame 那一條該刪：%s", key, why)
			}
			return
		}
		if !agrees {
			complain()
		}
	}
	for _, row := range rows {
		book := row[1]
		gameName, mapped := manualArmourNames[book]
		if !mapped {
			t.Errorf("附錄 3 有一列是 %q，名字表裡沒有", book)
			continue
		}
		record, found := records[gameName]
		if !found {
			t.Errorf("原版裡找不到沒有加值的 %q", gameName)
			continue
		}
		weight := int(record.Raw[gamepack.ItemWeightOffset]) | int(record.Raw[gamepack.ItemWeightOffset+1])<<8
		bookWeight, _ := strconv.Atoi(row[2])
		check(book+"/重量", weight == bookWeight, func() {
			t.Errorf("%s：書上那一欄寫 %d，原版記錄的重量是 %d", book, bookWeight, weight)
		})

		entry, err := table.Entry(record.Raw[gamepack.ItemTypeOffset])
		if err != nil {
			t.Errorf("%s 的型別讀不出來：%v", book, err)
			continue
		}
		bookArmour, _ := strconv.Atoi(row[3])
		tabletop := gamepack.ArmourClassScale - int(entry.Raw[gamepack.ItemTypeArmourClassOffset]&0x7f)
		check(book+"/防禦", tabletop == bookArmour, func() {
			t.Errorf("%s：書上防禦力 %d，原版型別的 AC 欄換算出來是 %d", book, bookArmour, tabletop)
		})

		bookMove, err := strconv.Atoi(strings.TrimSpace(row[4]))
		if err != nil {
			continue // 盾那一格書上是空的
		}
		rate := gamepack.ArmourMovementRate(weight, 0, gamepack.BaseMovementRate)
		check(book+"/移動", rate == bookMove, func() {
			t.Errorf("%s：書上移動 %d，原版依重量 %d 算出來是 %d", book, bookMove, weight, rate)
		})
	}
	for key, why := range armourBookAgainstGame {
		if !used[key] {
			t.Errorf("armourBookAgainstGame 的 %q 沒有被用到，可能是名字改了：%s", key, why)
		}
	}
	t.Logf("附錄 3：%d 列，除了記錄在案的 %d 格之外都與原版一致", len(rows), len(armourBookAgainstGame))
}

// 附錄 5「牧師對抗不死怪物」對得上 `START.EXE` 裡的轉變表。
//
// 這一則把兩份互不相干的來源綁在一起：書上印的是「這種不死怪物至少要幾級的
// 牧師才影響得了」，執行檔裡是一張 10×10 的門檻矩陣（`DS:45Bh`，spec 111）。
// 從矩陣算「第一個門檻不是 99 的列」，應該就是書上那個等級。
//
// 兩邊都對得上，代表**欄位的順序讀對了**——欄位讀錯一格，矩陣自己仍然自洽
// （右下角照樣比左上角好），只有拿書上的獨立列表去比才分得出來。
var undeadTurnColumns = []struct {
	name   string
	column int
	level  int
}{
	{"骷髏", 1, 1},
	{"僵屍", 2, 1},
	{"餓鬼", 3, 1},
	{"人類", 4, 1},
	{"幽靈", 7, 3},
	{"木乃伊", 8, 4},
	{"妖怪", 9, 5},
	{"吸血鬼", 10, 6},
}

var chineseLevels = map[string]int{"第一級": 1, "第二級": 2, "第三級": 3, "第四級": 4,
	"第五級": 5, "第六級": 6, "第七級": 7, "第八級": 8}

var undeadRow = regexp.MustCompile(`(?m)^\| ([^|]+?) \| (第[一二三四五六七八]級) \|$`)

func TestUndeadAppendixMatchesTheTurnTable(t *testing.T) {
	table, err := gamepack.ReadDOSTurnUndeadTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	raw, err := os.ReadFile(manualBookPath("journal-vol1.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	start := strings.Index(text, "#### 5. 牧師對抗不死怪物")
	if start < 0 {
		t.Fatal("找不到附錄 5")
	}
	end := strings.Index(text[start:], "#### 6. ")
	if end < 0 {
		t.Fatal("找不到附錄 6，附錄 5 的範圍切不出來")
	}
	printed := map[string]int{}
	for _, row := range undeadRow.FindAllStringSubmatch(text[start:start+end], -1) {
		level, ok := chineseLevels[row[2]]
		if !ok {
			t.Fatalf("讀不懂等級 %q", row[2])
		}
		printed[strings.TrimSpace(row[1])] = level
	}
	if len(printed) != len(undeadTurnColumns) {
		t.Fatalf("附錄 5 抓到 %d 列，名字表有 %d 列", len(printed), len(undeadTurnColumns))
	}
	for _, entry := range undeadTurnColumns {
		bookLevel, listed := printed[entry.name]
		if !listed {
			t.Errorf("附錄 5 裡沒有「%s」", entry.name)
			continue
		}
		if bookLevel != entry.level {
			t.Errorf("附錄 5 說「%s」要第 %d 級，名字表登的是第 %d 級",
				entry.name, bookLevel, entry.level)
		}
		lowest := 0
		for level := 1; level <= gamepack.TurnUndeadRows; level++ {
			if table.Threshold(level, entry.column) <= gamepack.TurnUndeadDie {
				lowest = level
				break
			}
		}
		if lowest != bookLevel {
			t.Errorf("%s（欄 %d）：書上要第 %d 級，轉變表算出來是第 %d 級",
				entry.name, entry.column, bookLevel, lowest)
		}
	}
	t.Logf("附錄 5 的 %d 種不死怪物，最低牧師等級全部與轉變表一致", len(undeadTurnColumns))
}
