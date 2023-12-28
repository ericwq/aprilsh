SIGQUIT: quit
PC=0x46fd81 m=0 sigcode=128

goroutine 0 [idle]:
runtime.futex()
        /usr/lib/go/src/runtime/sys_linux_amd64.s:557 +0x21 fp=0x7fffcfaed5d0 sp=0x7fffcfaed5c8 pc=0x46fd81
runtime.futexsleep(0x7fffcfaed648?, 0x442cd6?, 0x7fffcfaed648?)
        /usr/lib/go/src/runtime/os_linux.go:69 +0x30 fp=0x7fffcfaed620 sp=0x7fffcfaed5d0 pc=0x437f50
runtime.notesleep(0x9e2248)
        /usr/lib/go/src/runtime/lock_futex.go:160 +0x87 fp=0x7fffcfaed658 sp=0x7fffcfaed620 pc=0x4114e7
runtime.mPark(...)
        /usr/lib/go/src/runtime/proc.go:1632
runtime.stoplockedm()
        /usr/lib/go/src/runtime/proc.go:2780 +0x73 fp=0x7fffcfaed6b0 sp=0x7fffcfaed658 pc=0x442eb3
runtime.schedule()
        /usr/lib/go/src/runtime/proc.go:3561 +0x3a fp=0x7fffcfaed6e8 sp=0x7fffcfaed6b0 pc=0x444cfa
runtime.park_m(0xc000082b60?)
        /usr/lib/go/src/runtime/proc.go:3745 +0x11f fp=0x7fffcfaed730 sp=0x7fffcfaed6e8 pc=0x44527f
traceback: unexpected SPWRITE function runtime.mcall
runtime.mcall()
        /usr/lib/go/src/runtime/asm_amd64.s:458 +0x4e fp=0x7fffcfaed748 sp=0x7fffcfaed730 pc=0x46bfce

