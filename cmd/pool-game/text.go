package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/etenfont"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gametext"
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
	// 標題畫面的按鍵提示，說明書沒有，鍵名保持原文。
	msgTitleHint messageID = iota
	// 說明書 p.8：「螢幕上便會出現人物管理選擇項」。
	msgMenuTitle
	// 說明書 p.8 C）REATE NEW CHARACTER：「創造一名人物…並將之存放到『人物名單』內」。
	msgMenuCreate
	// 說明書 p.9 A）DD CHARACTER：「將人物加入隊伍」。
	msgMenuAdd
	// 說明書 p.9 L）OAD SAVED GAME：「叫出以前存下的遊戲進度」。
	msgMenuLoad
	// 說明書 p.10 B）EGIN ADVENTURING：「離開人物管理選擇項，開始冒險」。
	msgMenuBegin
	// 「人物名單」是說明書對 character library 的固定用詞。
	msgMenuCounts
	// 功能鍵列是 remake 自己的東西，說明書沒有；鍵名保持原文。
	msgFooter
)


// text 取出目前語言的字串。缺譯時退回英文而不是留白——留白在畫面上看不出是
// 缺譯還是繪製壞了。
// 介面字串放在 game pack 的 locale 檔（`internal/gamepack/pack/20-locale.*.json`），
// 不放在這裡：共用 engine 是作品中立的，作品的內容一律由 pack 提供。
// 這一側只留 `messageID` 與它的出處註解，字串本身用 `messageKeys` 的 key 去查。
//
// 繁中一律取自軟體世界代理當年的官方中文說明書
//（`docs/reference/manual/manual-vol2.md` p.8–p.10 的「人物管理選擇項」一節），
// 不是重新翻譯。說明書沒有對應字串的（畫面提示、功能鍵列）才自行擬定，
// 用詞依 `docs/reference/manual/glossary.md` 的定案譯名。
func (a *app) text(id messageID) string {
	if a.language == languageTraditionalChinese {
		if value := packMessage(id, "zh-TW"); value != "" {
			return value
		}
	}
	return packMessage(id, "en")
}

