package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/etenfont"
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
	msgSpellsTitle:     {"SPELLS", "法術一覽"},
	msgSpellsGroup:     {"%s  level %d", "%s　第 %d 級"},
	msgSpellsCleric:    {"Cleric", "神術"},
	msgSpellsMagicUser: {"Magic User", "巫術"},
	msgSpellsCount:     {"%d spells in this group", "本級共 %d 種"},
	msgSpellsFooter:    {"TAB next group  UP/DOWN choose  K/ESC close", "TAB 換級別　上下移動　K／ESC 關閉"},
	msgSpellsNoEffect:  {"The manual's spell chapter does not describe this one.", "說明書的法術章沒有收這一條。"},
	msgSpellsRangeFixed:          {"range %d", "射程 %d 格"},
	msgSpellsRangePerLevel:       {"range %d +%d/level", "射程 %d 格（每級 +%d）"},
	msgSpellsDurationFixed:       {"%d rounds", "持續 %d 回合"},
	msgSpellsDurationPerLevel:    {"%d rounds/level", "每級 %d 回合"},
	msgSpellsDurationBoth:        {"%d rounds +%d/level", "持續 %d 回合（每級 +%d）"},
	msgSpellsDurationUntilBroken: {"until dispelled", "持續到解除"},
	msgSpellsSaveNone:            {"no save", "不可豁免"},
	msgSpellsSaveSpell:           {"save vs. spell", "可豁免（法術）"},
	msgSpellsSavePoison:          {"save vs. poison", "可豁免（毒）"},
	msgSpellsTouch:               {"must hit", "須擲中"},
	msgEncounterDistance:         {"The monsters are %d squares away.", "怪物在 %d 格之外。"},
	msgEncounterPrompt:           {"Choose how to meet them.", "選擇要怎麼應對。"},
	msgShopTitle:      {"SHOP", "商店"},
	msgShopStatus:     {"Original Pool shop service: %d item(s) in stock.", "原版商店服務：架上 %d 件。"},
	msgShopBuyer:      {"Buyer %d/%d  %s  gold %d", "買家 %d/%d　%s　金幣 %d"},
	msgShopCount:      {"item %d of %d", "第 %d 件，共 %d 件"},
	msgShopFooter:     {"TAB switch buyer  UP/DOWN choose  ENTER buy  ESC leave", "TAB 換買家　上下選貨　ENTER 購買　ESC 離開"},
	msgShopBought:     {"%s bought %s for %d gold.", "%s 買下 %s，花了 %d 金幣。"},
	msgShopNoGold:     {"%s has %d gold but this costs %d.", "%s 只有 %d 金幣，這件要 %d。"},
	msgShopOverloaded: {"%s cannot carry any more.", "%s 拿不動了。"},
	msgShopNoParty:    {"The party is empty.", "隊伍裡沒有人。"},
	msgEquipmentTitle:       {"PARTY EQUIPMENT", "隊伍裝備"},
	msgEquipmentEmptyParty:  {"The party is empty.", "隊伍裡沒有人。"},
	msgEquipmentNoItems:     {"This character carries nothing.", "這名角色身上沒有東西。"},
	msgEquipmentReadyMark:   {"* ", "＊"},
	msgEquipmentUnreadyable: {"That item cannot be readied.", "這件東西不能裝備。"},
	msgEquipmentUnarmed:     {"Unarmed: the original leaves the damage dice alone.", "徒手：原版此時不動傷害骰。"},
	msgEquipmentStats:       {"Readied: THAC0 %d, damage %dd%d%+d", "已裝備：THAC0 %d，傷害 %dd%d%+d"},
	msgEquipmentFooter:      {"TAB switch character  UP/DOWN choose  ENTER ready  I/ESC close", "TAB 換人　上下選物品　ENTER 裝備／卸下　I／ESC 關閉"},
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

// gameTextFor 取出該語言的原版敘述文字譯文表。英文模式沒有表，
// 於是 Translate 一律原樣回傳，走的是同一條路徑。
func gameTextFor(lang language) (*gametext.Catalogue, error) {
	if lang != languageTraditionalChinese {
		return nil, nil
	}
	return gametext.TraditionalChinese()
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
	msgEquipmentFooter
)

