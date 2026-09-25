#!/usr/bin/env python3
"""原版隊伍被打光之後停在哪（#19）。

remake 目前全滅回標題（`returnToTitleAfterGameOver`），而那是 hypothesis：
overlay-05 `14CAh` 那一段把 `4961h` 設成 1 交回主迴圈，**誰讀它還沒讀出來**。
這一支拿原版當裁判：一支一級隊伍走進貧民窟、照正常按鍵打到全滅，
記下每一步的畫面、`4961h`、隊伍記錄的狀態位元組。

**扣血攔截要關掉**——那是作弊通關用的，開著就打不死。

輸出 docs/audit/dos-party-wipe.json 與 workplace/dosgolem-cheat/wipe/*.png。

全滅後原版顯示 THE END 頁，在 CRT `ReadKey`（`0622:031A` `int 16h AH=00h`）等一個鍵；
按下去之後主迴圈（ovr3 entry 1）看到 `DS:4960` 非零就返回，主程式呼叫 `037B:0000`
切回文字模式並 `Halt(0)`，**程式正常結束回 DOS**（查證見 `docs/re/dos-party-wipe-exit.md`）。
要用 dosgolem `2ab6f65` 之後的 shots：之前的版本讀鍵不阻塞，那一頁會自己跑過去，
結束後還繼續步進（issue #53）。serve 回 `exited: true` 時這一支停下來記收據，
按鍵之前那一刻存成 `wipe-anykey.state`。
用法：python3 tools/dosgolem-party-wipe.py（主機端會自己起容器）"""
import hashlib, importlib.util, json, os, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WORK = os.path.join(ROOT, 'workplace', 'dosgolem-cheat')
if not os.path.exists('/bin/shots'):
    os.makedirs(os.path.join(WORK, 'scratch'), exist_ok=True)
    # 收據記下 shots 是哪一版建的（issue #53 之前的版本會把全滅頁跑過頭）。
    commit = subprocess.run(['git', '-C', os.path.join(ROOT, 'workplace', 'dosgolem'), 'rev-parse', 'HEAD'],
                            capture_output=True, text=True).stdout.strip()
    sys.exit(subprocess.call(
        ['timeout', '3600', 'docker', 'run', '--rm', '-i', '--network', 'none', '--memory', '4g', '--cpus', '2',
         '--pids-limit', '256', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
         '-u', '%d:%d' % (os.getuid(), os.getgid()),
         '-v', os.path.join(ROOT, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots') + ':/bin/shots:ro',
         '-v', os.path.join(ROOT, 'workplace', 'oracle', 'dos') + ':/orig:ro',
         '-v', os.path.join(ROOT, 'tools') + ':/tools:ro',
         '-v', WORK + ':/work', '-v', os.path.join(WORK, 'scratch') + ':/scratch',
         '-e', 'PYTHONUNBUFFERED=1', '-e', 'PYTHONPATH=/tools', '-e', 'DOSGOLEM_COMMIT=' + commit, '-w', '/work',
         'python:3.13-alpine', 'python3', '/tools/dosgolem-party-wipe.py'] + sys.argv[1:]))

sys.path.insert(0, '/tools')
import dos_text
import dosgolem_drive as D

# 作弊通關那一支的駕駛（畫面判斷、settle、走路）整包重用；檔名有連字號，只能這樣載。
spec = importlib.util.spec_from_file_location('cheat', '/tools/dosgolem-cheat-playthrough.py')
cheat = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cheat)

OUT = '/work/wipe'
# 建一個一級戰士 → 進城 → 導覽跑完 → 往西一步進貧民窟。
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
              'where': dr.d.where(), 'gameover_4961': dr.d.class0(0x4961),
              # DS 段的兩個旗標（不是 ECL 變數）：`4960h` 是主迴圈的離開旗標，
              # `4961h` 是 ovr5 `15CFh` 寫、ovr37 `598h` 讀的那一個。
              'ds_4960': dr.d.mem(D.DS, 0x4960, 1)[0], 'ds_4961': dr.d.mem(D.DS, 0x4961, 1)[0],
              'combat_result_6DC7': dr.d.class1(0x6DC7),
              'party': party_states(dr.d),
              'sha256': hashlib.sha256(idx).hexdigest()}
    print('==', name, record['where'], '4961=%d' % record['gameover_4961'],
          '6DC7=%02X' % record['combat_result_6DC7'], '底列', repr(record['bottom'])[:60])
    return record


def party_states(d):
    """沿隊伍鏈讀每個人的狀態位元組（`+11Bh` 是生命值、`+11Dh` 是狀態，spec 018／137）。"""
    seg, off = d.far(0x5CF4)
    out = []
    for _ in range(8):
        if seg == 0 and off == 0:
            break
        record = d.mem(seg, off, 0x120)
        # `+11Bh` 目前 HP、`+32h` 最大 HP（spec 004，兩個都是 byte）、`+10Ch` 狀態 byte
        # （spec 018；名稱表還沒閉合，原樣保存）。`+84h` 是 active。
        out.append({'name': record[:15].split(b'\x00')[0].decode('ascii', 'replace').strip(),
                    'hp_11B': record[0x11B], 'max_hp_32': record[0x32], 'status_10C': record[0x10C],
                    'active_84': record[0x84]})
        nxt_off = record[0x104] | record[0x105] << 8
        nxt_seg = record[0x106] | record[0x107] << 8
        if (nxt_seg, nxt_off) in ((0, 0), (0xFFFF, 0xFFFF)):
            break
        seg, off = nxt_seg, nxt_off
    return out


