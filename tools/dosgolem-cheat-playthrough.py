#!/usr/bin/env python3
"""原版那一側的作弊通關（#5）：dosgolem 攔截扣血入口（隊員傷害 0、怪物一擊斃命），
以狀態檔分段，照 spec 137 路線 (a) 從標題推到結局。

用法（主機端）：
  python3 tools/dosgolem-cheat-playthrough.py <段名> [--from <狀態檔>]
段名與順序見 SEGMENTS。每段結束存 workplace/dosgolem-cheat/<段名>.state，
收據累積在 workplace/dosgolem-cheat/receipt.json，整理後寫 docs/audit/dosgolem-cheat-playthrough.json。

畫面判斷：底列文字（tools/dos_text.py 的字形表）與熱鍵顏色；位置、區塊、旗標一律讀記憶體。"""
import collections, hashlib, json, os, random, subprocess, sys, time

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if not os.path.exists('/bin/shots'):
    # 主機端：自己起容器再跑一次。
    work = os.path.join(ROOT, 'workplace', 'dosgolem-cheat')
    os.makedirs(os.path.join(work, 'scratch'), exist_ok=True)
    args = ['timeout', '7200', 'docker', 'run', '--rm', '-i', '--network', 'none', '--memory', '4g', '--cpus', '2',
            '--pids-limit', '256', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
            '-u', '%d:%d' % (os.getuid(), os.getgid()),
            '-v', os.path.join(ROOT, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots') + ':/bin/shots:ro',
            '-v', os.path.join(ROOT, 'workplace', 'oracle', 'dos') + ':/orig:ro',
            '-v', os.path.join(ROOT, 'tools') + ':/tools:ro',
            '-v', os.path.join(ROOT, 'docs', 'audit') + ':/audit:ro',
            '-v', work + ':/work', '-v', os.path.join(work, 'scratch') + ':/scratch',
            '-e', 'PYTHONUNBUFFERED=1', '-e', 'PYTHONPATH=/tools', '-w', '/work',
            '-e', 'DOS_TRACE=' + os.environ.get('DOS_TRACE', ''),
            'python:3.13-alpine', 'python3', '/tools/dosgolem-cheat-playthrough.py'] + sys.argv[1:]
    sys.exit(subprocess.call(args))

sys.path.insert(0, '/tools')
import dosgolem_drive as D
import dos_text

WORK = '/work'
GEO = json.load(open(os.path.join(WORK, 'geo.json')))
FLAGS = [0x4ABB, 0x4AC1, 0x4AB0, 0x4AA6, 0x4AA7, 0x4A24, 0x4AC4, 0x4A77, 0x4A78, 0x4ABA]
DELTA = [(0, -1), (1, 0), (0, 1), (-1, 0)]
DIRS = 'NESW'


class Stuck(Exception):
    pass


class Driver:
    def __init__(self, segment, load, keys=''):
        self.segment = segment
        self.log = open(os.path.join(WORK, segment + '.log'), 'a')
        self.d = D.Dos(load=load, keys=keys, log=self.log)
        self.boot_keys = keys
        self.checkpoints = []
        self.last_block = None
        self.actions = 0
        self.unknown = 0
        self.idle_waits = 0
        self.blocked = set()
        self.bashes = 0
        self.combat_stills = 0
        self.wall_clears = []
        self.ending_pages = []
        self.hedge_refusals = 0
        self.watch_ending = segment == 'ending'
        self.say('== %s from %s' % (segment, load))
        self.observe()

    def say(self, text):
        line = time.strftime('%H:%M:%S ') + text
        print(line)
        self.log.write(line + '\n')
        self.log.flush()

    # ---------- 記憶體 ----------
    def where(self):
        return self.d.where()

    def flags(self):
        return {'%04X' % a: '%02X' % v for a, v in self.d.class0_many(FLAGS).items()}

    def observe(self):
        w = self.where()
        block = (w['archive'], w['block'])
        if block != self.last_block:
            self.last_block = block
            cp = {'sequence': len(self.checkpoints), 'ecl_archive': w['archive'], 'ecl_block': w['block'],
                  'x': w['x'], 'y': w['y'], 'hour': self.d.class0(0x49C9), 'flags': self.flags(),
                  'step': self.d.steps}
            self.checkpoints.append(cp)
            self.say('checkpoint ECL%d/%d (%d,%d) %s' % (w['archive'], w['block'], w['x'], w['y'], cp['flags']))
        return w

    # ---------- 畫面 ----------
    KNOWN_LINES = ['PRESS <ENTER>/<RETURN> TO CONTINUE']

    def screen(self):
        s = D.Screen(self.d.frame(), self.d.font)
        if '?' in s.bottom():
            for text in self.KNOWN_LINES:
                if 0 in dos_text.align(s.idx, 24, text):
                    dos_text.learn(s.idx, 24, 0, text, self.d.font)
                    dos_text.save_font(os.path.join(WORK, 'font.json'), self.d.font)
                    s = D.Screen(s.idx, self.d.font)
        return s

    def learn_position(self, screen, w):
        text = '%d,%d %s' % (w['x'], w['y'], DIRS[w['facing']])
        hits = dos_text.align(screen.idx, 15, text, 17)
        if 17 in hits:
            before = len(self.d.font)
            dos_text.learn(screen.idx, 15, 17, text, self.d.font)
            if len(self.d.font) != before:
                dos_text.save_font(os.path.join(WORK, 'font.json'), self.d.font)

    def dump(self, screen, why):
        self.unknown += 1
        name = '%s-unknown-%02d.idx' % (self.segment, self.unknown)
        open(os.path.join(WORK, name), 'wb').write(screen.idx)
        dos_text.write_png(os.path.join(WORK, name[:-4] + '.png'), screen.idx)
        self.say('UNKNOWN (%s) → %s\n%s\n%s' % (why, name, screen.text(16, 24), screen.options()))

    def press(self, key):
        if key == 'Return' and self.watch_ending:
            self.note_ending_page(D.Screen(self.d.frame(), self.d.font))
        self.actions += 1
        r = self.d.key(key)
        self.observe()
        return r

    def pick(self, screen, words, row=24):
        """底列裡第一個符合的選項：按它的熱鍵。回傳選了哪一個。"""
        options = screen.options(row)
        for want in words:
            for word, hot, _ in options:
                if word.startswith(want):
                    # 游標所在的那一項整個詞同色，認不出熱鍵；原版底列的熱鍵都是首字母。
                    self.press((hot or word[0]).lower())
                    return word
        return None

    def pick_line(self, screen, want):
        """文字框裡的直式選單（每行一個選項）：找到含 want 的那一行，游標移過去按 Return。
        游標那一行的字是白色（15），題目與其他選項是綠色。"""
        rows = [r for r in range(16, 24) if want in screen.rows[r]]
        if not rows:
            return False
        target = rows[0]
        current = [r for r in range(16, 24) if screen.rows[r].strip() and
                   any(D.fg_color(screen.idx, c, r) == 15 for c in range(1, 39) if screen.rows[r][c:c + 1].strip())]
        if current and current[0] != target:
            # 原版直式選單：End 往下、Home 往上（方向鍵讀走了但不動游標，實測）。
            self.press('End' if current[0] < target else 'Home')
            return True
        self.press('Return')
        return True

    def mode(self, screen):
        b = screen.bottom()
        if b.startswith('AREA CAST VIEW ENCAMP') or b.startswith('CAST VIEW ENCAMP SEARCH'):   # 後者是野外（沒有 AREA）
            # 直式選單開著時底列可能還是移動列（城堡外圈的密語選單）：文字框裡有白色游標那一行就不算自由移動。
            # 判準是「白色那一行之後還有綠色行」：訊息（「D DIES.」）也可能是白色，但下面不會再接選項。
            colors = []
            for r in range(17, 23):
                if screen.rows[r][1:39].strip():
                    cs = {D.fg_color(screen.idx, c, r) for c in range(1, 39) if screen.rows[r][c:c + 1].strip()}
                    colors.append(15 if 15 in cs else 10 if 10 in cs else None)
            if 15 in colors and 10 in colors[colors.index(15) + 1:]:
                return 'other'
            return 'move'
        if b.startswith('MOVE VIEW AIM QUICK') or 'QUICK' in b and 'DONE' in b:
            return 'combat'
        return 'other'

    # ---------- 通用：把畫面推回自由移動 ----------
    def settle(self, policy=None, limit=200):
        """處理畫面上的文字、選單、戰鬥，直到回到自由移動。policy(screen) 可以先處理一次，回 True 表示處理了。"""
        last, same = None, 0
        for _ in range(limit):
            s = self.screen()
            m = self.mode(s)
            if m == 'move':
                return s
            # 比文字不比位元組：游標會閃，位元組每一幀都不同。
            same = same + 1 if s.rows == last else 0
            last = s.rows
            if same >= 3 and m != 'combat':
                # 同一個畫面按了三次都沒變：選單前面還有沒翻完的文字（古托井「CLIMB DOWN?」要先 Return 兩次，實測）。
                self.press('Return')
                same = 0
                continue
            if policy and policy(self, s):
                continue
            if m == 'combat':
                self.press('q')
                continue
            if self.d.mem(D.DS, 0x4954, 1)[0] == 5 and not s.bottom().startswith('CONTINUE BATTLE') \
                    and not [w for w, h, _ in s.options() if h]:
                # 戰鬥中底列在印動作訊息（GUARDING、IS DOWN…）：讓程式跑，畫面連兩次沒變才按 Return。
                self.d.wait()
                again = self.screen()
                if again.idx == s.idx:
                    self.combat_stills += 1
                    if self.combat_stills >= 2:
                        self.combat_stills = 0
                        self.press('Return')
                continue
            if self.generic(s):
                continue
            self.dump(s, 'settle')
            raise Stuck('unknown screen')
        raise Stuck('settle limit')

    def note_ending_page(self, s):
        """打贏泰倫斯拉克斯（`4ABA = FE`）之後出現過的每一頁文字，當結局頁收進收據。"""
        if self.d.class0(0x4ABA) != 0xFE:
            return
        # 整個畫面讀字：`PROGRAM 8` 的結局過場不一定用文字框的版面。
        page = ' | '.join(' '.join(r.replace('?', ' ').split()) for r in s.rows if r.replace('?', '').strip())
        if page and page not in self.ending_pages:
            self.ending_pages.append(page)
            dos_text.write_png(os.path.join(WORK, 'ending-page-%02d.png' % len(self.ending_pages)), s.idx)

    def generic(self, s):
        text = s.flat() + ' ' + s.bottom()   # 底列從第 0 欄起，不能套 flat 的去邊框
        opts = [w for w, _, _ in s.options()]
        if 'PRESS' in text and ('RETURN' in text or 'ENTER' in text or 'ANY KEY' in text):
            self.note_ending_page(s)
            self.press('Return')
            return True
        if s.bottom().startswith('A BATTLE BEGINS') or not s.bottom().strip():
            # 過場：底列是空的（文字剛印出、延遲、動畫），讓程式跑；畫面連兩次沒變才按 Return。
            self.note_ending_page(s)
            self.d.wait()
            self.idle_waits += 1
            if self.screen().idx == s.idx and self.idle_waits % 2 == 0:
                self.press('Return')
            return self.idle_waits < 40
        self.idle_waits = 0
        if 'BASH' in opts:
            # 鎖住的門：撞到開（至多 12 次），撞不開就離開，讓 step 記成擋路。
            self.bashes += 1
            if self.bashes <= 12:
                return bool(self.pick(s, ['BASH']))
            return bool(self.pick(s, ['EXIT', 'LEAVE']))
        if s.bottom().startswith('CONTINUE BATTLE'):   # 敵方都倒了或跑了：收場
            self.press('n')
            return True
        # 城衛隊的問句（spec 102；remake `cityWatchAnswer`）：第 0 項多半是開打的那一個。
        if 'DO YOU WANT TO BREAK IN' in text:
            self.press('n')
            return True
        if 'ROUSTED BY THE CITY WATCH' in text:
            return bool(self.pick(s, ['GO']))
        if 'CITY WATCH' in text and 'RUN' in opts:
            return bool(self.pick(s, ['RUN']))
        if 'FORCE' in opts and 'LEAVE' in opts:
            return bool(self.pick(s, ['LEAVE']))
        if 'STILL TREASURE LEFT' in text:
            self.press('n')
            return True
        if opts[:1] == ['COMBAT'] or 'COMBAT' in opts and 'FLEE' in opts:
            return bool(self.pick(s, ['COMBAT']))
        if 'SHARE' in opts and 'POOL' in opts:
            # 戰利品裡有錢：SHARE 平分給隊員（remake 探針同樣選 Share）。委任獎金的白金是東航線船票的錢。
            return bool(self.pick(s, ['SHARE']))
        if 'POOL' in opts and 'EXIT' in opts:     # 戰利品（有物品時多一個 TAKE）：物品不拿，離開
            return bool(self.pick(s, ['EXIT']))
        return False

    # ---------- 走路 ----------
    def passable(self, block, x, y, d):
        # 門值 1／2／3 都先當走得過：鎖不鎖由 ECL 腳本決定（貧民窟 (7,3)→(6,3) 與 (3,4)→(3,3)
        # 的牆型一模一樣，前者直接走過、後者跳 LOCKED 選單）。走不過去的邊由 step 記進 blocked。
        v = GEO[str(block)][y][x]['p'][d]
        return v != 9 and (block, x, y, d) not in self.blocked

    def plan(self, w, wanted, avoid=None):
        # 順序：照牆並避開事件格 → 照牆不避開 → 穿牆並避開（使用者 2026-09-17：兩邊都可以暫時穿牆，
        # 逐段對照得起來就好）。穿牆放最後，免得有正常路還去撞牆。
        path = self._plan(w, wanted, avoid, False)
        if path is None and avoid is not None:
            path = self._plan(w, wanted, None, False)
        if path is None:
            path = self._plan(w, wanted, avoid, True)
        return path

    def _plan(self, w, wanted, avoid, through):
        start = (w['x'], w['y'])
        prev = {start: None}
        q = collections.deque([start])
        while q:
            cur = q.popleft()
            if cur != start and wanted(*cur):
                path = []
                while prev[cur] is not None:
                    cur, d = prev[cur]
                    path.append(d)
                return path[::-1]
            for d in range(4):
                if not through and not self.passable(w['geo'], cur[0], cur[1], d):
                    continue
                if (w['geo'], cur[0], cur[1], d) in self.blocked:
                    # 穿了牆也過不去的邊（事件擋住）：穿牆模式同樣不走，否則會一直選同一步。
                    continue
                nx, ny = cur[0] + DELTA[d][0], cur[1] + DELTA[d][1]
                if not (0 <= nx < 16 and 0 <= ny < 16) or (nx, ny) in prev:
                    continue
                if avoid and avoid(nx, ny) and not wanted(nx, ny):
                    continue
                prev[(nx, ny)] = (cur, d)
                q.append((nx, ny))
        return None

    def face(self, want, policy=None):
        for _ in range(8):
            self.settle(policy)
            w = self.where()
            if w['facing'] == want:
                return w
            turn = (want - w['facing']) % 4
            self.press('Left' if turn == 3 else 'Right')
        raise Stuck('cannot face %d' % want)

    def step(self, d, policy=None):
        """朝 d 走一步，處理中途的一切，回傳走完之後的位置。"""
        before = self.face(d, policy)
        self.bashes = 0
        self.press('Up')
        self.settle(policy)
        after = self.where()
        if self.same_cell(before, after):
            after = self.step_through_wall(before, d, policy)
        return after

    @staticmethod
    def same_cell(a, b):
        return (a['archive'], a['block'], a['x'], a['y']) == (b['archive'], b['block'], b['x'], b['y'])

    def step_through_wall(self, before, d, policy):
        """被擋住的那一步：把 `DS:69BAh` 指到的 GEO 牆 nibble（spec 078：+0 北高／東低、+100h 南高／西低）
        在這一格朝 d 與鄰格朝回來的那一面清成 0，踏過去，再寫回原值。每一次記進 wall_clears。"""
        seg, off = self.d.far(0x69BA)
        x, y = before['x'], before['y']
        nx, ny = (x + DELTA[d][0]) % 16, (y + DELTA[d][1]) % 16
        # (平面, 遮罩)：北 (0, 高)、東 (0, 低)、南 (1, 高)、西 (1, 低)
        side = [(0, 0xF0), (0, 0x0F), (1, 0xF0), (1, 0x0F)]
        targets = [(side[d][0] * 0x100 + y * 16 + x, side[d][1]),
                   (side[(d + 2) % 4][0] * 0x100 + ny * 16 + nx, side[(d + 2) % 4][1])]
        saved = []
        for index, mask in targets:
            value = self.d.mem(seg, (off + index) & 0xFFFF, 1)[0]
            saved.append((index, value))
            self.d.poke(seg, (off + index) & 0xFFFF, [value & ~mask & 0xFF])
        self.wall_clears.append({'geo': before['geo'], 'x': x, 'y': y, 'dir': DIRS[d],
                                 'bytes': ['+%03X %02X' % (i, v) for i, v in saved]})
        self.say('through wall: GEO%d (%d,%d) %s %s' % (before['geo'], x, y, DIRS[d], self.wall_clears[-1]['bytes']))
        self.face(d, policy)
        self.press('Up')
        self.settle(policy)
        after = self.where()
        if after['geo'] == before['geo'] and self.d.far(0x69BA) == (seg, off):
            for index, value in saved:
                self.d.poke(seg, (off + index) & 0xFFFF, [value])
        if self.same_cell(before, after):
            self.blocked.add((before['geo'], x, y, d))
            self.say('blocked even without the wall: GEO%d (%d,%d) %s' % (before['geo'], x, y, DIRS[d]))
        return after


# ---------- 段落 ----------
def walk_to(dr, wanted, policy=None, avoid=None, limit=400):
    still = 0
    for _ in range(limit):
        w = dr.where()
        if wanted(w['x'], w['y']) and not getattr(wanted, 'strict', False):
            return w
        path = dr.plan(w, wanted, avoid)
        if not path:
            raise Stuck('no path from %s' % w)
        after = dr.step(path[0], policy)
        still = still + 1 if dr.same_cell(w, after) else 0
        if still >= 12:
            raise Stuck('walk: position unchanged for 12 steps at %s' % after)
    raise Stuck('walk limit')


def exit_edge(dr, side, policy=None):
    """走到地圖邊緣朝外踏出（side：0 北 1 東 2 南 3 西），直到區塊換掉。"""
    start = dr.where()
    here = (start['archive'], start['block'])
    def edge(x, y):
        return (side == 0 and y == 0 or side == 1 and x == 15 or side == 2 and y == 15 or side == 3 and x == 0) \
            and dr.passable(start['geo'], x, y, side)
    for _ in range(400):
        w = dr.where()
        if (w['archive'], w['block']) != here:
            return w
        if edge(w['x'], w['y']):
            dr.step(side, policy)
            continue
        avoid = None
        if (w['archive'], w['block']) == (2, 20):
            avoid = lambda x, y: terrain_code(w, x, y) in SLUMS_AVOID
        path = dr.plan(w, edge, avoid)
        if not path:
            raise Stuck('no path to edge %d from %s' % (side, w))
        dr.step(path[0], policy)
    raise Stuck('edge limit')


# 貧民窟探索時不踩的地形（`ecl2/20` 入口 1 依地形碼分派）：3 潛在委託人（答錯會動 `4A81`，攤位那一場
# 就不計數）、8 算命的老婦人（remake 路線 (a) 不殺她）、19 攤位（留到最後打，spec 137）。
SLUMS_AVOID = {3, 8, 19}


def slums_policy(dr, s):
    text = s.flat()
    if 'ELEGANTLY PANELLED' in text:
        return bool(dr.pick(s, ['LEAVE']))
    return bool(dr.pick(s, ['ATTACK', 'COMBAT', 'FIGHT']))


def terrain_code(w, x, y):
    return GEO[str(w['geo'])][y][x]['t'] & 0x7F


def booth_policy(dr, s):
    return bool(dr.pick(s, ['ATTACK']))


def seg_slums(dr):
    visited = set()
    rounds = stalled = 0
    last_count = -1
    while True:
        count = dr.d.class0(0x4ABB)
        if count >= 0xFE:
            return
        if count >= 0x18:
            # 攤位（地形 19，`ecl2/20 AE1Eh`）答 ATTACK：那一場之後 4ABB 鎖成 FE（spec 137 路線 (a)）。
            w = dr.where()
            if (w['archive'], w['block']) != (2, 20):
                exit_edge(dr, 3, slums_policy)
                continue
            dr.say('slums: 4ABB=%02X, going to the booth' % count)
            walk_to(dr, lambda x, y: terrain_code(w, x, y) == 19, booth_policy)
            dr.settle(booth_policy)
            if dr.d.class0(0x4ABB) != 0xFE:
                raise Stuck('booth did not close the slums: 4ABB=%02X' % dr.d.class0(0x4ABB))
            return
        w = dr.where()
        if (w['archive'], w['block']) == (3, 0):
            exit_edge(dr, 3, slums_policy)
            visited = set()
            continue
        if (w['archive'], w['block']) != (2, 20):
            raise Stuck('left the slums: %s' % w)
        visited.add((w['x'], w['y']))
        avoid = lambda x, y: terrain_code(w, x, y) in SLUMS_AVOID
        path = dr.plan(w, lambda x, y: (x, y) not in visited and not avoid(x, y), avoid)
        if not path:
            rounds += 1
            dr.say('slums round %d done: 4ABB=%02X 4A80=%02X' % (rounds, count, dr.d.class0(0x4A80)))
            stalled = stalled + 1 if count == last_count else 0
            last_count = count
            if stalled >= 3:
                raise Stuck('slums count stalled at %02X' % count)
            exit_edge(dr, 1, slums_policy)
            continue
        before = dr.d.class0(0x4ABB)
        dr.step(path[0], slums_policy)
        after = dr.d.class0(0x4ABB)
        if after != before:
            w2 = dr.where()
            dr.say('4ABB %02X→%02X at (%d,%d) terrain %d' % (before, after, w2['x'], w2['y'], w2['terrain']))


# 貧民窟屋內睡得成的地形（`ecl2/20 9A24h`：`@6E82 != 0` → 打斷 0／0；remake `TestSlumsQuietRoomLetsThePartyRest`）。
SLUMS_QUIET = {4, 7, 11, 17}


def sleep_until(dr, hour):
    """睡到 hour 點。城區會被城衛隊趕（spec 102），先走進貧民窟安靜的屋子睡，醒了再走回城區。"""
    w = dr.where()
    if (w['archive'], w['block']) == (3, 0):
        cross(dr, (3, 0), 3)
        w = dr.where()
        walk_to(dr, lambda x, y: terrain_code(w, x, y) in SLUMS_QUIET, slums_policy,
                avoid=lambda x, y: terrain_code(w, x, y) in SLUMS_AVOID)
        camp_until(dr, hour)
        cross(dr, (2, 20), 1, avoid_codes=SLUMS_AVOID)
        return
    camp_until(dr, hour)


def camp_until(dr, hour):
    """紮營睡到 hour 點（E 紮營 → R 休息 → H 選小時 → I 加 → R 開始）。醒來回到自由移動。"""
    now = dr.d.class0(0x49C9)
    hours = (hour - now) % 24 or 24
    dr.say('sleep: %02d → %02d (%d hours)' % (now, hour, hours))
    dr.settle()
    dr.press('e')
    s = dr.screen()
    if not s.bottom().startswith('CAMP:'):
        dr.dump(s, 'encamp')
        raise Stuck('ENCAMP did not open the camp menu')
    dr.press('r')
    dr.press('h')
    for _ in range(hours):
        dr.press('i')
    dr.press('r')
    for _ in range(400):
        s = dr.screen()
        if s.bottom().startswith('CAMP:'):
            dr.press('e')
            break
        if dr.mode(s) == 'move':
            break
        if dr.mode(s) == 'combat' or not dr.generic(s):
            if dr.mode(s) == 'combat':
                dr.press('q')
            else:
                dr.d.wait()
    dr.settle()
    dr.say('slept until %02d' % dr.d.class0(0x49C9))


def city_hall_policy(dr, s):
    return False


def city_hall(dr, want):
    """從貧民窟或城區走進市政廳踩職員 (5,5)，再從 (3,4) 走出來；want(flags) 為真才算交完。"""
    w = dr.where()
    if (w['archive'], w['block']) == (2, 20):
        exit_edge(dr, 1, slums_policy)
    w = dr.where()
    if (w['archive'], w['block']) != (3, 0):
        raise Stuck('city hall: not in the city %s' % w)
    street = lambda x, y: terrain_code(w, x, y) == 0
    for attempt in range(3):
        # 晚上（`49C9 >= 14`）門是鎖的：答 NO（YES 是 38 隻城衛隊，spec 102），睡到早上再進。
        walk_to(dr, lambda x, y: (x, y) == (3, 4), None, avoid=lambda x, y: not street(x, y))
        dr.face(1)
        dr.press('Up')
        s = dr.screen()
        if 'DOOR IS LOCKED' in s.flat():
            dr.press('n')
            dr.settle()
            sleep_until(dr, 6)
            continue
        dr.settle()
        break
    w = dr.where()
    if (w['archive'], w['block']) != (3, 8):
        raise Stuck('city hall door led to %s' % w)
    walk_to(dr, lambda x, y: (x, y) == (4, 5), clerk_policy)
    dr.step(1, clerk_policy)
    dr.settle(clerk_policy)
    dr.say('clerk: %s 4A01=%02X' % (dr.flags(), dr.d.class0(0x4A01)))
    if not want(dr.flags()):
        raise Stuck('clerk did not take the hand-in: %s' % dr.flags())
    walk_to(dr, lambda x, y: (x, y) == (4, 4), clerk_policy)
    dr.step(3, clerk_policy)
    w = dr.where()
    if (w['archive'], w['block']) != (3, 0):
        raise Stuck('leaving city hall ended at %s' % w)


def clerk_policy(dr, s):
    return False


def seg_handin_slums(dr):
    city_hall(dr, lambda f: f['4ABB'] == 'FF' and int(f['4AC1'], 16) >= 1)


# 古托井（GEO8/29）不跑事件或事件無害的地形（remake `kutoSafe`，`ecl8/29 9A4Bh` 依 `AFCEh` 分派）。
KUTO_SAFE = {0, 2, 3, 4, 5, 9, 13, 14, 16, 17, 18, 19, 20}


def cross(dr, here, side, avoid_codes=None, safe_codes=None, policy=None):
    """在 here=(archive, block) 裡走到 side 邊緣踏出去。"""
    w = dr.where()
    if (w['archive'], w['block']) != here:
        raise Stuck('cross: expected ECL%d/%d, at %s' % (here[0], here[1], w))
    def avoid(x, y):
        code = terrain_code(w, x, y)
        if avoid_codes is not None and code in avoid_codes:
            return True
        return safe_codes is not None and code not in safe_codes
    start = (w['archive'], w['block'])
    on_edge = lambda x, y: (side == 0 and y == 0 or side == 1 and x == 15 or side == 2 and y == 15 or side == 3 and x == 0)
    # 先挑朝外那一面開著的邊緣格（正常出口），沒有才隨便一格邊緣穿出去。
    open_edge = lambda x, y: on_edge(x, y) and dr.passable(w['geo'], x, y, side)
    edge = open_edge if any(open_edge(x, y) for x in range(16) for y in range(16)) else on_edge
    for _ in range(400):
        w = dr.where()
        if (w['archive'], w['block']) != start:
            return w
        if edge(w['x'], w['y']):
            dr.step(side, policy)
            continue
        path = dr.plan(w, edge, avoid) or dr.plan(w, edge)
        if not path:
            raise Stuck('cross: no path to edge %d from %s' % (side, w))
        dr.step(path[0], policy)
    raise Stuck('cross: limit')


def podol_policy(dr, s):
    if 'WHO ARE YOU' in s.flat():
        # 喬裝被盤問（`ecl1/18 9C31h PARLAY`）：SLY 走 `9C73h` 的魅力判定，過了就放行；NICE／MEEK 直接開打。
        return bool(dr.pick(s, ['SLY']))
    for want in ('DISGUISE PARTY AS MONSTERS', 'STAND AND LISTEN', 'WAIT FOR WINNER'):
        if dr.pick_line(s, want):
            return True
    return False


def seg_podol(dr):
    """波多廣場拍賣（spec 137〈波多廣場的委任要先接〉）：陸路進場答喬裝，拍賣格（地形 1）聽到得標者，
    `4AB0 = FE`；原路回市政廳交件，`4AB0 = FF`、`4AC1 = 2`。"""
    here = lambda: (dr.where()['archive'], dr.where()['block'])
    # 每一步看目前在哪、旗標到哪，從狀態檔接續時跳過已經做完的。
    if dr.d.class0(0x4AB0) < 0xFE:
        dr.settle(podol_policy)
        if here() == (3, 0):
            cross(dr, (3, 0), 3)
        if here() == (2, 20):
            cross(dr, (2, 20), 3, avoid_codes=SLUMS_AVOID)
        if here() == (8, 29):
            cross(dr, (8, 29), 3, safe_codes=KUTO_SAFE, policy=podol_policy)
        dr.settle(podol_policy)
        w = dr.where()
        if (w['archive'], w['block']) != (1, 18):
            raise Stuck('Kuto west edge led to %s' % w)
        dr.say('Podol: 4A35=%02X 4AB0=%02X' % (dr.d.class0(0x4A35), dr.d.class0(0x4AB0)))
        walk_to(dr, lambda x, y: terrain_code(w, x, y) == 1, podol_policy)
        dr.settle(podol_policy)
        for _ in range(4):
            # 站上地形 1 那一步拍賣還沒開，下一次前進時（這一步沒有移動）入口才跑到 `A2A3h`（實測）。
            if dr.d.class0(0x4AB0) == 0xFE:
                break
            dr.press('Up')
            dr.settle(podol_policy)
        if dr.d.class0(0x4AB0) != 0xFE:
            raise Stuck('auction did not close: 4AB0=%02X 4A35=%02X' % (dr.d.class0(0x4AB0), dr.d.class0(0x4A35)))
    if here() == (1, 18):
        cross(dr, (1, 18), 1, policy=podol_policy)
    if here() == (8, 29):
        cross(dr, (8, 29), 1, safe_codes=KUTO_SAFE)
    if here() == (2, 20):
        cross(dr, (2, 20), 1, avoid_codes=SLUMS_AVOID)
    city_hall(dr, lambda f: f['4AB0'] == 'FF' and int(f['4AC1'], 16) >= 2)


def step_onto(dr, target, policy=None, avoid=None):
    """走到 target 格旁邊再踏上去（已經站在上面就先退一格）：事件在踏進去那一步跑。"""
    w = dr.where()
    if target(w['x'], w['y']):
        for d in range(4):
            if dr.passable(w['geo'], w['x'], w['y'], d):
                dr.step(d, policy)
                break
    w = dr.where()
    def beside(x, y):
        return not target(x, y) and any(target((x + DELTA[d][0]) % 16, (y + DELTA[d][1]) % 16) and dr.passable(w['geo'], x, y, d)
                                        for d in range(4))
    if not beside(w['x'], w['y']):
        walk_to(dr, beside, policy, avoid)
    w = dr.where()
    for d in range(4):
        if target((w['x'] + DELTA[d][0]) % 16, (w['y'] + DELTA[d][1]) % 16) and dr.passable(w['geo'], w['x'], w['y'], d):
            return dr.step(d, policy)
    raise Stuck('step_onto: no side to step from %s' % w)


def answer_policy(answer):
    """YES／NO 選單答 answer；其他選單照打架的預設。"""
    def policy(dr, s):
        opts = [w for w, _, _ in s.options()]
        if 'YES' in opts and 'NO' in opts:
            return bool(dr.pick(s, [answer]))
        return bool(dr.pick(s, ['FIGHT', 'ATTACK', 'COMBAT']))
    return policy


def well_policy(direction):
    """古托井的上下問句：direction 是 'DOWN' 或 'UP'；問的是另一個方向就答 NO。井底一落地就問
    「CLIMB UP?」，一律答 YES 會在井口與井底之間來回（實測）。"""
    def policy(dr, s):
        text = s.flat()
        opts = [w for w, _, _ in s.options()]
        if 'YES' in opts and 'NO' in opts:
            if 'CLIMB DOWN' in text:
                return bool(dr.pick(s, ['YES' if direction == 'DOWN' else 'NO']))
            if 'CLIMB UP' in text:
                return bool(dr.pick(s, ['YES' if direction == 'UP' else 'NO']))
            return bool(dr.pick(s, ['NO']))
        return bool(dr.pick(s, ['FIGHT', 'ATTACK', 'COMBAT']))
    return policy


def seg_norris(dr):
    """古托井的諾里斯（remake `fightNorris`，`ecl8/29`）：踩井打兩場狗頭人（`4A23 |= 1`）→ 再踩井答 YES
    爬下去（GEO8/32）→ 井底密門答 NO → 地形 12 選 FIGHT（`9DFCh`：`4A24 = FF`、槽 0 `4AA6 = FE`）→
    井底 (7,7) 地形 16 答 YES 爬上來 → 原路回市政廳交件（`4AA6 = FF`、`4AC1 = 3`）。"""
    here = lambda: (dr.where()['archive'], dr.where()['block'])
    if dr.d.class0(0x4AA6) < 0xFE:
        # 從狀態檔接續時可能正停在井邊的「CLIMB DOWN?」：狗頭人打完就答 YES。
        dr.settle(well_policy('DOWN' if dr.d.class0(0x4A23) & 1 else 'NONE'))
        if here() == (3, 0):
            cross(dr, (3, 0), 3)
        if here() == (2, 20):
            cross(dr, (2, 20), 3, avoid_codes=SLUMS_AVOID)
        kuto = lambda x, y: terrain_code(dr.where(), x, y) not in KUTO_SAFE
        for visit in range(6):
            w = dr.where()
            if w['geo'] != 29 or dr.d.class0(0x4A23) & 1:
                break
            step_onto(dr, lambda x, y: terrain_code(w, x, y) == 1, well_policy('NONE'), kuto)
            dr.settle(well_policy('NONE'))
            dr.say('well visit %d: 4A23=%02X' % (visit + 1, dr.d.class0(0x4A23)))
        if dr.where()['geo'] == 29:
            w = dr.where()
            step_onto(dr, lambda x, y: terrain_code(w, x, y) == 1, well_policy('DOWN'), kuto)
            dr.settle(well_policy('DOWN'))
        w = dr.where()
        if w['geo'] != 32:
            raise Stuck('climbing down did not reach GEO32: %s' % w)
        dr.say('underground at (%d,%d)' % (w['x'], w['y']))
        for attempt in range(4):
            if dr.d.class0(0x4AA6) >= 0xFE:
                break
            w = dr.where()
            walk_to(dr, lambda x, y: terrain_code(w, x, y) == 12, well_policy('DOWN'))
            dr.settle(well_policy('DOWN'))
            dr.say('Norris attempt %d: 4A24=%02X 4AA6=%02X at (%d,%d)' % (attempt + 1, dr.d.class0(0x4A24), dr.d.class0(0x4AA6),
                                                                     dr.where()['x'], dr.where()['y']))
        if dr.d.class0(0x4AA6) != 0xFE:
            raise Stuck('Norris did not fall: 4AA6=%02X' % dr.d.class0(0x4AA6))
    if dr.where()['geo'] == 32:
        w = dr.where()
        step_onto(dr, lambda x, y: terrain_code(w, x, y) == 16, well_policy('UP'))
        dr.settle(well_policy('UP'))
        if dr.where()['geo'] != 29:
            raise Stuck('climbing up did not reach GEO29: %s' % dr.where())
    if here() == (8, 29):
        cross(dr, (8, 29), 1, safe_codes=KUTO_SAFE)
    if here() == (2, 20):
        cross(dr, (2, 20), 1, avoid_codes=SLUMS_AVOID)
    city_hall(dr, lambda f: f['4AA6'] == 'FF' and int(f['4AC1'], 16) >= 3)


def sokal_policy(dr, s):
    """索寇要塞（spec 137 第 4 段，`ecl4/21`）與港務長、碼頭的問句。"""
    text = s.flat()
    opts = [w for w, _, _ in s.options()]
    if 'TYPE A SINGLE WORD' in text:
        # 費蘭（地形 12，`AA7Eh INPUT STRING 3` 比 `LUX`）；中庭巡邏的密語是 SHESTNI（`9E9Ah`）。
        word = 'LUX' if dr.where()['terrain'] & 0x7F == 12 else 'SHESTNI'
        # 逐字送：一次 `type:LUX` 塞三個字進佇列，費蘭判成答錯（`4A13 = FF`，之後永遠不出現；實測）。
        for ch in word:
            dr.press(ch)
        dr.press('Return')
        return True
    if 'PALE FORM' in text and 'PARLAY' in opts:
        return bool(dr.pick(s, ['PARLAY']))
    if 'HAUGHTY' in opts:
        # 費蘭的 `AA50h PARLAY 0 0 0 0 0`：五種語氣都接到輸入密語，選哪個都一樣。
        return bool(dr.pick(s, ['NICE']))
    if 'TELL' in opts and any(o.startswith('LIE') for o in opts):
        return bool(dr.pick(s, ['TELL']))
    if 'YES' in opts and 'NO' in opts and 'BOAT' in text:
        return bool(dr.pick(s, ['YES']))
    return False


def seg_sokal(dr):
    """索寇要塞：港務長 (11,2) 朝北拿免費船票（`ecl3/0 9D54h`）→ (15,1) 上船 → 費蘭 PARLAY、LUX、
    TELL THE TRUTH（`4AA7 = FE`）→ 走出地圖邊界搭船回城（`6DD5h` 入口 0）→ 市政廳交件（`4AA7 = FF`、`4AC1 = 4`）。"""
    here = lambda: (dr.where()['archive'], dr.where()['block'])
    if dr.d.class0(0x4AA7) < 0xFE:
        dr.settle(sokal_policy)
        if here() == (3, 0):
            w = dr.where()
            walk_to(dr, lambda x, y: (x, y) == (11, 2), sokal_policy)
            dr.face(0, sokal_policy)
            dr.press('Up')
            dr.settle(sokal_policy)
            dr.say('harbour master: 4A01=%02X at %s' % (dr.d.class0(0x4A01), dr.where()))
            walk_to(dr, lambda x, y: (x, y) == (15, 1), sokal_policy)
            dr.settle(sokal_policy)
            for _ in range(3):
                if here() != (3, 0):
                    break
                dr.press('Up')
                dr.settle(sokal_policy)
        if here() != (4, 21):
            raise Stuck('the Sokal boat did not land at ECL4/21: %s' % dr.where())
        for attempt in range(6):
            if dr.d.class0(0x4AA7) >= 0xFE:
                break
            w = dr.where()
            walk_to(dr, lambda x, y: terrain_code(w, x, y) == 12, sokal_policy)
            dr.settle(sokal_policy)
            if dr.d.class0(0x4AA7) < 0xFE:
                dr.press('Up')
                dr.settle(sokal_policy)
            dr.say('Ferran attempt %d: 4AA7=%02X 4A13=%02X 4A26=%02X at %s' % (
                attempt + 1, dr.d.class0(0x4AA7), dr.d.class0(0x4A13), dr.d.class0(0x4A26), dr.where()))
        if dr.d.class0(0x4AA7) != 0xFE:
            raise Stuck('Ferran did not close Sokal Keep: 4AA7=%02X' % dr.d.class0(0x4AA7))
    if here() == (4, 21):
        cross(dr, (4, 21), 2, policy=sokal_policy)
        dr.settle(sokal_policy)
    city_hall(dr, lambda f: f['4AA7'] == 'FF' and int(f['4AC1'], 16) >= 4)


def ending_policy(dr, s):
    """結局段的選單（spec 137 第 6～11 段）。"""
    text = s.flat()
    opts = [w for w, _, _ in s.options()]
    if 'EAST' in opts and 'WEST' in opts:         # 港務長的航線（`ecl3/0 9CF0h`）
        return bool(dr.pick(s, ['EAST']))
    if 'WHO WILL PAY' in s.bottom():
        # 付船票：游標那一位付不起就往下換一位（End）再選。
        if 'NOT HAVE ENOUGH' in text or "DON\"T HAVE ENOUGH" in text or 'ENOUGH PLATINUM' in text:
            dr.press('End')
        return bool(dr.pick(s, ['SELECT']))
    if 'YES' in opts and 'NO' in opts and 'BOAT' in text:
        return bool(dr.pick(s, ['YES']))
    if 'STAY' in opts and 'TAKE' in opts and 'BOAT' in text:
        # 野外的回程船（ecl8/27 地點 3、ecl7/26 地點 7）：要往野外走，不搭。
        return bool(dr.pick(s, ['STAY']))
    if 'NORTH' in opts and dr.where()['block'] == 26:
        return bool(dr.pick(s, ['NORTH']))
    if 'HEDGE WITH POISONOUS' in text and 'YES' in opts:
        # 城堡內部的毒荊棘：答 NO。砍過去被刺中是直接死亡（「D DIES.」），不經過扣血入口 `2266h`，
        # 攔截蓋不到（實測）；答 NO 被擋住之後由駕駛穿牆。
        dr.hedge_refusals += 1
        return bool(dr.pick(s, ['NO']))
    if ('GO DOWN THESE STAIRS' in text or 'GO UP THESE STAIRS' in text) and 'NO' in opts:
        return bool(dr.pick(s, ['NO']))
    if 'WHAT IS THE PASSWORD' in text:
        return dr.pick_line(s, 'GIVE THE PASSWORD')
    if 'WHAT PASSWORD DO YOU GIVE' in text:
        # `ecl5/3 AA5Ch` 比 RHODIA；逐字送（一次塞整串會判錯，費蘭那一場實測）。
        for ch in 'RHODIA':
            dr.press(ch)
        dr.press('Return')
        return True
    if 'ATTACK' in opts and 'JOIN' in opts:
        # 覲見廳的龍：每個角色都投 ATTACK（選 JOIN 的角色被附身離隊，spec 137）。
        return bool(dr.pick(s, ['ATTACK']))
    return bool(dr.pick(s, ['COMBAT', 'ATTACK', 'FIGHT']))


def wild_step(dr, dx, dy, policy):
    """野外走一步：方向鍵直接往那個方向走（Up 北、Right 東、Down 南、Left 西，朝向跟著改；實測）。
    只走正四方向：先 X 後 Y。"""
    dr.settle(policy)
    key = 'Left' if dx < 0 else 'Right' if dx > 0 else 'Up' if dy < 0 else 'Down'
    dr.press(key)
    dr.settle(policy)


# 野外地點格（spec 105 的地點表）：踩上去就跑地點腳本，常常 `NEWECL` 把隊伍帶走，路過時要繞開。
WILD_SITES = {
    25: {(3, 8), (10, 8), (11, 8), (9, 9), (10, 9), (11, 9), (9, 10), (10, 10), (8, 24), (12, 30), (13, 30), (2, 31), (3, 31),
         (11, 31), (12, 31), (13, 31), (2, 32), (3, 32), (4, 32), (12, 32), (13, 32), (2, 33), (3, 33)},
    26: {(12, 10), (13, 10), (11, 11), (12, 11), (13, 11), (12, 12), (13, 12), (12, 26), (13, 26), (13, 27), (11, 28), (6, 14),
         (6, 16), (7, 29)},
    27: {(11, 8), (6, 14), (7, 14), (5, 15), (6, 15), (7, 15), (6, 16), (7, 16), (9, 29)},
}


def wilderness_to(dr, block, target, policy, limit=400):
    """在野外往 (block, target) 走（25 → 26 → 27 由西到東；跨圖是走到 X = 2 或 15 再踏出去，spec 105）。
    擋路的格子沒有照原版算：邊走邊記「這一格往這個方向走不動」，每一步挑離目標最近、踩過次數最少的方向。"""
    blocked, visits = set(), collections.Counter()
    for _ in range(limit):
        w = dr.where()
        if w['archive'] not in (6, 7, 8) or w['block'] not in (25, 26, 27):
            return w
        b = w['block']
        x, y = dr.d.class0(0x49C3), dr.d.class0(0x49C4)
        if b == block and (x, y) == target:
            return w
        visits[(b, x, y)] += 1
        if b > block:
            goal = lambda nx, ny: nx - 2
        elif b < block:
            goal = lambda nx, ny: 15 - nx
        else:
            goal = lambda nx, ny: abs(nx - target[0]) + abs(ny - target[1])
        moves = []
        for dx, dy in ((-1, 0), (1, 0), (0, -1), (0, 1)):
            if (b, x, y, dx, dy) in blocked:
                continue
            nx, ny = x + dx, y + dy
            if (nx, ny) in WILD_SITES.get(b, ()) and not (b == block and (nx, ny) == target):
                continue
            moves.append((goal(nx, ny) + 2 * visits[(b, nx, ny)], dx, dy))
        if not moves:
            raise Stuck('wilderness: every direction blocked at ECL%d (%d,%d)' % (b, x, y))
        _, dx, dy = min(moves)
        wild_step(dr, dx, dy, policy)
        after = dr.where()
        if (after['block'], dr.d.class0(0x49C3), dr.d.class0(0x49C4)) == (b, x, y):
            blocked.add((b, x, y, dx, dy))
    raise Stuck('wilderness walk limit at %s (%d,%d)' % (dr.where(), dr.d.class0(0x49C3), dr.d.class0(0x49C4)))


def castle_code(w, x, y):
    return GEO[str(w['geo'])][y][x]['t'] & 0x1F


def seg_ending(dr):
    """東航線 → 野外 → 波多廣場北緣 → 斯托亞諾夫城門 → 城堡 → 覲見廳（spec 137 第 6～11 段）。"""
    here = lambda: (dr.where()['archive'], dr.where()['block'])
    pol = ending_policy
    dr.settle(pol)
    if here() == (3, 0):
        if dr.d.class0(0x4AC4) != 1 or dr.where()['y'] > 2:
            walk_to(dr, lambda x, y: (x, y) == (11, 2), pol)
            dr.face(0, pol)
            dr.press('Up')
            dr.settle(pol)
            dr.say('harbour master: 4AC4=%02X at %s' % (dr.d.class0(0x4AC4), dr.where()))
        walk_to(dr, lambda x, y: (x, y) == (15, 1), pol)
        dr.settle(pol)
        for _ in range(3):
            if here() != (3, 0):
                break
            dr.press('Up')
            dr.settle(pol)
    if here()[1] in (25, 26, 27):
        dr.say('wilderness: %s (%d,%d)' % (here(), dr.d.class0(0x49C3), dr.d.class0(0x49C4)))
        wilderness_to(dr, 26, (11, 28), pol)
        for _ in range(4):
            if here() == (1, 18):
                break
            dr.settle(pol)
            if here() == (1, 18):
                break
            dr.press('Up')
            dr.settle(pol)
    if here() == (1, 18):
        w = dr.where()
        dr.say('Podol from the wilderness at (%d,%d) hour %d' % (w['x'], w['y'], dr.d.class0(0x49C9)))
        if dr.d.class0(0x49C9) >= 12:
            camp_until(dr, 6)
        cross(dr, (1, 18), 0, policy=pol)
        dr.settle(pol)
    if here() == (2, 9):
        dr.say('Stojanow gate at %s hour %d 4A77=%02X' % (dr.where(), dr.d.class0(0x49C9), dr.d.class0(0x4A77)))
        guards = lambda x, y: castle_code(dr.where(), x, y) == 3
        for attempt in range(4):
            if dr.d.class0(0x4A77) & 64:
                break
            if dr.d.class0(0x49C9) >= 14:
                camp_until(dr, 6)
            w = dr.where()
            step_onto(dr, lambda x, y: castle_code(w, x, y) == 9 and y == 14, pol, guards)
            dr.settle(pol)
            dr.say('merchant attempt %d: 4A77=%02X' % (attempt + 1, dr.d.class0(0x4A77)))
        cross(dr, (2, 9), 0, avoid_codes=None, policy=pol)
        dr.settle(pol)
    if here() in ((5, 3), (5, 4), (5, 6)):
        dr.say('castle outer at %s' % dr.where())
        # 城堡是四張 GEO 拼成的 32×32（`ecl5/5 9AEBh`：3 東→4、南→6；4 西→3；5 北→4、西→6；6 北→3）。
        # 城門格（地形 21）在 GEO3 的 (4,7)／(4,8)，踩上去 `ecl5/3 B38Ah NEWECL 5`。
        for _ in range(12):
            if here() == (5, 5):
                break
            w = dr.where()
            if w['geo'] == 3:
                # 地形 30／31 是外圈的密語城門，remake 從東側繞到地形 21，這裡也繞開。
                walk_to(dr, lambda x, y: castle_code(w, x, y) == 21, pol, avoid=lambda x, y: castle_code(w, x, y) in (30, 31))
                dr.settle(pol)
                if here() != (5, 5):
                    dr.press('Up')
                    dr.settle(pol)
                continue
            side = {6: 0, 4: 3, 5: 0}.get(w['geo'])
            if side is None:
                raise Stuck('castle outer: unexpected GEO%d at %s' % (w['geo'], w))
            edge = lambda x, y, side=side: (side == 0 and y == 0) or (side == 3 and x == 0)
            if not edge(w['x'], w['y']):
                walk_to(dr, edge, pol)
            dr.step(side, pol)
        if here() != (5, 5):
            raise Stuck('castle outer did not reach the interior script: %s' % dr.where())
    if here() == (5, 5):
        dr.say('castle interior at %s' % dr.where())
        for _ in range(6):
            w = dr.where()
            if here() != (5, 5):
                break
            if w['geo'] == 3 and (w['x'], w['y']) == (13, 15):
                dr.step(1, pol)
                continue
            inner = lambda x, y: castle_code(w, x, y) in (30, 31)
            walk_to(dr, lambda x, y: (x, y) == (13, 15) and w['geo'] == 3, pol, inner)
    if here() == (5, 7):
        dr.say('audience hall at %s' % dr.where())
        for _ in range(6):
            if dr.d.class0(0x4ABA) == 0xFE:
                break
            w = dr.where()
            if castle_code(w, w['x'], w['y']) == 8:
                dr.press('Up')
                dr.settle(pol)
            else:
                # 只走地形 0：1／2 是上下樓梯、4 活板門、5 美杜莎、3／6／7／9／10 是房間與 NPC（spec 137 GEO5/7 那張圖）。
                walk_to(dr, lambda x, y: castle_code(w, x, y) == 8, pol, avoid=lambda x, y: castle_code(w, x, y) not in (0, 8))
                dr.settle(pol)
            dr.say('audience hall: 4ABA=%02X 4A6D=%02X at %s' % (dr.d.class0(0x4ABA), dr.d.class0(0x4A6D), dr.where()))
    if dr.d.class0(0x4ABA) != 0xFE:
        raise Stuck('ending not reached: 4ABA=%02X at %s' % (dr.d.class0(0x4ABA), dr.where()))
    for _ in range(40):
        s = dr.screen()
        if dr.mode(s) == 'move':
            break
        page = s.flat()
        if page and page not in dr.ending_pages:
            dr.ending_pages.append(page)
        if not dr.generic(s) and not pol(dr, s):
            dr.press('Return')
    dr.say('ending: 4ABA=FE, %d text pages, back at %s' % (len(dr.ending_pages), dr.where()))


BOOT_KEYS_FROM = '/audit/dosgolem-deployment-peek-goblins.json'


def boot_keys():
    keys = json.load(open(BOOT_KEYS_FROM))['keys']
    tail = ',e,b,rep:14:Return,Up,Up,Up'
    return keys[:keys.index(tail) + len(tail)]


def seg_boot(dr):
    """標題 → 建六人隊伍 → 開場導覽 → 城區往西進貧民窟（鍵序沿用 #33 的部署收據）。"""
    w = dr.where()
    if (w['archive'], w['block']) != (2, 20):
        raise Stuck('boot keys did not reach the slums: %s' % w)


SEGMENTS = collections.OrderedDict([
    ('boot', seg_boot),
    ('slums', seg_slums),
    ('handin-slums', seg_handin_slums),
    ('podol', seg_podol),
    ('norris', seg_norris),
    ('sokal', seg_sokal),
    ('ending', seg_ending),
])


def main():
    name = sys.argv[1]
    load = None
    if '--from' in sys.argv:
        load = sys.argv[sys.argv.index('--from') + 1]
    else:
        names = list(SEGMENTS)
        load = os.path.join(WORK, names[names.index(name) - 1] + '.state') if names.index(name) else None
    if name == 'boot':
        load = None
    dr = Driver(name, load, boot_keys() if name == 'boot' else '')
    started = time.time()
    status = 'done'
    try:
        SEGMENTS[name](dr)
    except Stuck as e:
        status = 'stuck: %s' % e
        dr.say(status)
        dr.d.save(os.path.join(WORK, name + '-stuck.state'))
    else:
        dr.settle()
        dr.d.save(os.path.join(WORK, name + '.state'))
    receipt_path = os.path.join(WORK, 'receipt.json')
    receipt = json.load(open(receipt_path)) if os.path.exists(receipt_path) else {}
    state = os.path.join(WORK, name + ('.state' if status == 'done' else '-stuck.state'))
    if 'runs' not in receipt.get(name, {}):
        receipt[name] = {'runs': []}
    receipt[name]['runs'].append({'status': status, 'load': load, 'seconds': round(time.time() - started),
                     'actions': dr.actions, 'boot_keys': dr.boot_keys, 'keys': dr.d.keys, 'steps': dr.d.steps,
                     'state_sha256': hashlib.sha256(open(state, 'rb').read()).hexdigest(),
                     'end': dr.where(), 'flags': dr.flags(), 'checkpoints': dr.checkpoints,
                     'wall_clears': dr.wall_clears, 'ending_pages': dr.ending_pages,
                     'damage_intercepts': len(dr.d.damage),
                     'party_damage_nonzero_after_hook': sum(1 for r in dr.d.damage if (not r.get('in_combat') or r.get('side') == 0) and r.get('changed', 0) != 0)})
    json.dump(receipt, open(receipt_path, 'w'), indent=1)
    dr.d.quit()
    dr.say('%s: %s in %ds, %d actions' % (name, status, time.time() - started, dr.actions))


if __name__ == '__main__':
    main()