const (
	msgTacticalTitle messageID = iota + 200
	msgTacticalNoMap
	msgTacticalNoState
	msgTacticalBoard
	msgTacticalRound
	msgTacticalProvisional
	msgTacticalKeys
	msgTacticalPrompt
	msgTacticalBack
)

func init() {
	for id, entry := range map[messageID][2]string{
		msgTacticalTitle:   {"TACTICAL MAP PREVIEW", "戰術戰場"},
		msgTacticalNoMap:   {"DUNGEON MAP IS NOT LOADED", "地城地圖尚未載入"},
		msgTacticalNoState: {"TACTICAL STATE IS NOT BUILT", "戰場尚未建立"},
		msgTacticalBoard: {
			"DUNGEON %d,%d  CELLS %d  BLOCKING %d  PARTY %d  FOES %d",
			"地城 %d,%d　格 %d　阻擋 %d　隊伍 %d　敵方 %d",
		},
		msgTacticalRound: {
			"ROUND %d  MOVER %d  SCORE %d  BUDGET %d (%s)  %s",
			"回合 %d　行動者 %d　先攻 %d　步數 %d（%s）　%s",
		},
		// 這一行標的是三塊還沒有原版依據的東西，中英文都要看得出來是暫定的。
		msgTacticalProvisional: {
			"PROVISIONAL: AI, DEPLOYMENT, PARTY DAMAGE",
			"暫定：敵方 AI、部署位置、隊伍傷害骰",
		},
		msgTacticalKeys: {
			"H I M Q P O K G: STEP   ENTER: END TURN   D: DELAY",
			"H I M Q P O K G 移動　ENTER 結束回合　D 延後",
		},
		msgTacticalPrompt: {"Y: FIGHT ON   N: END THE BATTLE", "Y 繼續戰鬥　N 結束戰鬥"},
		msgTacticalBack:   {"F5: BACK", "F5 返回"},
	} {
		messages[id] = entry
	}
}

// 戰術狀態列的字串。這些是 remake 自己產生的訊息，不是原版文字，
// 但玩家看得到，所以一樣要有中文。
const (
	msgStatusDelayed messageID = iota + 300
	msgStatusTurnEnded
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
	msgFoeNoTarget
	msgFoeAttacked
	msgFoeClosed
	msgBudgetPlaceholder
	msgBudgetStagedMonster
)

func init() {
	for id, entry := range map[messageID][2]string{
		msgStatusDelayed:        {"DELAYED", "延後"},
		msgStatusTurnEnded:      {"TURN ENDED", "回合結束"},
		msgStatusContinuePrompt: {"CONTINUE BATTLE? Y/N", "要繼續戰鬥嗎？ Y／N"},
		msgStatusDefeat:         {"DEFEAT", "全滅"},
		msgStatusVictory:        {"VICTORY", "獲勝"},
		msgStatusRound:          {"ROUND %d", "第 %d 回合"},
		msgStatusOffBoard:       {"OFF BOARD: LEAVE COMBAT PROMPT", "走出盤面：詢問是否離開戰鬥"},
		msgStatusBlocked:        {"BLOCKED", "走不過去"},
		msgStatusMoved:          {"MOVED %d", "往 %d 移動"},
		msgStatusMissed:         {"ATTACK %d MISSED (D20 %d)", "攻擊 %d 落空（D20 %d）"},
		msgStatusHit:            {"HIT %d FOR %d (HP %d)", "打中 %d 造成 %d（剩 %d 生命力）"},
		msgStatusDown:           {"%d IS DOWN", "%d 倒下了"},
		msgFoeNoTarget:          {"FOE %d FOUND NO TARGET", "敵方 %d 找不到目標"},
		msgFoeAttacked:          {"FOE %d AFTER %d STEPS: %s", "敵方 %d 走了 %d 步：%s"},
		msgFoeClosed:            {"FOE %d CLOSED %d STEPS ON %d", "敵方 %d 朝 %d 走近 %d 步"},
		msgBudgetPlaceholder:    {"PLACEHOLDER", "暫定值"},
		msgBudgetStagedMonster:  {"STAGED MONSTER", "怪物記錄"},
	} {
		messages[id] = entry
	}
}
