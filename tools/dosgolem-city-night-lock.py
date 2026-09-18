#!/usr/bin/env python3
"""原版晚上站上鎖門格會怎樣（#39）。

spec 102 讀出城區入口 0 的 `ecl3/0 9920h`：`49C9 >= 14` 時，面前是特定牆型的門
會出現 "THE DOOR IS LOCKED. DO YOU WANT TO BREAK IN?"。靜態分支 exact、比較方向
已用 dosgolem 對過，**缺的是 runtime 收據**——原版晚上站在那一格的畫面沒拍過。

做法：照參考基準的鍵序走完建角與導覽，紮營把時鐘排到 14 點以後（不注入），
再走到市政廳門口那一格朝東，拍畫面並讀 `6E7D`。正對照是同一條路線在白天要進得去市政廳。

輸出 docs/audit/dos-city-night-lock.json 與 workplace/city-night-lock/*.png。
用法：python3 tools/dosgolem-city-night-lock.py（主機端會自己起容器）"""
import hashlib, importlib.util, json, os, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WORK = os.path.join(ROOT, 'workplace', 'dosgolem-cheat')
OUT = os.path.join(WORK, 'night-lock')
if not os.path.exists('/bin/shots'):
    os.makedirs(os.path.join(WORK, 'scratch'), exist_ok=True)
    sys.exit(subprocess.call(
        ['timeout', '3600', 'docker', 'run', '--rm', '-i', '--network', 'none', '--memory', '4g', '--cpus', '2',
         '--pids-limit', '256', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
         '-u', '%d:%d' % (os.getuid(), os.getgid()),
         '-v', os.path.join(ROOT, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots') + ':/bin/shots:ro',
         '-v', os.path.join(ROOT, 'workplace', 'oracle', 'dos') + ':/orig:ro',
         '-v', os.path.join(ROOT, 'tools') + ':/tools:ro',
         '-v', WORK + ':/work', '-v', os.path.join(WORK, 'scratch') + ':/scratch',
         '-e', 'PYTHONUNBUFFERED=1', '-e', 'PYTHONPATH=/tools', '-w', '/work',
         'python:3.13-alpine', 'python3', '/tools/dosgolem-city-night-lock.py'] + sys.argv[1:]))

sys.path.insert(0, '/tools')
import dos_text
import dosgolem_drive as D

spec = importlib.util.spec_from_file_location('cheat', '/tools/dosgolem-cheat-playthrough.py')
cheat = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cheat)

OUT = '/work/night-lock'
BOOT = ('rep:9:Space,Return,Return,c,Return,Return,Return,Return,Return,Return,y,H,E,R,O,'
        'Return,k,e,y,a,a,e,b,rep:14:Return')


def snapshot(dr, name):
    screen = dr.screen()
    idx = screen.idx
    open(os.path.join(OUT, name + '.idx'), 'wb').write(idx)
    dos_text.write_png(os.path.join(OUT, name + '.png'), idx)
    rows = [r for r in screen.rows if r.strip()]
    record = {'name': name, 'rows': rows, 'bottom': screen.bottom(),
              'options': [[w, h] for w, h, _ in screen.options()],
              'where': dr.d.where(), 'clock_49C9': dr.d.class0(0x49C9),
              'door_kind_6E7D': dr.d.class1(0x6E7D),
              'sha256': hashlib.sha256(idx).hexdigest()}
    print('==', name, record['where'], '49C9=%d 6E7D=%d' % (record['clock_49C9'], record['door_kind_6E7D']),
          '底列', repr(record['bottom'])[:56])
    for row in rows[2:7]:
        text = row.strip('? ')
        if text:
            print('   ', text[:62])
    return record


def walk_east_to_city_hall(dr):
    """導覽終點 (0,4) 往東走到市政廳門口 (3,4)，朝東。"""
    for _ in range(6):
        if dr.d.where()['x'] >= 3:
            break
        dr.face(1)
        dr.press('Up')
        dr.settle()
    dr.face(1)


def main():
    os.makedirs(OUT, exist_ok=True)
    original = D.Dos
    D.Dos = lambda *a, **k: original(*a, **dict(k, intercept=False))
    dr = cheat.Driver('night-lock', load=None, keys=BOOT)
    D.Dos = original
    for page in range(120):
        if 'ENCAMP' in dr.screen().bottom().upper():
            break
        dr.press('Return')
    steps = [snapshot(dr, '00-tour-end')]
    # 白天那條從新遊戲走；夜間那條改從 `chain1-sokal.state` 起跑——它已經站在
    # 市政廳門口、時鐘 13 點、六人隊伍（推時鐘與活下來都便宜得多，見 `night`）。

    # 正對照：白天走到市政廳門口，市政廳要進得去。
    walk_east_to_city_hall(dr)
    steps.append(snapshot(dr, '01-day-at-door'))
    dr.press('Up')
    dr.settle()
    steps.append(snapshot(dr, '02-day-inside'))

    dr.d.quit()
    original2 = D.Dos
    D.Dos = lambda *a, **k: original2(*a, **dict(k, intercept=False))
    dr = cheat.Driver('night-lock', load='/work/chain1-sokal.state')
    D.Dos = original2
    dr.settle()

    try:
        night(dr, steps)
    except Exception as failure:  # noqa: BLE001 - 量到一半也要留下收據
        print('夜間那一段停住了：%s' % failure)
    write_receipt(steps)
    dr.d.quit()


def night(dr, steps):
    """把時鐘推過 14 點再走到市政廳門口。

    推時鐘的方法是**走路**（一步一分鐘）：城區街上紮營會被城衛隊在 00:05 攔下來，
    十二次只推進一小時；貧民窟睡得完但會跳遭遇，一人隊伍收不了場。
    起點改用 `chain1-sokal.state`——它已經站在市政廳門口 (3,4)、時鐘 13 點、六人隊伍。
    """
    for attempt in range(400):
        if dr.d.class0(0x49C9) >= 14:
            break
        # 在門口附近來回踏：往東一步會進市政廳，所以改成往西、往東各一步。
        dr.face(3)
        dr.press('Up')
        dr.settle()
        dr.face(1)
        if dr.d.where()['x'] < 3:
            dr.press('Up')
            dr.settle()
        if attempt % 20 == 0:
            print('   走路推時鐘第 %d 步：49C9=%d 位置=%s' % (attempt, dr.d.class0(0x49C9), dr.d.where()))
    steps.append(snapshot(dr, '03-night'))
    # 起點就是門口那一格，不必再規劃路線（`walk_to` 收的是判準函式不是座標）。
    dr.face(1)
    steps.append(snapshot(dr, '04-night-at-door'))
    dr.press('Up')
    dr.settle()
    steps.append(snapshot(dr, '05-night-door'))


def write_receipt(steps):
    receipt = {'schema': 'pool-dos-city-night-lock/1', 'generator': 'dosgolem', 'boot_keys': BOOT,
               'original_exe_sha256': hashlib.sha256(open('/orig/start.exe', 'rb').read()).hexdigest(),
               'steps': steps}
    json.dump(receipt, open('/work/dos-city-night-lock.json', 'w'), ensure_ascii=False, indent=2)
    open('/work/dos-city-night-lock.json', 'a').write('\n')
    print('寫出 /work/dos-city-night-lock.json（%d 張）' % len(steps))


main()
