#!/usr/bin/env python3
"""原版雇一名傭兵、再開打，讀 NPC 記錄的戰鬥欄位（#107，spec 154）。

路線全是正常按鍵：建一名角色（HERO）→ 導覽跑完 → 走進訓練所（ecl3/11）的競技場 →
不決鬥（n）→ 找夥伴（y）→ 接受第一個開價（y）→ 不再看別人（n）。ADD NPC（ecl3/11 `9F1Ch`）
在 overlay-03 `2F46h` 跑一次 overlay-25 entry 7，所以加入那一刻讀到的記錄就是重算後的值。
接著在同一格紮營休息：入口 2（`9985h`）把打斷機率設起來，入口 3（`99AAh`）是城市衛兵
趕人（FIGHT／GO），按 f 開打，讀開打那一幀的 NPC 記錄與 runtime（`+108h`）。

dosgolem 攔截扣血入口（與作弊通關同一個 hook：隊員不扣血、怪物一擊斃命），只為了讓
衛兵那一場不會把一人隊伍打死；讀的是開打那一幀，還沒有人出手。

用法（主機端，Docker 由本檔起）：
  python3 tools/dosgolem-npc-combat-runtime.py
可用環境變數改來源：DOSGOLEM_SHOTS（shots 執行檔）、DOS_ORIG（原版目錄）、DOS_FONT（字形表）。
產物：workplace/npc-combat/*.png、npc-combat-runtime.json，抄到 docs/audit/dosgolem-npc-combat-runtime.json。"""
import hashlib, json, os, shutil, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WORK = os.path.join(ROOT, 'workplace', 'npc-combat')
if not os.path.exists('/bin/shots'):
    shots = os.environ.get('DOSGOLEM_SHOTS', os.path.join(ROOT, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots'))
    orig = os.environ.get('DOS_ORIG', os.path.join(ROOT, 'workplace', 'oracle', 'dos'))
    font = os.environ.get('DOS_FONT', os.path.join(ROOT, 'workplace', 'dosgolem-cheat', 'font.json'))
    for path in (shots, os.path.join(orig, 'start.exe'), font):
        if not os.path.isfile(path):
            raise SystemExit('缺 %s' % path)
    shutil.rmtree(os.path.join(WORK, 'scratch'), ignore_errors=True)
    os.makedirs(os.path.join(WORK, 'scratch'))
    shutil.copy(font, os.path.join(WORK, 'font.json'))
    code = subprocess.call(
        ['timeout', '3600', 'docker', 'run', '--rm', '-i', '--network', 'none', '--memory', '4g', '--cpus', '2',
         '--pids-limit', '256', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
         '-u', '%d:%d' % (os.getuid(), os.getgid()),
         '-v', shots + ':/bin/shots:ro', '-v', orig + ':/orig:ro',
         '-v', os.path.join(ROOT, 'tools') + ':/tools:ro',
         '-v', WORK + ':/work', '-v', os.path.join(WORK, 'scratch') + ':/scratch',
         '-e', 'PYTHONUNBUFFERED=1', '-e', 'PYTHONPATH=/tools', '-w', '/work',
         'python:3.13-alpine', 'python3', '/tools/dosgolem-npc-combat-runtime.py'])
    if code != 0:
        sys.exit(code)
    receipt = json.load(open(os.path.join(WORK, 'npc-combat-runtime.json')))
    receipt['shots_sha256'] = hashlib.sha256(open(shots, 'rb').read()).hexdigest()
    # shots 放在 dosgolem checkout 的 workplace/bin/ 底下；記下那個 checkout 的分支與 commit
    # （執行檔的 SHA-256 才是這一次真的跑的那一支）。
    git = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(shots))), '.git')
    if os.path.isfile(os.path.join(git, 'HEAD')):
        head = open(os.path.join(git, 'HEAD')).read().strip()
        receipt['dosgolem_checkout_head'] = head
        if head.startswith('ref: '):
            ref = head[5:]
            path = os.path.join(git, ref)
            if os.path.isfile(path):
                receipt['dosgolem_checkout_commit'] = open(path).read().strip()
            elif os.path.isfile(os.path.join(git, 'packed-refs')):
                for line in open(os.path.join(git, 'packed-refs')):
                    if line.strip().endswith(' ' + ref):
                        receipt['dosgolem_checkout_commit'] = line.split()[0]
    out = os.path.join(ROOT, 'docs', 'audit', 'dosgolem-npc-combat-runtime.json')
    with open(out, 'w') as f:
        json.dump(receipt, f, ensure_ascii=False, indent=1)
        f.write('\n')
    print('寫出', out)
    sys.exit(0)

