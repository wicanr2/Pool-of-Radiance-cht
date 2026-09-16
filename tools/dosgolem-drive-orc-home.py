#!/usr/bin/env python3
"""看畫面再送鍵的 dosgolem 驅動（#33）：把五人隊從貧民窟入口走到三場固定事件（獸人的家、
衛兵攔截、驚動衛兵）開打，讀部署表與之後兩批敵方走位。每一步從上一步存的狀態展開
（shots -load-state／-save-state，一步約 1 秒），送一小段鍵，讀底部指令列（色號列 192..199
的 md5）分辨現在是走路／遭遇選單／戰鬥／LOCKED 選單，再決定下一段鍵：遭遇選單按 w
（WAIT，本次表怪物撤退）、被突襲進戰鬥或 DS 抽樣到別段就在前一步插 Left,Right 擾動時序
重跑、LOCKED 按 b 到開為止、其他文字按一下 Return。

用法（主機端，Docker 由本檔起）：
  python3 tools/dosgolem-drive-orc-home.py
產物在 workplace/orc-drive/：drive.log、<場>-fight.json（開打那一幀與走位快照）。
收據整理後放 docs/audit/dosgolem-deployment-peek-{orc-home,guards,alarm}.json。"""
import json, os, subprocess, sys, shutil, hashlib
ROOT=os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SIG={'cf41dc77f12777e0e3763dfb64c6f974':'adventure','1c7f49e1706c3f0fb8d8e6844f424275':'encounter','476956d246cbb3fe72a101c2ae636c7e':'encounter','ae3777220412b286f1fe7796c0769a65':'combat','3f480272575a4b7d4f4a80fa4e5e70dc':'locked'}
DOSGOLEM=os.path.join(ROOT,'workplace','dosgolem')
SOURCE=os.path.join(ROOT,'workplace','oracle','dos')
WORK=os.path.join(ROOT,'workplace','orc-drive')
PEEK="ds:2D0:16,ds:6A0B:3,ds:45B2:9,ds:6772:2,ds:5E85:200,ds:6039:1250"
os.makedirs(WORK,exist_ok=True)
for d in ('gocache','gomodcache'): os.makedirs(os.path.join(DOSGOLEM,'workplace',d),exist_ok=True)

def run(keys, load, save, tag, budget=150000000):
    out=os.path.join(WORK,tag); shutil.rmtree(out,ignore_errors=True); os.makedirs(out)
    # 暫存層要跨步驟留著：狀態檔記的是開著的檔（game.ovr 之類），少了它還原不了。
    scratch=os.path.join(WORK,'scratch'); os.makedirs(scratch,exist_ok=True)
    if not load: shutil.rmtree(scratch); os.makedirs(scratch)
    args=['docker','run','--rm','--network','none','--memory','4g','--cpus','2','--pids-limit','256',
      '--log-opt','max-size=10m','--log-opt','max-file=3','-u','%d:%d'%(os.getuid(),os.getgid()),
      '-v',DOSGOLEM+':/dosgolem','-v',SOURCE+':/orig:ro','-v',WORK+':/work','-v',scratch+':/scratch',
      '-v',os.path.join(DOSGOLEM,'workplace','gocache')+':/gocache','-v',os.path.join(DOSGOLEM,'workplace','gomodcache')+':/gomodcache',
      '-e','GOCACHE=/gocache','-e','GOMODCACHE=/gomodcache','-e','HOME=/tmp','-e','GOFLAGS=-mod=mod',
      '-w','/dosgolem','golang:1.24-bookworm','go','run','./cmd/shots','-exe','/orig/start.exe','-root','/orig','-scratch','/scratch',
      '-out','/work/'+tag,'-budget',str(budget),'-idle','3000000','-peek',PEEK,'-trace-peek','ds:5E85:200,ds:6772:2','-trace-call','5BB:C94','-keys',keys]
    if load: args+=['-load-state','/work/'+load]
    if save: args+=['-save-state','/work/'+save]
    p=subprocess.run(args,capture_output=True,text=True,timeout=1800)
    open(os.path.join(out,'stdout.txt'),'w').write(p.stdout+p.stderr)
    if p.returncode!=0: print(p.stdout[-2000:],p.stderr[-2000:]); raise SystemExit('shots failed')
    return json.load(open(os.path.join(out,'shots.json')))

def bar(idxpath):
    b=open(idxpath,'rb').read(); return hashlib.md5(b[192*320:200*320]).hexdigest()

def classify(shot,tag):
    h=bar(os.path.join(WORK,tag,shot['path'].replace('.png','.idx')))
    return SIG.get(h,'unknown:'+h)