## mainSrv.wait()
goroutine 1 [semacquire, 81 minutes]:
runtime.gopark(0x46c032?, 0xc000059e60?, 0x0?, 0x3?, 0xc000059e40?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000059e20 sp=0xc000059e00 pc=0x43e8ae
runtime.goparkunlock(...)
        /usr/lib/go/src/runtime/proc.go:404
runtime.semacquire1(0xc000070230, 0x1?, 0x1, 0x0, 0xa0?)
        /usr/lib/go/src/runtime/sema.go:160 +0x218 fp=0xc000059e88 sp=0xc000059e20 pc=0x44f3f8
sync.runtime_Semacquire(0x722383?)
        /usr/lib/go/src/runtime/sema.go:62 +0x25 fp=0xc000059ec0 sp=0xc000059e88 pc=0x46a665
sync.(*WaitGroup).Wait(0xc0000701e0?)
        /usr/lib/go/src/sync/waitgroup.go:116 +0x48 fp=0xc000059ee8 sp=0xc000059ec0 pc=0x4893a8
main.(*mainSrv).wait(...)
        /home/ide/develop/aprilsh/frontend/server/server.go:1539
main.main()
        /home/ide/develop/aprilsh/frontend/server/server.go:449 +0x24c fp=0xc000059f40 sp=0xc000059ee8 pc=0x68834c
runtime.main()
        /usr/lib/go/src/runtime/proc.go:267 +0x2bb fp=0xc000059fe0 sp=0xc000059f40 pc=0x43e45b
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000059fe8 sp=0xc000059fe0 pc=0x46dfc1

goroutine 2 [force gc (idle), 81 minutes]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000044fa8 sp=0xc000044f88 pc=0x43e8ae
runtime.goparkunlock(...)
        /usr/lib/go/src/runtime/proc.go:404
runtime.forcegchelper()
        /usr/lib/go/src/runtime/proc.go:322 +0xb3 fp=0xc000044fe0 sp=0xc000044fa8 pc=0x43e733
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000044fe8 sp=0xc000044fe0 pc=0x46dfc1
created by runtime.init.6 in goroutine 1
        /usr/lib/go/src/runtime/proc.go:310 +0x1a

goroutine 3 [GC sweep wait]:
runtime.gopark(0x1?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000045778 sp=0xc000045758 pc=0x43e8ae
runtime.goparkunlock(...)
        /usr/lib/go/src/runtime/proc.go:404
runtime.bgsweep(0x0?)
        /usr/lib/go/src/runtime/mgcsweep.go:321 +0xdf fp=0xc0000457c8 sp=0xc000045778 pc=0x42a7df
runtime.gcenable.func1()
        /usr/lib/go/src/runtime/mgc.go:200 +0x25 fp=0xc0000457e0 sp=0xc0000457c8 pc=0x41f945
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000457e8 sp=0xc0000457e0 pc=0x46dfc1
created by runtime.gcenable in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:200 +0x66

goroutine 4 [GC scavenge wait]:
runtime.gopark(0xdff527?, 0xdea52a?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000045f70 sp=0xc000045f50 pc=0x43e8ae
runtime.goparkunlock(...)
        /usr/lib/go/src/runtime/proc.go:404
runtime.(*scavengerState).park(0x9e18a0)
        /usr/lib/go/src/runtime/mgcscavenge.go:425 +0x49 fp=0xc000045fa0 sp=0xc000045f70 pc=0x428069
runtime.bgscavenge(0x0?)
        /usr/lib/go/src/runtime/mgcscavenge.go:658 +0x59 fp=0xc000045fc8 sp=0xc000045fa0 pc=0x428619
runtime.gcenable.func2()
        /usr/lib/go/src/runtime/mgc.go:201 +0x25 fp=0xc000045fe0 sp=0xc000045fc8 pc=0x41f8e5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000045fe8 sp=0xc000045fe0 pc=0x46dfc1
created by runtime.gcenable in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:201 +0xa5

goroutine 5 [finalizer wait]:
runtime.gopark(0x0?, 0x736038?, 0x0?, 0x80?, 0x2000000020?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000044620 sp=0xc000044600 pc=0x43e8ae
runtime.runfinq()
        /usr/lib/go/src/runtime/mfinal.go:193 +0x107 fp=0xc0000447e0 sp=0xc000044620 pc=0x41e967
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000447e8 sp=0xc0000447e0 pc=0x46dfc1
created by runtime.createfing in goroutine 1
        /usr/lib/go/src/runtime/mfinal.go:163 +0x3d

goroutine 6 [GC worker (idle)]:
runtime.gopark(0x20569884b5b32?, 0x3?, 0xf?, 0x45?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000046750 sp=0xc000046730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000467e0 sp=0xc000046750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000467e8 sp=0xc0000467e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 18 [GC worker (idle)]:
runtime.gopark(0x2056720f7f2f4?, 0x1?, 0x7e?, 0x8?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000040750 sp=0xc000040730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000407e0 sp=0xc000040750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000407e8 sp=0xc0000407e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 34 [GC worker (idle)]:
runtime.gopark(0x205699f22ab8a?, 0x1?, 0x8f?, 0x34?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000588750 sp=0xc000588730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0005887e0 sp=0xc000588750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0005887e8 sp=0xc0005887e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 7 [GC worker (idle)]:
runtime.gopark(0x20569884b664e?, 0x3?, 0x86?, 0x2c?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000046f50 sp=0xc000046f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000046fe0 sp=0xc000046f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000046fe8 sp=0xc000046fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 35 [GC worker (idle), 4 minutes]:
runtime.gopark(0x2052c81571ed9?, 0x3?, 0x31?, 0x4?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000588f50 sp=0xc000588f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000588fe0 sp=0xc000588f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000588fe8 sp=0xc000588fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 36 [GC worker (idle), 9 minutes]:
runtime.gopark(0x204e981bc5d4c?, 0x3?, 0x62?, 0x2a?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000589750 sp=0xc000589730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0005897e0 sp=0xc000589750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0005897e8 sp=0xc0005897e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 19 [GC worker (idle)]:
runtime.gopark(0xa10d20?, 0x1?, 0xf6?, 0x1c?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000040f50 sp=0xc000040f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000040fe0 sp=0xc000040f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000040fe8 sp=0xc000040fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 8 [GC worker (idle)]:
runtime.gopark(0x205699f226009?, 0x1?, 0x4e?, 0x46?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000047750 sp=0xc000047730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000477e0 sp=0xc000047750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000477e8 sp=0xc0000477e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

## mainSrv.run() ReadFromUDP
goroutine 50 [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000056980 sp=0xc000056960 pc=0x43e8ae
runtime.netpollblock(0x0?, 0x409326?, 0x0?)
        /usr/lib/go/src/runtime/netpoll.go:564 +0xf7 fp=0xc0000569b8 sp=0xc000056980 pc=0x437317
internal/poll.runtime_pollWait(0x7fd834e9de80, 0x72)
        /usr/lib/go/src/runtime/netpoll.go:343 +0x85 fp=0xc0000569d8 sp=0xc0000569b8 pc=0x468c85
internal/poll.(*pollDesc).wait(0xc00058c180?, 0xc000056ce0?, 0x0)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000056a00 sp=0xc0000569d8 pc=0x4d4247
internal/poll.(*pollDesc).waitRead(...)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadFromInet6(0xc00058c180, {0xc000056ce0, 0x80, 0x80}, 0x7fd834e9dec8?)
        /usr/lib/go/src/internal/poll/fd_unix.go:274 +0x22b fp=0xc000056a98 sp=0xc000056a00 pc=0x4d626b
net.(*netFD).readFromInet6(0xc00058c180, {0xc000056ce0?, 0xffffffffffffffff?, 0xffffffffffffffff?}, 0x0?)
        /usr/lib/go/src/net/fd_posix.go:72 +0x25 fp=0xc000056ae8 sp=0xc000056a98 pc=0x5623a5
net.(*UDPConn).readFrom(0x30?, {0xc000056ce0?, 0xc000e56780?, 0x0?}, 0xc000e56780)
        /usr/lib/go/src/net/udpsock_posix.go:59 +0x79 fp=0xc000056bd8 sp=0xc000056ae8 pc=0x579219
net.(*UDPConn).readFromUDP(0xc000048080, {0xc000056ce0?, 0x9e1820?, 0x9e1820?}, 0xc000abb4d0?)
        /usr/lib/go/src/net/udpsock.go:149 +0x30 fp=0xc000056c30 sp=0xc000056bd8 pc=0x577610
net.(*UDPConn).ReadFromUDP(...)
        /usr/lib/go/src/net/udpsock.go:141
main.(*mainSrv).run(0xc0000701e0, 0xc0001183c0)
        /home/ide/develop/aprilsh/frontend/server/server.go:1391 +0x4fe fp=0xc000056fb8 sp=0xc000056c30 pc=0x68e81e
main.(*mainSrv).start.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:1266 +0x2a fp=0xc000056fe0 sp=0xc000056fb8 pc=0x68dfca
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000056fe8 sp=0xc000056fe0 pc=0x46dfc1
created by main.(*mainSrv).start in goroutine 1
        /home/ide/develop/aprilsh/frontend/server/server.go:1265 +0x159

goroutine 51 [select, 81 minutes, locked to thread]:
runtime.gopark(0xc00058bfa8?, 0x2?, 0x60?, 0xbe?, 0xc00058bfa4?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc00058be38 sp=0xc00058be18 pc=0x43e8ae
runtime.selectgo(0xc00058bfa8, 0xc00058bfa0, 0x0?, 0x0, 0x0?, 0x1)
        /usr/lib/go/src/runtime/select.go:327 +0x725 fp=0xc00058bf58 sp=0xc00058be38 pc=0x44e3c5
runtime.ensureSigM.func1()
        /usr/lib/go/src/runtime/signal_unix.go:1014 +0x19f fp=0xc00058bfe0 sp=0xc00058bf58 pc=0x4655df
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc00058bfe8 sp=0xc00058bfe0 pc=0x46dfc1
created by runtime.ensureSigM in goroutine 50
        /usr/lib/go/src/runtime/signal_unix.go:997 +0xc8

goroutine 20 [syscall, 81 minutes]:
runtime.notetsleepg(0x0?, 0x0?)
        /usr/lib/go/src/runtime/lock_futex.go:236 +0x29 fp=0xc0005867a0 sp=0xc000586768 pc=0x4117c9
os/signal.signal_recv()
        /usr/lib/go/src/runtime/sigqueue.go:152 +0x29 fp=0xc0005867c0 sp=0xc0005867a0 pc=0x46a9a9
os/signal.loop()
        /usr/lib/go/src/os/signal/signal_unix.go:23 +0x13 fp=0xc0005867e0 sp=0xc0005867c0 pc=0x582693
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0005867e8 sp=0xc0005867e0 pc=0x46dfc1
created by os/signal.Notify.func1.1 in goroutine 50
        /usr/lib/go/src/os/signal/signal.go:151 +0x1f

## runWorker() shell.Wait()
goroutine 37 [syscall, 81 minutes]:
syscall.Syscall6(0x695a60?, 0xc000e4fcb0?, 0x40b64c?, 0x6e2dc0?, 0x5a5394?, 0xc000e4fcb0?, 0x40b41e?)
        /usr/lib/go/src/syscall/syscall_linux.go:91 +0x30 fp=0xc000e4fc70 sp=0xc000e4fbe8 pc=0x4bc970
os.(*Process).blockUntilWaitable(0xc000ad6060)
        /usr/lib/go/src/os/wait_waitid.go:32 +0x76 fp=0xc000e4fd48 sp=0xc000e4fc70 pc=0x4e4056
os.(*Process).wait(0xc000ad6060)
        /usr/lib/go/src/os/exec_unix.go:22 +0x25 fp=0xc000e4fda8 sp=0xc000e4fd48 pc=0x4dfd05
os.(*Process).Wait(...)
        /usr/lib/go/src/os/exec.go:134
main.runWorker(0xc0000c8640, 0xc000070240, 0x686660?)
        /home/ide/develop/aprilsh/frontend/server/server.go:648 +0xa67 fp=0xc000e4ff88 sp=0xc000e4fda8 pc=0x6893a7
main.(*mainSrv).run.func2(0x0?, 0x0?, 0xc000589f98?)
        /home/ide/develop/aprilsh/frontend/server/server.go:1434 +0x2e fp=0xc000e4ffb8 sp=0xc000e4ff88 pc=0x68faee
main.(*mainSrv).run.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:1436 +0x2f fp=0xc000e4ffe0 sp=0xc000e4ffb8 pc=0x68fa8f
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000e4ffe8 sp=0xc000e4ffe0 pc=0x46dfc1
created by main.(*mainSrv).run in goroutine 50
        /home/ide/develop/aprilsh/frontend/server/server.go:1433 +0xa12

## serve() AddUtmpx()
goroutine 38 [syscall, 75 minutes]:
runtime.cgocall(0x695ae0, 0xc000e4baf8)
        /usr/lib/go/src/runtime/cgocall.go:157 +0x4b fp=0xc000e4bad0 sp=0xc000e4ba98 pc=0x409b8b
github.com/ericwq/goutmp._Cfunc_write_uwtmp_record(0x7fd87bb94ce0, 0x7fd87bb94cd0, 0x7fd87bb94d00, 0x386, 0x1)
        _cgo_gotypes.go:171 +0x4b fp=0xc000e4baf8 sp=0xc000e4bad0 pc=0x5a3b6b
github.com/ericwq/goutmp.UtmpxAddRecord(0xc000048018, {0xc000d85c20, 0x22})
        /go/pkg/mod/github.com/ericwq/goutmp@v0.4.5/goutmp_linux.go:174 +0x145 fp=0xc000e4bb88 sp=0xc000e4baf8 pc=0x5a3e25
github.com/ericwq/aprilsh/util.AddUtmpx(...)
        /home/ide/develop/aprilsh/util/utmp_unix.go:17
main.serve(0xc000048018, 0xc000048068, 0xc0000c0770, 0xc000ce8ff0, 0x0, 0x0)
        /home/ide/develop/aprilsh/frontend/server/server.go:865 +0x1985 fp=0xc000e4bf98 sp=0xc000e4bb88 pc=0x68b2a5
main.runWorker.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:621 +0x4a fp=0xc000e4bfe0 sp=0xc000e4bf98 pc=0x6896ea
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000e4bfe8 sp=0xc000e4bfe0 pc=0x46dfc1
created by main.runWorker in goroutine 37
        /home/ide/develop/aprilsh/frontend/server/server.go:619 +0x719

## serve() ReadFromNetwork()
goroutine 54 [chan send, 74 minutes]:
runtime.gopark(0x987628?, 0x6b9ac0?, 0x0?, 0x6c?, 0x6b9c40?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc00017ce28 sp=0xc00017ce08 pc=0x43e8ae
runtime.chansend(0xc0005a8900, 0xc00017cf10, 0x1, 0xc0005d9de0?)
        /usr/lib/go/src/runtime/chan.go:259 +0x3a5 fp=0xc00017ce98 sp=0xc00017ce28 pc=0x40b205
runtime.chansend1(0xc00063e180?, 0x1?)
        /usr/lib/go/src/runtime/chan.go:145 +0x17 fp=0xc00017cec8 sp=0xc00017ce98 pc=0x40ae57
github.com/ericwq/aprilsh/frontend.ReadFromNetwork(0x0?, 0x0?, 0x0?, {0x7876e8, 0xc00063e180})
        /home/ide/develop/aprilsh/frontend/read.go:106 +0xf3 fp=0xc00017cf40 sp=0xc00017cec8 pc=0x5d8793
main.serve.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:714 +0x38 fp=0xc00017cf78 sp=0xc00017cf40 pc=0x68ca78
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc00017cfe0 sp=0xc00017cf78 pc=0x686576
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc00017cfe8 sp=0xc00017cfe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 38
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

## serve() ReadFromFile()
goroutine 55 [syscall, 81 minutes]:
syscall.Syscall(0x2ace51993?, 0x6e13c0?, 0x4d4127?, 0x7ffff800000?)
        /usr/lib/go/src/syscall/syscall_linux.go:69 +0x25 fp=0xc000041db0 sp=0xc000041d40 pc=0x4bc8e5
syscall.read(0xc0001446c0?, {0xc000c58000?, 0x1?, 0x72?})
        /usr/lib/go/src/syscall/zsyscall_linux_amd64.go:721 +0x38 fp=0xc000041df0 sp=0xc000041db0 pc=0x4ba918
syscall.Read(...)
        /usr/lib/go/src/syscall/syscall_unix.go:181
internal/poll.ignoringEINTRIO(...)
        /usr/lib/go/src/internal/poll/fd_unix.go:736
internal/poll.(*FD).Read(0xc0001446c0, {0xc000c58000, 0x4000, 0x4000})
        /usr/lib/go/src/internal/poll/fd_unix.go:160 +0x2ae fp=0xc000041e88 sp=0xc000041df0 pc=0x4d556e
os.(*File).read(...)
        /usr/lib/go/src/os/file_posix.go:29
os.(*File).Read(0xc000048018, {0xc000c58000?, 0x9e1820?, 0x9e1820?})
        /usr/lib/go/src/os/file.go:118 +0x52 fp=0xc000041ec8 sp=0xc000041e88 pc=0x4e0492
github.com/ericwq/aprilsh/frontend.ReadFromFile(0x1, 0x0?, 0x0?, {0x788268, 0xc000048018})
        /home/ide/develop/aprilsh/frontend/read.go:60 +0xb1 fp=0xc000041f40 sp=0xc000041ec8 pc=0x5d8591
main.serve.func2()
        /home/ide/develop/aprilsh/frontend/server/server.go:720 +0x35 fp=0xc000041f78 sp=0xc000041f40 pc=0x68ca15
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc000041fe0 sp=0xc000041f78 pc=0x686576
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000041fe8 sp=0xc000041fe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 38
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96