sys.path.insert(0, '/tools')
import dos_text
import dosgolem_drive as D

OUT = '/work'
# 建角與導覽：與 tools/dosgolem-tour-cell-rest.py 同一段。
BOOT = 'rep:9:Space,Return,Return,c,Return,Return,Return,Return,Return,Return,y,H,E,R,O,Return,k,e,y,a,a,e,b,rep:14:Return'
# 導覽終點 (0,4) → 訓練所（ecl3/11）→ 競技場 (7,2)：與 docs/audit/dos-thief-training-skills.json 同一段路。
ARENA = 'Right,Right,Up,Up,Left,Up,Right,Up,Up,Up,Left,Up,Right,Up,Up,Return'.split(',')
# 紮營休息兩小時（與 tools/dosgolem-tour-cell-rest.py 同一組鍵）。
REST = ['e', 'r', 'h', 'i', 'i', 'r']
RECORD = 0x11D
frames = []


def shot(d, name):
    idx = d.frame()
    dos_text.write_png(os.path.join(OUT, '%02d-%s.png' % (len(frames) + 1, name)), idx)
    screen = D.Screen(idx, d.font)
    rows = [r for r in screen.rows if r.strip()]
    where = d.where()
    print('==', name, where)
    print('\n'.join('   | ' + r for r in rows))
    frames.append({'name': name, 'where': where, 'text': screen.flat(), 'bottom': screen.bottom(),
                   'sha256': hashlib.sha256(idx).hexdigest()})
    return screen


def press(d, key, name=None):
    d.key(key)
    d.wait()
    return shot(d, name or key)


def chain(d):
    """沿 DS:5CF4h 的串列（`+104h` 接下一筆）讀每一筆記錄；戰鬥中同一條串列後段是怪物。"""
    seg, off = d.far(0x5CF4)
    records = []
    while (seg or off) and len(records) < 40:
        raw = d.mem(seg, off, RECORD)
        runtime = None
        rseg, roff = raw[0x10A] | raw[0x10B] << 8, raw[0x108] | raw[0x109] << 8
        if rseg or roff:
            runtime = d.mem(rseg, roff, 0x20)
        records.append({'address': '%04X:%04X' % (seg, off), 'raw': raw, 'runtime': runtime})
        seg, off = raw[0x106] | raw[0x107] << 8, raw[0x104] | raw[0x105] << 8
    return records


def fields(entry):
    b = entry['raw']
    out = {
        'address': entry['address'],
        'name': b[1:1 + b[0]].decode('latin1'),
        'race_2Eh': b[0x2E], 'class_levels_96h': b[0x96:0x9E].hex(), 'hit_dice_73h': b[0x73],
        'base_thac0_2Dh': b[0x2D], 'base_ac_A9h': b[0xA9], 'base_move_72h': b[0x72],
        'ability_flag_AAh': b[0xAA], 'dice_A2h_A8h': b[0xA2:0xA9].hex(),
        'sweep_limit_6Bh': b[0x6B], 'morale_84h': b[0x84], 'random_ABh': b[0xAB],
        'icon_BBh': b[0xBB], 'icon_BCh': b[0xBC], 'side_10Eh': b[0x10E],
        'thac0_110h': b[0x110], 'ac_111h': b[0x111], 'rear_ac_112h': b[0x112],
        'runtime_dice_114h_11Ah': b[0x114:0x11B].hex(), 'move_11Ch': b[0x11C],
        'carried_102h': b[0x102] | b[0x103] << 8, 'current_hp_11Bh': b[0x11B],
        'weapon_0CCh': b[0xCC:0xD0].hex(), 'items_C8h': b[0xC8:0xCC].hex(), 'effects_7Fh': b[0x7F:0x83].hex(),
        'record_hex': b.hex(),
    }
    if entry['runtime'] is not None:
        out['runtime_108h_hex'] = entry['runtime'].hex()
        out['runtime_sweep_limit_5'] = entry['runtime'][5]
    return out


