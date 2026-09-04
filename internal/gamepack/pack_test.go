package gamepack_test

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// pack 要能經由共用 engine 的載入器讀進來——那是這一整件事的重點：
// 內容留在作品這一側，engine 只認 `engine.Pack` 這個結構。
func TestGamePackLoadsThroughTheSharedEngine(t *testing.T) {
	pack, err := gamepack.Pack()
	if err != nil {
		t.Fatalf("載入 Pool 的 game pack：%v", err)
	}
	if pack.SchemaVersion != 1 {
		t.Errorf("schema_version 是 %d，應該是 1", pack.SchemaVersion)
	}
	if pack.ID != "pool-of-radiance.phlan" {
		t.Errorf("id 是 %q", pack.ID)
	}
	if pack.DefaultLocale != "en" {
		t.Errorf("default_locale 是 %q", pack.DefaultLocale)
	}
	if pack.Presentation == nil {
		t.Fatal("pack 沒有 presentation")
	}
	// 原生畫布 320×200，前端用兩倍放大成 640×400（`cmd/pool-game` 的
	// logicalWidth／logicalHeight）。這三個數字改了畫面就會跑掉。
	if pack.Presentation.NativeWidth != 320 || pack.Presentation.NativeHeight != 200 ||
		pack.Presentation.NativeScale != 2 {
		t.Errorf("presentation 是 %d×%d ×%d，應該是 320×200 ×2",
			pack.Presentation.NativeWidth, pack.Presentation.NativeHeight,
			pack.Presentation.NativeScale)
	}
}

// 分檔的合併順序由檔名決定，所以列出來的順序本身要是排好的。
func TestGamePackPartsAreListedInMergeOrder(t *testing.T) {
	names, err := gamepack.PackPartNames()
	if err != nil {
		t.Fatalf("列出 pack 分檔：%v", err)
	}
	if len(names) < 3 {
		t.Fatalf("只有 %d 個分檔：%v", len(names), names)
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("分檔沒有照檔名排序：%v", names)
	}
	if names[0] != "pack/00-core.json" {
		t.Errorf("第一個分檔是 %q，header 應該在 00-core.json", names[0])
	}
}

// 兩個語言的鍵要完全一樣。少一邊的症狀是介面一半中文一半英文，
// 而那在測試裡不會自己冒出來。
func TestGamePackLocalesHaveTheSameKeys(t *testing.T) {
	english, err := gamepack.LocaleTable("en")
	if err != nil {
		t.Fatalf("英文表：%v", err)
	}
	chinese, err := gamepack.LocaleTable("zh-TW")
	if err != nil {
		t.Fatalf("繁中表：%v", err)
	}
	if len(english) == 0 {
		t.Fatal("英文表是空的")
	}
	for key := range english {
		if _, ok := chinese[key]; !ok {
			t.Errorf("%q 只有英文", key)
		}
	}
	for key := range chinese {
		if _, ok := english[key]; !ok {
			t.Errorf("%q 只有繁中", key)
		}
	}
	// 負對照：查一個不存在的語言要回錯，不是回空表——空表的症狀是整個介面
	// 變成空字串，看起來像繪製壞掉。
	if _, err := gamepack.LocaleTable("klingon"); err == nil {
		t.Error("不存在的語言沒有回錯")
	}
}

// **不得抄 CoAB 的資料**。這一則是機械檢查：pack 的每一個位元組裡都不准出現
// CoAB 專有的識別字。沒有它，「沒有抄」只是宣稱。
func TestGamePackCarriesNoAzureBondsContent(t *testing.T) {
	forbidden := []string{
		"curse-of-the-azure-bonds", "azure", "Tilverton", "Yulash", "Zhentil",
		"Moander", "Dragonspear", "Phlan is not in CoAB",
	}
	names, err := gamepack.PackPartNames()
	if err != nil {
		t.Fatalf("列出 pack 分檔：%v", err)
	}
	pack, err := gamepack.Pack()
	if err != nil {
		t.Fatalf("載入 pack：%v", err)
	}
	encoded, err := json.Marshal(pack)
	if err != nil {
		t.Fatalf("序列化 pack：%v", err)
	}
	lowered := strings.ToLower(string(encoded))
	for _, word := range forbidden {
		if strings.Contains(lowered, strings.ToLower(word)) {
			t.Errorf("pack 裡出現了 CoAB 的字眼 %q", word)
		}
	}
	// 正對照：檢查真的看得到內容，不是掃了一份空字串。
	if !strings.Contains(lowered, "pool-of-radiance") {
		t.Fatalf("掃描面有洞：%d 個分檔序列化之後找不到 Pool 自己的 id", len(names))
	}
}
