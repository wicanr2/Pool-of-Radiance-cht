#!/usr/bin/env python3
"""原版在導覽終點那一格紮營休息兩小時會怎樣（#42）。

remake 在同一格休息完會停在格子文字，內容是導覽的結尾句（'OUR TOUR IS ENDED...'），
而不是回到冒險畫面；發行包對拍因此停在第 33 張。**原版怎麼收尾沒有人量過**——
現有的 dosgolem 基準（`ref-spells`）走的是 `e,m,m`，沒有休息。

做法：照參考基準的鍵序走完建角與導覽，停在城區 (0,4)，然後
`e`（紮營）→ `r`（休息）→ `h`（小時）→ `i`,`i`（+2）→ `r`（開始休息），
每一步抓畫面與 `DS:6A0B` 的座標、時鐘（class 0 `49C9h`）。

輸出 docs/audit/dos-tour-cell-rest.json 與 workplace/tour-cell-rest/*.png。
用法：python3 tools/dosgolem-tour-cell-rest.py（主機端會自己起容器）"""
import hashlib, json, os, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WORK = os.path.join(ROOT, 'workplace', 'tour-cell-rest')
if not os.path.exists('/bin/shots'):
    os.makedirs(os.path.join(WORK, 'scratch'), exist_ok=True)
    # 字型表是駕駛認字用的，與其他 dosgolem 工具共用同一份。
    font = os.path.join(ROOT, 'workplace', 'dosgolem-cheat', 'font.json')
    if not os.path.exists(os.path.join(WORK, 'font.json')):
        subprocess.check_call(['cp', font, os.path.join(WORK, 'font.json')])
    sys.exit(subprocess.call(
        ['timeout', '1800', 'docker', 'run', '--rm', '-i', '--network', 'none', '--memory', '4g', '--cpus', '2',
         '--pids-limit', '256', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
         '-u', '%d:%d' % (os.getuid(), os.getgid()),
         '-v', os.path.join(ROOT, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots') + ':/bin/shots:ro',
         '-v', os.path.join(ROOT, 'workplace', 'oracle', 'dos') + ':/orig:ro',
         '-v', os.path.join(ROOT, 'tools') + ':/tools:ro',
         '-v', WORK + ':/work', '-v', os.path.join(WORK, 'scratch') + ':/scratch',
         '-e', 'PYTHONUNBUFFERED=1', '-e', 'PYTHONPATH=/tools', '-w', '/work',
         'python:3.13-alpine', 'python3', '/tools/dosgolem-tour-cell-rest.py'] + sys.argv[1:]))

sys.path.insert(0, '/tools')
import dos_text
import dosgolem_drive as D

OUT = '/work'
# 建角 → 進城 → 導覽跑完，停在 (0,4)。與 `workplace/dosgolem-ref/provenance.json`
# 的鍵序同一段（那一份多走了地圖、檢視等等，這裡只要導覽結束那一刻）。
BOOT = 'rep:9:Space,Return,Return,c,Return,Return,Return,Return,Return,Return,y,H,E,R,O,Return,k,e,y,a,a,e,b,rep:14:Return'
# 紮營休息兩小時：e 紮營、r 休息、h 選小時、i i 加兩小時、r 開始休息。
REST = ['e', 'r', 'h', 'i', 'i', 'r']


def shot(d, name):
    idx = d.frame()
    open(os.path.join(OUT, name + '.idx'), 'wb').write(idx)
    dos_text.write_png(os.path.join(OUT, name + '.png'), idx)
    screen = D.Screen(idx, d.font)
    rows = [r for r in screen.rows if r.strip()]
    where = d.where()
    print('==', name, where)
    print('\n'.join(rows))
    return {'name': name, 'rows': rows, 'bottom': screen.bottom(),
            'options': [[w, h] for w, h, _ in screen.options()],
            'where': where, 'clock_49C9': d.class0(0x49C9),
            'map_exit_flag_6DD5': d.class1(0x6DD5),
            'sha256': hashlib.sha256(idx).hexdigest()}


def main():
    d = D.Dos(work=OUT, keys=BOOT, intercept=False)
    d.wait()
    # 導覽是自己跑的，每一頁等一次按鍵。**按到它結束為止**：只送固定次數的 Return
    # 會停在導覽中間，而那時候按 `e` 不是紮營（第一次跑就是這樣，畫面停在
    # 「ON YOUR RIGHT IS ONE OF THE…」）。判準是底列不再要求按鍵。
    # 判準是**冒險畫面的指令列**（`AREA CAST VIEW ENCAMP SEARCH LOOK`），不是
    # 「底列沒有 PRESS」——導覽翻頁的空檔底列就是空的，用它當判準會在導覽中間停下來
    # （第二次跑就是這樣，接著按 `e` 又翻出一頁導覽）。
    for page in range(120):
        screen = D.Screen(d.frame(), d.font)
        if 'ENCAMP' in screen.bottom().upper():
            break
        d.key('Return')
        d.wait()
    else:
        raise SystemExit('導覽沒有結束')
    steps = [shot(d, 'tour-end')]
    for index, key in enumerate(REST):
        d.key(key)
        d.wait()
        steps.append(shot(d, 'rest-%d-%s' % (index, key)))
    # 休息跑完之後畫面會自己動一陣子（時間推進）。多等幾次，直到畫面不再變。
    last = None
    for settle in range(12):
        d.wait()
        idx = d.frame()
        digest = hashlib.sha256(idx).hexdigest()
        if digest == last:
            break
        last = digest
    steps.append(shot(d, 'after-rest'))
    # 打斷之後的兩個出口各量一次：`STAY` 在原地、`GO` 走人。存檔再讀回來，
    # 兩條路才是從同一個狀態分出去的。
    d.save('/work/interrupted.state')
    for key, name in (('s', 'after-stay'), ('g', 'after-go')):
        d.key(key)
        d.wait()
        steps.append(shot(d, name))
        # 回到打斷那一刻再量另一條。
        d.quit()
        d = D.Dos(work=OUT, load='/work/interrupted.state', intercept=False)
        d.wait()
    receipt = {'schema': 'pool-dos-tour-cell-rest/1', 'generator': 'dosgolem',
               'boot_keys': BOOT, 'rest_keys': REST,
               'original_exe_sha256': hashlib.sha256(open('/orig/start.exe', 'rb').read()).hexdigest(),
               'steps': steps}
    path = '/work/dos-tour-cell-rest.json'
    json.dump(receipt, open(path, 'w'), ensure_ascii=False, indent=2)
    open(path, 'a').write('\n')
    print('寫出', path)
    d.quit()


main()
