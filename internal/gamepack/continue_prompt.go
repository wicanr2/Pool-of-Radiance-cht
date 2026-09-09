package gamepack

import (
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/tpov"
)

// 「按鍵繼續」那一列（spec 082）。
//
// 停頓是腳本自己放的：要玩家按一下的地方，ECL 會放一個**只有一個選項**的
// `HORIZONTAL MENU`，而那個選項是腳本裡的字串——市政廳外是
// `PRESS <RETURN> OR BUTTON TO CONTINUE`，別的 block 另外還有五種寫法。
// overlay-03 `11B1h` 在選項數是 1 的時候**不畫腳本給的那一條**，改畫自己這
// 兩條之一（引用點 `120Ch`／`1218h`），所以玩家實際上只會看到一種寫法。
//
// 兩條之間怎麼挑還沒讀（推測是有沒有搖桿），但原版畫面印的是鍵盤那一條
//（dosgolem 基準圖 `03`，見 `docs/reference/original-dos/adventure/README.md`），
// remake 沒有搖桿，所以用 ContinuePromptKeyboard。

// ContinuePromptOverlay 是那兩條字串所在的 overlay。
const ContinuePromptOverlay = 3

// ContinuePromptKey 分辨兩條。判斷要用它，不要拿字串比。
type ContinuePromptKey string

const (
	// ContinuePromptButton 是提到按鈕的那一條（`1165h`）。
	ContinuePromptButton ContinuePromptKey = "button"
	// ContinuePromptKeyboard 是只提鍵盤的那一條（`118Ah`）。
	ContinuePromptKeyboard ContinuePromptKey = "keyboard"
)

// ContinuePrompt 是一條提示。Text 是原版的字串本身。
type ContinuePrompt struct {
	Key    ContinuePromptKey
	Offset int
	Text   string
}

// continuePromptLayout 是兩條在 overlay-03 裡的位置，量自原版位元組。
var continuePromptLayout = []struct {
	key    ContinuePromptKey
	offset int
}{
	{ContinuePromptButton, 0x1165},
	{ContinuePromptKeyboard, 0x118a},
}

// ReadDOSContinuePrompts 從原版 ZIP 取那兩條字串。
func ReadDOSContinuePrompts(zipPath string) ([]ContinuePrompt, error) {
	executable, err := readStartExecutable(zipPath)
	if err != nil {
		return nil, err
	}
	overlayFile, err := readArchiveMember(zipPath, "GAME.OVR")
	if err != nil {
		return nil, err
	}
	overlays, err := tpov.Decode(executable, overlayFile)
	if err != nil {
		return nil, err
	}
	if len(overlays) <= ContinuePromptOverlay {
		return nil, fmt.Errorf("GAME.OVR has %d overlays, want more than %d",
			len(overlays), ContinuePromptOverlay)
	}
	code := overlays[ContinuePromptOverlay].Code
	prompts := make([]ContinuePrompt, 0, len(continuePromptLayout))
	for _, item := range continuePromptLayout {
		text, ok := pascalString(code, item.offset)
		if !ok {
			return nil, fmt.Errorf("overlay-%d has no Pascal string at %#x",
				ContinuePromptOverlay, item.offset)
		}
		prompts = append(prompts, ContinuePrompt{Key: item.key, Offset: item.offset, Text: text})
	}
	return prompts, nil
}

// DOSContinuePrompt 取單獨一條。
func DOSContinuePrompt(zipPath string, key ContinuePromptKey) (string, error) {
	prompts, err := ReadDOSContinuePrompts(zipPath)
	if err != nil {
		return "", err
	}
	for _, prompt := range prompts {
		if prompt.Key == key {
			return prompt.Text, nil
		}
	}
	return "", fmt.Errorf("continue prompt %q is absent", key)
}
