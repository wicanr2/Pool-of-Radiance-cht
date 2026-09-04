package main

import (
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// 原版靠 `11h PRINT` 把一句話拼起來，`12h PRINTCLEAR` 才是新的一頁。
// ecl7/23 的密碼確認框是三段：
//
//	A4A7 PRINTCLEAR "DO YOU REALLY MEAN"
//	A4B9 PRINT      <玩家打進去的字>
//	A4BD PRINT      "?"
//
// 把 11h 也當成取代的話，畫面上只會剩下最後那個問號，玩家看不懂在問什麼。
// 市政廳的 `A524 PRINTCLEAR #7 → PRINT 4A1Ah → PRINT #3` 是同一個模式。
func TestEventTextJoinsPrintOntoTheCurrentPage(t *testing.T) {
	for _, row := range []struct {
		name   string
		events []eclvm.Event
		want   string
	}{
		{
			name: "密碼確認框三段拼起來",
			events: []eclvm.Event{
				{Opcode: 0x12, Text: "DO YOU REALLY MEAN"},
				{Opcode: 0x11, Text: "NOKNOK"},
				{Opcode: 0x11, Text: "?"},
			},
			want: "DO YOU REALLY MEAN NOKNOK?",
		},
		{
			name: "PRINTCLEAR 是新的一頁，不接在上一頁後面",
			events: []eclvm.Event{
				{Opcode: 0x12, Text: "第一頁"},
				{Opcode: 0x12, Text: "第二頁"},
			},
			want: "第二頁",
		},
		{
			name: "PRINT RETURN 之後的 PRINTCLEAR 續行",
			events: []eclvm.Event{
				{Opcode: 0x12, Text: "第一行"},
				{Opcode: 0x33},
				{Opcode: 0x12, Text: "第二行"},
			},
			want: "第一行\n第二行",
		},
		{
			name: "CLEAR BOX 清掉整個框",
			events: []eclvm.Event{
				{Opcode: 0x12, Text: "會被清掉"},
				{Opcode: 0x3D},
				{Opcode: 0x11, Text: "新的"},
			},
			want: "新的",
		},
		{
			name: "市政廳的布告是一句，不是兩頁",
			events: []eclvm.Event{
				{Opcode: 0x12, Text: "PROCLAMATIONS ARE POSTED ON THE WALLS, IN YOUR JOURNAL YOU NOTE"},
				{Opcode: 0x11, Text: "PROCLAMATIONS LXIV, LXXVIII, CIX, AND LIX."},
			},
			want: "PROCLAMATIONS ARE POSTED ON THE WALLS, IN YOUR JOURNAL YOU NOTE PROCLAMATIONS LXIV, LXXVIII, CIX, AND LIX.",
		},
		{
			name: "片段自帶空白時不重複補",
			events: []eclvm.Event{
				{Opcode: 0x12, Text: "前面 "},
				{Opcode: 0x11, Text: "後面"},
			},
			want: "前面 後面",
		},
		{
			name: "空字串的事件不影響目前這一頁",
			events: []eclvm.Event{
				{Opcode: 0x12, Text: "留著"},
				{Opcode: 0x11, Text: ""},
			},
			want: "留著",
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			application := &app{}
			application.applyCellECLResult(eclvm.Result{Events: row.events})
			if application.eventText != row.want {
				t.Errorf("文字框是 %q，要的是 %q", application.eventText, row.want)
			}
		})
	}
}
