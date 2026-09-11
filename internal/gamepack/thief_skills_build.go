package gamepack

import "fmt"

// 建角時算那八格的是 overlay-23 entry 4（`031Eh`），spec 095 有逐行反組譯。
// 整支是一個 `i = 1..8` 的迴圈：
//
//	記錄[+76h + i] = 基礎[賊等級 × 8 + i] + 種族[種族碼 × 8 + i]
//	i <= 5 再加   敏捷[DEX × 5 + i]
//
// 種族修正為負且絕對值大於基礎，那一格直接夾到 0。
const (
	// ThiefSkillBaseTableOffset 是基礎表的 DS 位移。
	ThiefSkillBaseTableOffset = 0x3c6b
	// ThiefSkillRaceTableOffset 是種族修正表的 DS 位移。
	ThiefSkillRaceTableOffset = 0x3cb3
	// ThiefSkillDexterityTableOffset 是敏捷修正表的 DS 位移。
	ThiefSkillDexterityTableOffset = 0x3cc6
	// ThiefSkillDexterityRowWidth 是敏捷表的列寬。它是 5 不是 8——
	// 後三項（聽聲、爬牆、讀語言）沒有敏捷修正，表裡也沒有那三格。
	ThiefSkillDexterityRowWidth = 5
	// ThiefSkillDexteritySkills 是吃敏捷修正的技能數，也就是 entry 4 的
	// `i >= 6 → 這一格做完` 那道閘門。
	ThiefSkillDexteritySkills = 5

	// ThiefSkillBuildDexterity 是建角時套進敏捷表的索引。
	//
	// **是 0，不是角色的敏捷。** 建角填記錄的順序是先職業與等級、後能力值：
	// overlay-16 在 `0CA7h` 判 `記錄 +9Ch > 0`（賊等級當時已經有值）之後
	// `0CAFh` 就呼叫 entry 4，而寫能力值那一段在 `1206h` 以後。entry 4 讀
	// `記錄 +13h` 讀到的是還沒填的 0，所以敏捷那一段套的一律是第 0 列
	// `+5 +10 +5 0 0`。
	//
	// 三個原版樣本（矮人 DEX 11、人類 DEX 13 兩個）帶 0 進去逐格吻合，
	// 帶真實敏捷進去三個全錯——兩種種族、兩種敏捷交叉證的，不是湊出來的。
	ThiefSkillBuildDexterity = 0

	// ThiefLevelIndex 是賊在八個單一職業等級裡的位置（記錄 `+9Ch`）。
	ThiefLevelIndex = 6

	// thiefSkillTableSpan 是三張表攤平之後要讀的長度：從基礎表起點到
	// 敏捷表最後一列（能力值上限 18）的結尾。
	thiefSkillTableSpan = ThiefSkillDexterityTableOffset - ThiefSkillBaseTableOffset +
		18*ThiefSkillDexterityRowWidth + ThiefSkillDexterityRowWidth + 1
)

// ThiefSkillTables 是建角要用的三張表。
//
// 攤平存一整段，不拆成三個二維陣列：**基礎表與種族表在原版是重疊的**——
// 基礎表等級 9 那一列（DS `3C6Bh + 72 + 1`）就是種族表種族碼 0 那一列
// （DS `3CB3h + 0 + 1`），同一段位元組兩張表各認各的。種族碼 0 是怪物，
// 走不到建角，所以原版就這樣放著。拆成陣列會把這件事弄丟，之後有人照
// 「一張表一個陣列」去改就會對不上原版的位元組。
type ThiefSkillTables struct {
	// Data 是 DS `ThiefSkillBaseTableOffset` 起的一整段，索引照原版的算式。
	Data []int8
}

// ParseThiefSkillTables 從 START.EXE 的完整內容取出那一段。
func ParseThiefSkillTables(executable []byte) (ThiefSkillTables, error) {
	start := ThiefSkillBaseTableOffset + startDataSegmentFileDelta
	if start < 0 || start+thiefSkillTableSpan > len(executable) {
		return ThiefSkillTables{}, fmt.Errorf("Pool START.EXE is %d bytes, DS:%04Xh needs %d more",
			len(executable), ThiefSkillBaseTableOffset, start+thiefSkillTableSpan-len(executable))
	}
	data := make([]int8, thiefSkillTableSpan)
	for index, value := range executable[start : start+thiefSkillTableSpan] {
		data[index] = int8(value)
	}
	return ThiefSkillTables{Data: data}, nil
}

// ReadDOSThiefSkillTables 從遊戲壓縮檔取出那三張表。
func ReadDOSThiefSkillTables(zipPath string) (ThiefSkillTables, error) {
	executable, err := readStartExecutable(zipPath)
	if err != nil {
		return ThiefSkillTables{}, err
	}
	return ParseThiefSkillTables(executable)
}

// at 把 DS 位移換成這一段裡的索引。
func (t ThiefSkillTables) at(ds int) (int, error) {
	index := ds - ThiefSkillBaseTableOffset
	if index < 0 || index >= len(t.Data) {
		return 0, fmt.Errorf("Pool thief skill tables have %d bytes, DS:%04Xh is outside", len(t.Data), ds)
	}
	return int(t.Data[index]), nil
}

// Build 照 entry 4 算出八格。dexterity 傳 ThiefSkillBuildDexterity 就是建角的
// 行為；升級／訓練那條路徑還沒查（那時候記錄裡的敏捷是有值的），所以這裡把
// 它留成參數而不是寫死。
func (t ThiefSkillTables) Build(thiefLevel, raceCode, dexterity int) (ThiefSkills, error) {
	var skills ThiefSkills
	if thiefLevel <= 0 {
		// 不是賊，八格就是 0——那是正確答案不是佔位。
		return skills, nil
	}
	for i := 1; i <= ThiefSkillCount; i++ {
		base, err := t.at(ThiefSkillBaseTableOffset + thiefLevel*ThiefSkillCount + i)
		if err != nil {
			return skills, err
		}
		racial, err := t.at(ThiefSkillRaceTableOffset + raceCode*ThiefSkillCount + i)
		if err != nil {
			return skills, err
		}
		if racial < 0 && -racial > base {
			skills[i-1] = 0
			continue
		}
		value := base + racial
		if i <= ThiefSkillDexteritySkills {
			adjust, err := t.at(ThiefSkillDexterityTableOffset + dexterity*ThiefSkillDexterityRowWidth + i)
			if err != nil {
				return skills, err
			}
			value += adjust
		}
		// 原版存的是 `mov [di+76h+i], al`，只取低位。夾限只管種族那一步，
		// 敏捷讓它變負的時候原版就是繞回去——照抄，不自作主張補夾限。
		skills[i-1] = uint8(value)
	}
	return skills, nil
}