ROUTE=[((14,4),'S'),((14,5),'S'),((14,6),'S'),((14,7),'W'),((13,7),'N'),((13,6),'N'),((13,5),'N'),((13,4),'N'),((13,3),'N'),((13,2),'W'),((12,2),'W'),((11,2),'W'),((10,2),'W'),((9,2),'W'),((8,2),'W'),((7,2),'S'),((7,3),'W'),((6,3),'W'),((5,3),'S'),((5,4),'W'),((4,4),'W'),((3,4),'N')]
FACE={'N':0,'E':1,'S':2,'W':3}
def turns(cur,want):
    t=(want-cur)%4
    return {0:[],1:['Right'],3:['Left'],2:['Right','Right']}[t]
def pos(shot):
    seg,v=shot['peek']['ds:6A0B'].split('|')
    if seg!='0850': return None
    return (int(v[0:2],16),int(v[2:4],16),int(v[4:6],16)//2)

def drive(start_state, start_index, facing, log, route=None, prefix='r'):
    route=route or ROUTE
    state=start_state
    for i in range(start_index,len(route)):
        cell,d=route[i]; want=FACE[d]
        keys=turns(facing,want)+['Up']
        tag='%s%02d'%(prefix,i)
        for attempt in range(6):
            pre=['Left','Right']*attempt
            shots=run(','.join(pre+keys),state,tag+'.state',tag)
            last=shots[-1]; kind=classify(last,tag); p=pos(last)
            log.write('%s attempt %d keys %s → %s pos %s\n'%(tag,attempt,pre+keys,kind,p)); log.flush()
            if p is None and kind=='adventure':
                continue  # DS 抽樣到別段，換個時序重跑
            nxt=(cell[0]+[0,1,0,-1][want], cell[1]+[-1,0,1,0][want])
            if kind=='adventure' and (p[0],p[1])==nxt:
                facing=want; state=tag+'.state'; break
            if kind=='adventure':
                print('STOP: did not move at',tag,'pos',p,'want',nxt,'frame',os.path.join(WORK,tag,last['path'])); return None,i,facing,state
            if kind=='encounter':
                shots=run('w',tag+'.state',tag+'w.state',tag+'w'); last=shots[-1]; k2=classify(last,tag+'w')
                log.write('   WAIT → %s pos %s\n'%(k2,pos(last))); log.flush()
                if k2=='adventure' and pos(last) and pos(last)[:2]==nxt:
                    facing=want; state=tag+'w.state'; break
                if k2=='adventure':
                    print('STOP after WAIT at',tag,pos(last),'frame',os.path.join(WORK,tag+'w',last['path'])); return None,i,facing,state
                continue
            if kind=='locked':
                st=tag+'.state'; opened=False
                for bash in range(12):
                    shots=run('b',st,tag+'b%d.state'%bash,tag+'b%d'%bash); last=shots[-1]; k2=classify(last,tag+'b%d'%bash); p2=pos(last)
                    log.write('   BASH %d → %s pos %s\n'%(bash,k2,p2)); log.flush()
                    st=tag+'b%d.state'%bash
                    if k2=='adventure' and p2 and p2[:2]==nxt: opened=True; break
                    if k2=='adventure':
                        shots=run('Up',st,tag+'bu.state',tag+'bu'); last=shots[-1]; k3=classify(last,tag+'bu'); p3=pos(last)
                        log.write('   after bash Up → %s pos %s\n'%(k3,p3)); log.flush()
                        st=tag+'bu.state'
                        if k3=='adventure' and p3 and p3[:2]==nxt: opened=True; break
                        if k3!='locked':
                            print('STOP after bash+Up at',tag,k3,p3,'frame',os.path.join(WORK,tag+'bu',last['path'])); return None,i,facing,state
                    elif k2!='locked':
                        print('STOP in bash at',tag,k2,p2,'frame',os.path.join(WORK,tag+'b%d'%bash,last['path'])); return None,i,facing,state
                if opened:
                    facing=want; state=st; break
                print('bash never opened at',tag); return None,i,facing,state
            if kind=='combat':
                continue
            if kind.startswith('unknown'):
                # 多半是格子事件的文字（PRESS RETURN）：按一下 Return 看會不會回到走路。
                shots=run('Return',tag+'.state',tag+'t.state',tag+'t'); last=shots[-1]; k2=classify(last,tag+'t'); p2=pos(last)
                log.write('   text? Return → %s pos %s\n'%(k2,p2)); log.flush()
                if k2=='adventure' and p2 and p2[:2]==nxt:
                    facing=want; state=tag+'t.state'; break
            print('STOP at',tag,kind,p,'frame',os.path.join(WORK,tag,last['path'])); return None,i,facing,state
        else:
            print('gave up at',tag); return None,i,facing,state
    return state,len(ROUTE),facing,state

ROUTE_GUARDS=ROUTE[:20]+[((4,4),'S'),((4,5),'S'),((4,6),'W'),((3,6),'W'),((2,6),'S'),((2,7),'W'),((1,7),'S'),((1,8),'W'),((0,8),'N')]
ROUTE_ALARM=ROUTE[:20]+[((4,4),'S'),((4,5),'S'),((4,6),'W'),((3,6),'W'),((2,6),'S'),((2,7),'S'),((2,8),'S'),((2,9),'S'),((2,10),'S'),((2,11),'E')]
BASE_KEYS=("rep:9:Space,Return,Return,"+"".join("c,rep:6:Return,y,%s,Return,k,e,y,"%L for L in "ABCDEF")+
  "a,End,a,End,a,End,a,End,a,End,a,e,b,rep:14:Return,Up,Up,Up,rep:10:Return")

def fight(start,prefix,batches=2,combat_state=None):
    """從踏上事件格的狀態按 Return 到戰鬥畫面，記開打那一幀，再每個隊員 d,g 兩批記走位。
    combat_state 給的是已經在戰鬥畫面的狀態檔（同目錄下要有同名的 shots 目錄），跳過按 Return 那一段。"""
    if combat_state:
        st=combat_state; shots=json.load(open(os.path.join(WORK,combat_state[:-6],'shots.json')))
    else:
        st=start
        for n in range(8):
            shots=run('Return',st,'%s%d.state'%(prefix,n),'%s%d'%(prefix,n)); st='%s%d.state'%(prefix,n)
            if classify(shots[-1],'%s%d'%(prefix,n))=='combat': break
        else: raise SystemExit('%s：沒進戰鬥'%prefix)
    dep=shots[-1]; rows=[]; prev=None; actions=[]; prev_a=None; dice=[]
    def table_of(p):
        b=bytes.fromhex(p['ds:5E85'].split('|')[1]); n=b[3]
        return [(k,b[4*k],b[4*k+1],b[4*k+3]) for k in range(1,n)]
    for r in range(1,batches+1):
        shots=run(','.join(['d','g']*5),st,'%sr%d.state'%(prefix,r),'%sr%d'%(prefix,r)); st='%sr%d.state'%(prefix,r)
        for s_ in shots:
            p=s_['peek']
            if p.get('ds:5E85','').startswith('0850|'):
                t=table_of(p)
                if t!=prev: rows.append({'round_batch':r,'frame':s_['index'],'label':s_['label'],'counts':p['ds:6772'][5:],'table':t}); prev=t
            # 逐步快照（-trace-peek）：一筆一步或一次死亡
            for snap in (s_.get('peek_trace') or []):
                q=snap['peek']
                if not q.get('ds:5E85','').startswith('0850|'): continue
                t=table_of(q)
                if t!=prev_a: actions.append({'round_batch':r,'frame':s_['index'],'label':s_['label'],'step':snap['step'],'counts':q.get('ds:6772','')[5:],'table':t}); prev_a=t
            # 骰流（-trace-call 5BB:C94，Turbo Pascal 的 Random(n)）：一筆一次，帶呼叫端
            for c in (s_.get('calls') or []):
                dice.append({'round_batch':r,'frame':s_['index'],'step':c['step'],'sides':c['arg'],'roll':c['result']+1,'caller':'%04X:%04X'%(c['caller_cs'],c['caller_ip'])})
    json.dump({'deploy_frame':{k:v for k,v in dep.items() if k!='path'},'rounds':rows,'actions':actions,'dice':dice},open(os.path.join(WORK,prefix+'-fight.json'),'w'))

if __name__=='__main__':
    log=open(os.path.join(WORK,'drive.log'),'a')
    if not os.path.exists(os.path.join(WORK,'tour.state')):
        shots=run(BASE_KEYS,None,'tour.state','boot'); log.write('boot → %s\n'%pos(shots[-1]))
    shots=run('Up,Up','tour.state','enter.state','enter'); log.write('enter → %s\n'%pos(shots[-1]))
    # 獸人的家：走完 ROUTE，最後一步撬門踏進 (3,3)
    state,i,facing,_=drive('enter.state',0,3,log)
    if state is None: raise SystemExit('沒走到獸人的家')
    fight(state,'orc')
    # 衛兵與驚動衛兵從 (4,4)（r19.state）分岔
    state,i,facing,_=drive('r19.state',20,3,log,ROUTE_GUARDS,'g')
    if state is None: raise SystemExit('沒走到衛兵')
    fight(state,'guards')
    state,i,facing,_=drive('g24.state',25,2,log,ROUTE_ALARM,'a')
    if state is None: raise SystemExit('沒走到驚動衛兵')
    fight(state,'alarm')
    print('done; 收據在', WORK)
