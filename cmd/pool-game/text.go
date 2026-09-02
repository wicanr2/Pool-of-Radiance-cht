package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/etenfont"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// language 是 UI 的顯示語言。
type language int

const (
	languageEnglish language = iota
	languageTraditionalChinese
)

type messageID int

const (
	msgTitleHint messageID = iota
	msgMenuTitle
	msgMenuCreate
	msgMenuAdd
	msgMenuLoad
	msgMenuBegin
	msgMenuCounts
	msgFooter
)

// messages 的繁中一律取自軟體世界代理當年的官方中文說明書
// （`docs/reference/manual/manual-vol2.md` p.8–p.10 的「人物管理選擇項」一節），
// 不是重新翻譯。說明書沒有對應字串的（畫面提示、功能鍵列）才自行擬定，
// 用詞依 `docs/reference/manual/glossary.md` 的定案譯名。
var messages = map[messageID][2]string{
	// 標題畫面的按鍵提示，說明書沒有，鍵名保持原文。
	msgTitleHint: {"ENTER / SPACE", "ENTER／空白鍵"},
	// 說明書 p.8：「螢幕上便會出現人物管理選擇項」。
	msgMenuTitle: {"PARTY CREATION MENU", "人物管理選擇項"},
	// 說明書 p.8 C）REATE NEW CHARACTER：「創造一名人物…並將之存放到『人物名單』內」。
	msgMenuCreate: {"C  CREATE NEW CHARACTER", "C　創造新人物"},
	// 說明書 p.9 A）DD CHARACTER：「將人物加入隊伍」。
	msgMenuAdd: {"A  ADD CHARACTER TO PARTY", "A　將人物加入隊伍"},
	// 說明書 p.9 L）OAD SAVED GAME：「叫出以前存下的遊戲進度」。
	msgMenuLoad: {"L  LOAD SAVED GAME", "L　叫出存下的進度"},
	// 說明書 p.10 B）EGIN ADVENTURING：「離開人物管理選擇項，開始冒險」。
	msgMenuBegin: {"B  BEGIN ADVENTURING", "B　開始冒險"},
	// 「人物名單」是說明書對 character library 的固定用詞。
	msgMenuCounts: {"LIBRARY %d   PARTY %d/6", "人物名單 %d　隊伍 %d/6"},
	// 功能鍵列是 remake 自己的東西，說明書沒有；鍵名保持原文。
	msgFooter: {
		"F1 Help  F2 Theme  F5 Tactical  ESC Back  F10 Quit",
		"F1 說明　F2 配色　F5 戰術圖　ESC 返回　F10 離開",
	},
}

// text 取出目前語言的字串。缺譯時退回英文而不是留白——留白在畫面上看不出是
// 缺譯還是繪製壞了。
func (a *app) text(id messageID) string {
	entry, ok := messages[id]
	if !ok {
		return ""
	}
	if a.language == languageTraditionalChinese && entry[1] != "" {
		return entry[1]
	}
	return entry[0]
}

// uiFace 是 drawText 實際使用的字型。Ebiten 的 Draw 是單執行緒的，啟動時設定
// 一次之後就不再變動，所以這裡用套件層變數而不是把每個呼叫點都改成帶字型。
var uiFace font.Face = basicfont.Face7x13

