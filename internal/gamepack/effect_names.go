package gamepack

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// 效果的顯示名稱（spec 069）。原版在 overlay-15 `0F9Ch..12FFh` 用一串
// `cmp ax, 碼` 把效果碼換成字串，字串就內嵌在同一支 overlay 裡。
//
// 這串鏈也是 `3Dh` 的直接證據：spec 069 從預設人物推出「`3Dh` 只出現在戴著
// Ring of Fire Resistance 的人身上」，這裡它字面上就叫 "Fire Resistance"。
//
// `26h`（Gauntlets of Ogre Power）**不在這串鏈裡**——那件東西改的是力量，
// 顯示在能力值上，不是掛一個具名狀態。

//go:embed effect_names.zh-TW.json
var effectNamesJSON []byte

// EffectName 是一個效果碼的名稱。
type EffectName struct {
	// Code 是效果串列節點 `+0` 的代碼。
	Code uint8 `json:"code"`
	// Name 是原版的字串，逐字元照抄。
	Name string `json:"name"`
	// Text 是繁體中文。
	Text string `json:"text"`
}

// EffectNameTable 是整張表。
type EffectNameTable struct {
	Schema  string       `json:"schema"`
	Locale  string       `json:"locale"`
	Source  string       `json:"source"`
	Entries []EffectName `json:"entries"`
}

var effectNames EffectNameTable

func init() {
	if err := json.Unmarshal(effectNamesJSON, &effectNames); err != nil {
		panic(fmt.Sprintf("decode Pool effect names: %v", err))
	}
}

// EffectNames 回傳整張表，順序就是效果碼由小到大。
func EffectNames() []EffectName { return effectNames.Entries }

// EffectNameFor 依效果碼查名稱。查不到的碼回 false——原版有一部分效果碼
// 沒有顯示名稱，那是正確答案，不是缺漏。
func EffectNameFor(code uint8) (EffectName, bool) {
	for _, entry := range effectNames.Entries {
		if entry.Code == code {
			return entry, true
		}
	}
	return EffectName{}, false
}
