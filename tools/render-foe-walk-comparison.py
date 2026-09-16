#!/usr/bin/env python3
"""把三場固定事件的敵方走位並排成 docs/audit/foe-walk-comparison.md：原版那一側讀
docs/audit/dosgolem-deployment-peek-{orc-home,guards,alarm}.json（tools/dosgolem-drive-orc-home.py
的收據），remake 那一側讀 `go test -run TestDeploymentMatchesTheSlumsReceipts -v` 的 log
（先存成 workplace/remake-rounds.txt，只留含 "round" 的行）。"""
import json, os, re, sys
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
NAMES = {'orc-home': '獸人的家 (3,3) N', 'guards': '衛兵攔截 (0,7) N', 'alarm': '驚動衛兵 (3,11) E'}
log_path = sys.argv[1] if len(sys.argv) > 1 else os.path.join(ROOT, 'workplace', 'remake-rounds.txt')
lines = open(log_path).read().split('\n')
idx = [i for i, l in enumerate(lines) if 'round 1:' in l]
assert len(idx) == 3, '要三場的 log（TestDeploymentMatchesTheSlumsReceipts 三個子測試）'
out = ['# 貧民窟三場固定事件：敵方走位逐格對照（原版 dosgolem vs remake，同狀態）', '',
       '由 `tools/dosgolem-drive-orc-home.py` 的收據（`docs/audit/dosgolem-deployment-peek-{orc-home,guards,alarm}.json`）與',
       '`TestDeploymentMatchesTheSlumsReceipts` 的 log 產生（`tools/render-foe-walk-comparison.py`）。兩邊同一格、同朝向、同人數、',
       '距離 0；原版隊員每回合 DONE→GUARD，remake 隊員 ENTER 結束回合。原版的「批」是五個隊員各按一次 d,g，',
       '期間敵方依先攻穿插行動，表裡每一列是位置表有變動的那一幀；remake 的列是回合邊界。`x` 標死掉（體型類別 0）。', '']
summary = []
def cell(v):
    x, y, c = v
    return '(%d,%d)%s' % (x, y, 'x' if c == 0 else '')
for k, name in enumerate(['orc-home', 'guards', 'alarm']):
    rounds = json.load(open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-deployment-peek-%s.json' % name)))['rounds']
    seg = lines[idx[k]:idx[k + 1]] if k + 1 < len(idx) else lines[idx[k]:]
    tabs = []
    for l in seg:
        m = re.search(r'round (\d+):(.*)', l)
        if not m:
            continue
        cells = {int(a): (int(b), int(c), int(e)) for a, b, c, e in re.findall(r'(\d+):\((\d+),(\d+),(\d+)\)', m.group(2))}
        tabs.append((int(m.group(1)), cells))
    n = len(rounds[0]['table'])
    out += ['## %s' % NAMES[name], '', '### 原版（每一列是位置表變動的一幀；批 1 = 第一輪五個 d,g）', '']
    hdr = '| 幀 | ' + ' | '.join(str(i) for i in range(1, n + 1)) + ' |'
    out += [hdr, '|' + '---|' * (n + 1)]
    for r in rounds:
        cells = {i: (x, y, c) for i, x, y, c in r['table']}
        out.append('| 批%d 幀%d %s | ' % (r['round_batch'], r['frame'], r['label']) + ' | '.join(cell(cells[i]) for i in range(1, n + 1)) + ' |')
    out += ['', '### remake（每一列是回合開頭）', '', hdr, '|' + '---|' * (n + 1)]
    for r, cells in tabs:
        out.append('| 第 %d 輪 | ' % r + ' | '.join(cell(cells[i]) if i in cells else '—' for i in range(1, n + 1)) + ' |')
    o_first = [r for r in rounds if r['round_batch'] == 1][-1]
    o_cells = {i: (x, y) for i, x, y, c in o_first['table']}
    r_cells = {i: v[:2] for i, v in tabs[1][1].items()} if len(tabs) > 1 else {}
    start = {i: (x, y) for i, x, y, c in rounds[0]['table']}
    same = [i for i in range(6, n + 1) if o_cells.get(i) == r_cells.get(i)]
    moved_o = [i for i in range(6, n + 1) if o_cells[i] != start[i]]
    moved_r = [i for i in range(6, n + 1) if r_cells.get(i) != start[i]]
    both = [i for i in same if i in moved_o]
    out += ['', '第一輪之後（原版批 1 結束 vs remake 第 2 輪開頭）：敵方 %d 隻，終點同格 %d 隻；原版動了 %d 隻 %s，remake 動了 %d 隻 %s；兩邊都動而且同格的：%s。' % (
        n - 5, len(same), len(moved_o), moved_o, len(moved_r), moved_r, both), '']
    summary.append((NAMES[name], n - 5, len(same), len(moved_o), len(moved_r), both))
out += ['## 一句話', '']
for nm, foes, same, mo, mr, both in summary:
    out.append('- %s：%d 隻，第一輪後終點同格 %d 隻（原版動 %d、remake 動 %d，都動而且同格 %s）。' % (nm, foes, same, mo, mr, both))
out += ['', '不同的那些：誰先動（先攻擲骰）、追誰（`38A6h` 從候選名單擲骰）與 GUARD 攻擊造成的死亡順序；規則層（五偏移依序試、卡住換模式、死者不擋路、一輪走幾格）在三場都沒有量到相反證據。動作層的對照（逐步快照加骰流，`TestFoeWalkReproducesEveryOriginalAction`）三場 40 個原版動作 remake 全部走到同一格：模式、目標與卡住重挑的骰照原版餵，走位規則是同一套（spec 096〈骰流對回去之後〉）。這一張表兩邊各用自己的骰，差的只剩先攻與擲骰本身。']
target = os.path.join(ROOT, 'docs', 'audit', 'foe-walk-comparison.md')
open(target, 'w').write('\n'.join(out) + '\n')
print('寫出', target)