def settle(d, rounds=12):
    last = None
    for _ in range(rounds):
        d.wait()
        digest = hashlib.sha256(d.frame()).hexdigest()
        if digest == last:
            return
        last = digest


def main():
    d = D.Dos(work=OUT, keys=BOOT)
    d.wait()
    for _ in range(120):
        if 'ENCAMP' in D.Screen(d.frame(), d.font).bottom().upper():
            break
        d.key('Return')
        d.wait()
    else:
        raise SystemExit('導覽沒有結束')
    shot(d, 'tour-end')
    for key in ARENA:
        screen = press(d, key)
    if 'DUEL' not in screen.flat():
        raise SystemExit('沒有走到競技場')
    screen = press(d, 'n', 'no-duel')
    if 'PARTNER' not in screen.flat():
        raise SystemExit('沒有問找不找夥伴')
    screen = press(d, 'y', 'seek-partner')
    offer = screen.flat()
    if 'JOIN YOU' not in offer:
        raise SystemExit('沒有開價：%s' % offer)
    before = chain(d)
    screen = press(d, 'y', 'accept')
    joined = chain(d)
    if len(joined) != len(before) + 1:
        raise SystemExit('隊伍沒有多一人：%d → %d' % (len(before), len(joined)))
    npc = fields(joined[-1])
    print('NPC', npc['name'], 'THAC0', npc['thac0_110h'], 'AC', npc['ac_111h'], '+6Bh', npc['sweep_limit_6Bh'])
    screen = press(d, 'n', 'no-more')
    d.save('/work/hired.state')
    for key in REST:
        screen = press(d, key, 'rest-' + key)
    settle(d)
    screen = shot(d, 'after-rest')
    fight = None
    for _ in range(6):
        if 'FIGHT' in screen.bottom():
            fight = screen.flat()
            break
        # 休息被打斷之後可能先有一頁 PRESS RETURN 或留下／離開的選單；只接受衛兵那一句。
        screen = press(d, 'Return', 'after-rest-return')
    if fight is None:
        raise SystemExit('休息沒有被衛兵打斷：%s' % screen.flat())
    screen = press(d, 'f', 'fight')
    for _ in range(40):
        if d.where()['combat'] and d.mem(D.DS, 0x5CF4, 4) != b'\0\0\0\0':
            break
        d.key('Return')
        d.wait()
    else:
        raise SystemExit('沒有進戰鬥')
    settle(d)
    shot(d, 'combat-start')
    combat = [fields(entry) for entry in chain(d)]
    receipt = {
        'schema': 'pool-dosgolem-npc-combat/1', 'generator': 'dosgolem',
        'original_exe_sha256': hashlib.sha256(open('/orig/start.exe', 'rb').read()).hexdigest(),
        'boot_keys': BOOT, 'arena_keys': ARENA, 'hire_keys': ['n', 'y', 'y', 'n'], 'rest_keys': REST,
        'fight_key': 'f', 'offer': offer, 'watch': fight,
        'how': '記錄沿 DS:5CF4h（+104h 接下一筆）各讀 11Dh bytes；runtime 是 +108h 指到的 20h bytes',
        'joined': npc, 'combat': combat, 'frames': frames,
    }
    json.dump(receipt, open('/work/npc-combat-runtime.json', 'w'), ensure_ascii=False, indent=1)
    d.quit()


main()