class ProgramExited(Exception):
    """原版呼叫了 int 21h AH=4Ch；reply 是結束那一刻的 serve 回覆。"""

    def __init__(self, key, reply):
        super().__init__('原版在 %s 之後結束（離開碼 %s）' % (key, reply.get('exit_code')))
        self.key, self.reply = key, reply


def watch_exit(dos):
    """包住 key／wait：回覆帶 exited 就丟 ProgramExited，駕駛的 settle 才不會繼續按。"""
    key, wait = dos.key, dos.wait
    dos.key_replies = []

    def checked_key(name):
        reply = key(name)
        dos.key_replies.append({'key': name, 'consumed': reply.get('consumed'), 'steps': reply.get('steps'),
                                'exited': bool(reply.get('exited'))})
        del dos.key_replies[:-8]
        if reply.get('exited'):
            raise ProgramExited(name, reply)
        return reply

    def checked_wait():
        reply = wait()
        if reply.get('exited'):
            raise ProgramExited('wait', reply)
        return reply
    dos.key, dos.wait = checked_key, checked_wait


def watch_anykey(dr, steps):
    """全滅頁第一次出現時（按鍵之前）拍一張、存狀態，之後重現不必重跑開機。"""
    press = dr.press
    seen = []

    def checked_press(key):
        if not seen and 'PRESS ANY KEY' in dr.screen().bottom():
            seen.append(key)
            steps.append(snapshot(dr, '%02d-anykey' % len(steps)))
            dr.d.save('/work/wipe-anykey.state')
        return press(key)
    dr.press = checked_press


def record_exit(dr, steps, index, exited):
    steps.append(snapshot(dr, '%02d-exited' % index))
    dr.d.save('/work/wipe.state')
    return {'after_key': exited.key, 'exit_code': exited.reply.get('exit_code'), 'steps': exited.reply.get('steps'),
            'cs_ip': exited.reply.get('cs_ip'), 'last_key_replies': list(dr.d.key_replies)}


def main():
    os.makedirs(OUT, exist_ok=True)
    # **不要攔截扣血**：那一支是作弊通關用的，開著就打不死（本檔的重點就是要被打死）。
    original = D.Dos
    D.Dos = lambda *a, **k: original(*a, **dict(k, intercept=False))
    dr = cheat.Driver('wipe', load=None, keys=BOOT)
    D.Dos = original
    watch_exit(dr.d)
    dr.settle()
    steps = [snapshot(dr, '00-tour-end')]
    watch_anykey(dr, steps)
    # 往西一步進貧民窟，然後一直往前走，撞到什麼打什麼，直到打光為止。
    dr.press('Up')
    dr.settle()
    steps.append(snapshot(dr, '01-slums'))
    exit_record = None
    for index in range(400):
        try:
            dr.press('Up')
            dr.settle()
        except ProgramExited as exited:
            exit_record = record_exit(dr, steps, index + 2, exited)
            print(exited)
            break
        except cheat.Stuck as stuck:
            # 駕駛不認得的畫面就是這一支要找的東西（全滅之後那一頁）。**停在這裡不要只拍一張**：
            # 按下去才看得到原版把玩家帶到哪。
            steps.append(snapshot(dr, '%02d-stuck' % (index + 2)))
            # 存下這一刻：後面要反覆試按鍵，重開機走到這裡要好幾分鐘。
            dr.d.save('/work/wipe.state')
            print('駕駛不認得這個畫面：', stuck)
            # 「PRESS ANY KEY TO CONTINUE」推不推得動要逐一試，每次都先讓模擬器把畫面跑完。
            try:
                before = dr.d.steps
                reply = dr.d.wait()
                print('等一次：steps %d → %d，回覆 %s' % (before, dr.d.steps, {k: v for k, v in reply.items() if k in ('settled', 'consumed', 'queued', 'cs_ip')}))
                for extra, key in enumerate(['Return', 'Space', 'Escape', 'y', 'Return', 'Space']):
                    dr.d.key(key)
                    dr.d.wait()
                    steps.append(snapshot(dr, '%02d-after-%d-%s' % (index + 2, extra, key)))
            except ProgramExited as exited:
                exit_record = record_exit(dr, steps, index + 2, exited)
                print(exited)
            break
        screen = dr.screen()
        alive = [m for m in party_states(dr.d) if m['hp_11B'] > 0]
        if not alive or 'END' in ' '.join(screen.rows).upper():
            steps.append(snapshot(dr, '%02d-wiped' % (index + 2)))
            # 全滅之後再按幾下，看原版把玩家帶到哪。
            try:
                for extra in range(6):
                    dr.press('Return')
                    steps.append(snapshot(dr, '%02d-after-%d' % (index + 2, extra)))
            except ProgramExited as exited:
                exit_record = record_exit(dr, steps, index + 2, exited)
                print(exited)
            break
    receipt = {'schema': 'pool-dos-party-wipe/1', 'generator': 'dosgolem',
               'dosgolem_commit': os.environ.get('DOSGOLEM_COMMIT') or None, 'boot_keys': BOOT,
               'intercept_damage': False,
               'original_exe_sha256': hashlib.sha256(open('/orig/start.exe', 'rb').read()).hexdigest(),
               'steps': steps, 'program_exit': exit_record}
    json.dump(receipt, open('/work/dos-party-wipe.json', 'w'), ensure_ascii=False, indent=2)
    open('/work/dos-party-wipe.json', 'a').write('\n')
    print('寫出 /work/dos-party-wipe.json')
    dr.d.quit()


main()