// resolveUILanguage 依旗標決定語言並載入字型。
//
// 指定繁中卻沒有給字型時失敗即關閉：內建的 7x13 沒有漢字，硬跑會整片留白，
// 而留白看起來像繪製壞掉，不像缺字型。
func resolveUILanguage(langFlag, standardPath, symbolPath, asciiPath string) (language, font.Face, error) {
	wantChinese := false
	switch langFlag {
	case "en":
	case "zh":
		wantChinese = true
	case "auto":
		wantChinese = standardPath != ""
	default:
		return 0, nil, fmt.Errorf("Pool UI language %q is not one of en, zh, auto", langFlag)
	}
	if !wantChinese {
		return languageEnglish, basicfont.Face7x13, nil
	}
	if standardPath == "" {
		return 0, nil, fmt.Errorf("Pool Traditional Chinese UI needs -eten-font; the built-in face has no Han glyphs")
	}
	// 同目錄的兩個伴隨檔自動帶上：ascfont.15 是半形 ASCII，spcfont.15 是全形標點。
	// 沒有 spcfont.15 時，全形逗號句號會退回內建的半形字元——看得懂，但不是原樣，
	// 所以截圖的 manifest 要記下當時有沒有這個檔。
	beside := func(name string) string {
		candidate := filepath.Join(filepath.Dir(standardPath), name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		return ""
	}
	if asciiPath == "" {
		asciiPath = beside("ascfont.15")
	}
	if symbolPath == "" {
		symbolPath = beside("spcfont.15")
	}
	face, err := etenfont.LoadWithASCII(standardPath, symbolPath, asciiPath, basicfont.Face7x13, true)
	if err != nil {
		return 0, nil, err
	}
	return languageTraditionalChinese, face, nil
}

// optionNames 是建角選項的繁中名稱，全部取自官方中文說明書：
// 種族與性別見 p.10–p.11，四種職業見 p.13–p.14，九個陣營見 p.15 的對照表。
// 兼職不另外列，依說明書 p.14「××／×× 的選擇項代表兼職」由組成職業合成。
var optionNames = map[string]string{
	"dwarf": "矮人", "gnome": "侏儒", "half-elf": "半精靈",
	"halfling": "半身人", "human": "人類", "elf": "精靈",

	"male": "男性", "female": "女性",

	"cleric": "牧師", "fighter": "戰士", "magic-user": "魔法師", "thief": "賊",

	"lawful-good": "嚴守善良", "lawful-neutral": "嚴守中立", "lawful-evil": "嚴守邪惡",
	"neutral-good": "普通善良", "true-neutral": "真正中立", "neutral-evil": "普通邪惡",
	"chaotic-good": "混亂的善良", "chaotic-neutral": "混亂的中立", "chaotic-evil": "混亂的邪惡",
}

// optionText 取出一個建角選項的顯示名稱。兼職的 ID 是以連字號串起來的組成職業，
// 依說明書的寫法用全形斜線接起來；查不到就退回英文標籤。
func (a *app) optionText(id, label string) string {
	if a.language != languageTraditionalChinese {
		return label
	}
	if name, ok := optionNames[id]; ok {
		return name
	}
	parts := strings.Split(id, "-")
	names := make([]string, 0, len(parts))
	for index := 0; index < len(parts); {
		// magic-user 本身帶連字號，所以先試兩段再試一段。
		if index+1 < len(parts) {
			if name, ok := optionNames[parts[index]+"-"+parts[index+1]]; ok {
				names = append(names, name)
				index += 2
				continue
			}
		}
		name, ok := optionNames[parts[index]]
		if !ok {
			return label
		}
		names = append(names, name)
		index++
	}
	return strings.Join(names, "／")
}

// 建角畫面的字串。屬性名稱取自說明書 p.11–p.12 的六段解釋
// （力量、智慧、睿智、敏捷、體質、魅力），生命力與金幣同頁。
const (
	msgStageRace messageID = iota + 100
	msgStageGender
	msgStageClass
	msgStageAlignment
	msgCharacterSheet
	msgRolling
	msgAge
	msgGoldAndHP
	msgKeepCharacter
	msgCharacterName
	msgNameRule
)

func init() {
	for id, entry := range map[messageID][2]string{
		msgStageRace:      {"PICK RACE", "選擇種族"},
		msgStageGender:    {"PICK GENDER", "選擇性別"},
		msgStageClass:     {"PICK CLASS", "選擇職業"},
		msgStageAlignment: {"PICK ALIGNMENT", "選擇陣營"},
		msgCharacterSheet: {"CHARACTER SHEET", "人物資料"},
		msgRolling:        {"Rolling...", "重擲中…"},
		msgAge:            {"AGE %d", "年齡 %d"},
		msgGoldAndHP:      {"GOLD %d     HP %d/%d", "金幣 %d　　生命力 %d/%d"},
		msgKeepCharacter: {
			"KEEP THIS CHARACTER?  ENTER/Y = YES   R = REROLL",
			"保留這個人物？　ENTER／Y 保留　R 重擲",
		},
		msgCharacterName: {"CHARACTER NAME:", "人物姓名："},
		msgNameRule:      {"1-15 CHARACTERS; ENTER ACCEPTS", "1～15 個字元，ENTER 確定"},
	} {
		messages[id] = entry
	}
}

// hintNames 是建角各階段的提示。繁中依說明書的對應段落改寫成一行：
// 種族限制職業見 p.10「種族能影響職別的種類」，兼職分經驗見 p.13 第 (2) 點，
// 陣營影響 NPC 觀感見 p.15，肖像與戰鬥造形見 p.11 與第五章。
var hintNames = map[string]string{
	"race":      "種族會限制能選的職業，這份清單依原版排列。",
	"class":     "兼職的人物昇級較慢，經驗點數會分到各職別去。",
	"alignment": "陣營是人物的生活方式，會影響特殊隊員對他的觀感。",
	"portrait":  "H 換頭、B 換身體，K 決定這個肖像。",
	"icon":      "編輯部位、雙色與大小時，右邊會同步預覽 READY 與 ACTION。",
}

func (a *app) hint(stage string) string {
	if a.language == languageTraditionalChinese {
		if text, ok := hintNames[stage]; ok {
			return text
		}
	}
	return creation.HintFor(stage)
}

// abilityNames 是六項屬性的縮寫與繁中名稱，順序與 record 的能力值陣列相同。
var abilityNames = [6][2]string{
	{"STR", "力量"}, {"INT", "智慧"}, {"WIS", "睿智"},
	{"DEX", "敏捷"}, {"CON", "體質"}, {"CHA", "魅力"},
}

func (a *app) abilityName(index int) string {
	if index < 0 || index >= len(abilityNames) {
		return ""
	}
	if a.language == languageTraditionalChinese {
		return abilityNames[index][1]
	}
	return abilityNames[index][0]
}