// packMessage 查一則訊息。查不到就回空字串——呼叫端本來就把空字串當「這一則
// 沒有文字」處理（原本的 `messages` 表也是這樣）。
//
// pack 載不進來時回 key 本身，讓失敗看得見：整片空字串看起來像排版壞掉，
// 而畫面上出現 `ui.menuTitle` 一眼就知道是資料沒載到。
func packMessage(id messageID, locale string) string {
	key, ok := messageKeys[id]
	if !ok {
		return ""
	}
	table, err := gamepack.LocaleTable(locale)
	if err != nil {
		return key
	}
	return table[key]
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

// gameTextFor 取出該語言的原版敘述文字譯文表。英文模式沒有表，
// 於是 Translate 一律原樣回傳，走的是同一條路徑。
func gameTextFor(lang language) (*gametext.Catalogue, error) {
	if lang != languageTraditionalChinese {
		return nil, nil
	}
	return gametext.TraditionalChinese()
}

// monsterTextFor 是怪物名表。與 `gameTextFor` 分開，因為兩者的來源不同：
// 敘述文字來自 ECL 的 6-bit packed 字串，怪物名來自 `MONnCHA` 記錄。
func monsterTextFor(lang language) (*gametext.MonsterCatalogue, error) {
	if lang != languageTraditionalChinese {
		return nil, nil
	}
	return gametext.TraditionalChineseMonsters()
}

// runeWidth 回傳一個字元佔的半形格數。倚天字型的漢字是 16 像素寬、ASCII 是 8，
// 所以換行是以半形格為單位算的。
func runeWidth(r rune) int {
	switch {
	case r >= 0x1100 && r <= 0x115F, // 韓文字母
		r >= 0x2E80 && r <= 0x303E, // 部首與 CJK 標點
		r >= 0x3041 && r <= 0x33FF, // 假名、注音、相容字
		r >= 0x4E00 && r <= 0x9FFF, // 漢字
		r >= 0xF900 && r <= 0xFAFF, // 相容漢字
		r >= 0xFE30 && r <= 0xFE4F, // 縱書標點
		r >= 0xFF00 && r <= 0xFF60, // 全形 ASCII
		r >= 0xFFE0 && r <= 0xFFE6:
		return 2
	default:
		return 1
	}
}

// closingPunctuation 是不該落在行首的字元。中文排版裡把它們留在上一行末尾，
// 即使那一行因此多出一格。
// closingPunctuationSlack 是允許收尾標點超出的格數：一個全形標點。
const closingPunctuationSlack = 2

const closingPunctuation = "。，、；：？！）」』〉》”’,.;:?!)]}"

// wrapDisplay 依半形格數換行，同時吃得下中英文。ASCII 以空白斷詞、整個詞不拆；
// 漢字每一個字都可以斷。行首不放收尾標點。
//
// wrapASCII 只看空白，中文一整段沒有空白，用它會得到一條長到溢出對話框的線。
// displayText 把字型沒有字模的排版符號換成有的形狀；替換表在 etenfont，
// 與字型覆蓋率稽核共用同一份，兩邊才不會對不上。
func displayText(value string) string { return etenfont.ReplaceUnavailable(value) }

func wrapDisplay(value string, columns int) []string {
	if columns < 1 {
		return nil
	}
	type token struct {
		text  string
		width int
	}
	var tokens []token
	var word strings.Builder
	flush := func() {
		if word.Len() == 0 {
			return
		}
		text := word.String()
		width := 0
		for _, r := range text {
			width += runeWidth(r)
		}
		tokens = append(tokens, token{text, width})
		word.Reset()
	}
	for _, r := range value {
		switch {
		case r == ' ' || r == '\t' || r == '\n':
			flush()
		case runeWidth(r) == 2:
			flush()
			tokens = append(tokens, token{string(r), 2})
		default:
			word.WriteRune(r)
		}
	}
	flush()

	var lines []string
	var line strings.Builder
	used := 0
	for index := 0; index < len(tokens); index++ {
		current := tokens[index]
		// 兩個半形詞會相鄰，只可能是原文那裡本來就有空白——切詞時全形字自成一個
		// token，半形的一串只在遇到空白時才斷。所以這裡補回來的空白就是原本那個。
		// 舊版的條件寫成「這個 token 寬度是 1」，於是 `PRESS RETURN OR BUTTON`
		// 會被接成 `PRESSRETURNORBUTTON`。
		separator := ""
		if used > 0 && line.Len() > 0 && runeWidth([]rune(current.text)[0]) == 1 {
			last, _ := utf8DecodeLast(line.String())
			if runeWidth(last) == 1 {
				separator = " "
			}
		}
		need := current.width + len(separator)
		if used > 0 && used+need > columns {
			// 收尾標點寧可讓這一行多一格，也不要落到下一行的行首；但只讓一個
			// 標點溢位，否則連續的「。」」會一路把字推出畫面外。
			closer := strings.ContainsRune(closingPunctuation, []rune(current.text)[0])
			if !closer || used+need > columns+closingPunctuationSlack {
				lines = append(lines, line.String())
				line.Reset()
				used, separator, need = 0, "", current.width
			}
		}
		line.WriteString(separator)
		line.WriteString(current.text)
		used += need
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	return lines
}

// utf8DecodeLast 取出字串最後一個字元。
func utf8DecodeLast(value string) (rune, bool) {
	runes := []rune(value)
	if len(runes) == 0 {
		return 0, false
	}
	return runes[len(runes)-1], true
}

// 戰術畫面的字串。戰鬥用語依說明書第五章與 glossary 的定案（ROUND＝戰鬥回合）。
// 法術一覽的字串。
const (
	msgSpellsTitle messageID = iota + 600
	msgSpellsGroup
	msgSpellsCleric
	msgSpellsMagicUser
	msgSpellsCount
	msgSpellsFooter
	msgSpellsNoEffect
	msgSpellsDurationFixed
	msgSpellsDurationPerLevel
	msgSpellsDurationBoth
	msgSpellsDurationUntilBroken
	msgSpellsSaveNone
	msgSpellsSaveSpell
	msgSpellsSavePoison
	msgSpellsTouch
	msgSpellsRangeFixed
	msgSpellsRangePerLevel
	msgEncounterDistance
	msgEncounterPrompt
)

// 商店畫面的字串。
const (
	msgShopTitle messageID = iota + 500
	msgShopStatus
	msgShopBuyer
	msgShopCount
	msgShopFooter
	msgShopBought
	msgShopNoGold
	msgShopOverloaded
	msgShopNoParty
)

// 裝備畫面的字串。
const (
	msgEquipmentTitle messageID = iota + 400
	msgEquipmentEmptyParty
	msgEquipmentNoItems
	msgEquipmentReadyMark
	msgEquipmentUnreadyable
	msgEquipmentUnarmed
	msgEquipmentStats
	msgEquipmentDefence
	msgProgramManaging
	msgProgramReturn
	msgProgramNeedsParty
	msgTrainCommand
	msgTrainNotYet
	msgTrainGained
	msgTrainNeedsMember
	msgSpellsNeedsMember
	msgSpellsMemorised
	msgSpellsForgot
	msgSpellsNotMemorised
	msgSpellsNoSlot
	msgSpellsMemoriseHint
	msgSpellsNotInBook
	msgSpellsNoCredit
	msgSpellsCannotLearn
	msgSpellsAlreadyKnown
	msgSpellsLearned
	msgSpellsSlotLine
	msgCastNotACaster
	msgCastNothingReady
	msgCastNoTarget
	msgEndingPrompt
	msgCastNotPerson
	msgCastStronger
	msgCastCharmed
	msgCastCharmedNone
	msgCastHeld
	msgCastResisted
	msgCastHealed
	msgCastHit
	msgCastDown
	msgCastTookEffect
	msgCastHint
	msgCastSlept
	msgCastSleptNone
	msgCastCured
	msgCastNoEffect
	msgCastArea
	msgCastWholeSide
	msgCastAiming
	msgAimAttack
	msgAimOutOfRange
	msgAimBlocked
	msgCampNeedsParty
	msgCampRest
	msgCampMemorise
	msgCampExit
	msgCampTitle
	msgCampHint
	msgCampRested
	msgCampHealedOnly
	msgCampPending
	msgCampNothingPending
	msgDamageHit
	msgDamageDies
	msgDamageSaved
	msgParlayPrompt
	msgEclInputPrompt
	msgRobbed
	msgWhoPrompt
	msgNPCJoined
	msgParlayHaughty
	msgParlaySly
	msgParlayNice
	msgParlayMeek
	msgParlayAbusive
	msgEquipmentFooter
)

const (
	msgTacticalTitle messageID = iota + 200
	msgTacticalNoMap
	msgTacticalNoState
	msgTacticalBoard
	msgTacticalRound
	// 這一項標的是三塊還沒有原版依據的東西，中英文都要看得出來是暫定的。
	msgTacticalProvisional
	msgTacticalKeys
	msgTacticalPrompt
	msgTacticalBack
)

// 戰術狀態列的字串。這些是 remake 自己產生的訊息，不是原版文字，
// 但玩家看得到，所以一樣要有中文。
const (
	msgStatusDelayed messageID = iota + 300
	msgStatusTurnEnded
	// 原文是 overlay-08 `0857h` 的 `Continue Battle:`。
	msgStatusContinuePrompt
	msgStatusDefeat
	msgStatusVictory
	msgStatusRound
	msgStatusOffBoard
	msgStatusBlocked
	msgStatusMoved
	msgStatusMissed
	msgStatusHit
	msgStatusDown
	msgStatusAsleep
	msgStatusCharmed
	msgStatusHeld
	msgFoeNoTarget
	msgFoeAttacked
	msgFoeClosed
	msgBudgetPlaceholder
	msgBudgetStagedMonster
)

