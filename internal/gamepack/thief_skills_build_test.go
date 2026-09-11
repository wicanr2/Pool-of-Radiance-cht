package gamepack

import "testing"

// 三個樣本是 dosgolem 用原版 START.EXE 建出來的一級賊（spec 095），
// `.CHA` 的 `+77h` 八格逐位元組抄在這裡。
//
// 兩種種族、兩種敏捷：矮人的種族修正非零（含一個要夾到 0 的），人類全 0；
// 兩人的敏捷差兩點，敏捷表那兩列差 `-5 0 -5 -10 0`。**實測沒有那個差**，
// 所以建角不看敏捷這件事是這兩個樣本自己交叉證的。
func TestThiefSkillsMatchTheOriginalCharacterFiles(t *testing.T) {
	tables, err := ReadDOSThiefSkillTables(poolZipPath())
	if err != nil {
		t.Fatalf("讀賊技能表：%v", err)
	}
	for _, want := range []struct {
		name      string
		level     int
		race      int
		dexterity int
		skills    ThiefSkills
	}{
		{"矮人賊", 1, 1, 11, ThiefSkills{35, 45, 40, 15, 10, 10, 75, 0}},
		{"人類賊", 1, 7, 13, ThiefSkills{35, 35, 25, 15, 10, 10, 85, 0}},
		{"人類賊 2", 1, 7, 13, ThiefSkills{35, 35, 25, 15, 10, 10, 85, 0}},
	} {
		got, err := tables.Build(want.level, want.race, ThiefSkillBuildDexterity)
		if err != nil {
			t.Fatalf("%s：%v", want.name, err)
		}
		if got != want.skills {
			t.Errorf("%s 的八格是 %v，原版是 %v", want.name, got, want.skills)
		}
		// 正對照：帶真實敏捷進去一定要對不上，否則這個測試證明不了
		// 「建角不看敏捷」——兩邊都過的話是表讀錯了。
		withDexterity, err := tables.Build(want.level, want.race, want.dexterity)
		if err != nil {
			t.Fatalf("%s 帶敏捷：%v", want.name, err)
		}
		if withDexterity == want.skills {
			t.Errorf("%s 帶敏捷 %d 也算出 %v——那表示敏捷那一段根本沒生效，表讀錯了",
				want.name, want.dexterity, got)
		}
	}
}

// 非賊是八格全 0，而且那是正確答案不是「還沒做」。
func TestThiefSkillsAreZeroForNonThieves(t *testing.T) {
	tables, err := ReadDOSThiefSkillTables(poolZipPath())
	if err != nil {
		t.Fatalf("讀賊技能表：%v", err)
	}
	got, err := tables.Build(0, 7, ThiefSkillBuildDexterity)
	if err != nil {
		t.Fatalf("Build：%v", err)
	}
	if got != (ThiefSkills{}) {
		t.Errorf("非賊算出 %v，應該全 0", got)
	}
}

// 第 9 級那一列是表的另一端，也是基礎表與種族表重疊的那一列
// （基礎表等級 9 就是種族表種族碼 0），拆成三個陣列的人會在這裡對不上。
//
// 預設人物 TINA 是第 9 級賊，八格是 `80 77 65 80 66 30 98 45`——**與這裡
// 算出來的不同**。她是 SSI 手工填的預設人物，不是這支算出來的，所以不能
// 拿她當這條算式的驗收樣本。
func TestThiefSkillsAtNinthLevelStayInsideTheTable(t *testing.T) {
	tables, err := ReadDOSThiefSkillTables(poolZipPath())
	if err != nil {
		t.Fatalf("讀賊技能表：%v", err)
	}
	got, err := tables.Build(9, 1, ThiefSkillBuildDexterity)
	if err != nil {
		t.Fatalf("Build：%v", err)
	}
	// 第 9 級的基礎值是 `70 62 60 70 56 30 98 45`，矮人的種族修正是
	// `0 10 15 0 0 0 -10 -5`，建角那一列敏捷是 `5 10 5 0 0`。
	want := ThiefSkills{75, 82, 80, 70, 56, 30, 88, 40}
	if got != want {
		t.Errorf("第 9 級矮人賊算出 %v，照三張表應該是 %v", got, want)
	}
}
