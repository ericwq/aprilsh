goroutine 0 [idle]:
runtime.futex()
        /usr/lib/go/src/runtime/sys_linux_amd64.s:557 +0x21 fp=0x7ffec2a9e130 sp=0x7ffec2a9e128 pc=0x46fd81
runtime.futexsleep(0x7ffec2a9e1a8?, 0x442cd6?, 0x7ffec2a9e1a8?)
        /usr/lib/go/src/runtime/os_linux.go:69 +0x30 fp=0x7ffec2a9e180 sp=0x7ffec2a9e130 pc=0x437f50
runtime.notesleep(0x9e2228)
        /usr/lib/go/src/runtime/lock_futex.go:160 +0x87 fp=0x7ffec2a9e1b8 sp=0x7ffec2a9e180 pc=0x4114e7
runtime.mPark(...)
        /usr/lib/go/src/runtime/proc.go:1632
runtime.stoplockedm()
        /usr/lib/go/src/runtime/proc.go:2780 +0x73 fp=0x7ffec2a9e210 sp=0x7ffec2a9e1b8 pc=0x442eb3
runtime.schedule()
        /usr/lib/go/src/runtime/proc.go:3561 +0x3a fp=0x7ffec2a9e248 sp=0x7ffec2a9e210 pc=0x444cfa
runtime.park_m(0xc000134b60?)
        /usr/lib/go/src/runtime/proc.go:3745 +0x11f fp=0x7ffec2a9e290 sp=0x7ffec2a9e248 pc=0x44527f
traceback: unexpected SPWRITE function runtime.mcall
runtime.mcall()
        /usr/lib/go/src/runtime/asm_amd64.s:458 +0x4e fp=0x7ffec2a9e2a8 sp=0x7ffec2a9e290 pc=0x46bfce

## mainSrv.wait()
goroutine 1 [semacquire, 86 minutes]:
runtime.gopark(0x46c032?, 0xc000054e60?, 0x0?, 0x3?, 0xc000054e40?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000054e20 sp=0xc000054e00 pc=0x43e8ae
runtime.goparkunlock(...)
        /usr/lib/go/src/runtime/proc.go:404
runtime.semacquire1(0xc000070230, 0x1?, 0x1, 0x0, 0xa0?)
        /usr/lib/go/src/runtime/sema.go:160 +0x218 fp=0xc000054e88 sp=0xc000054e20 pc=0x44f3f8
sync.runtime_Semacquire(0x722383?)
        /usr/lib/go/src/runtime/sema.go:62 +0x25 fp=0xc000054ec0 sp=0xc000054e88 pc=0x46a665
sync.(*WaitGroup).Wait(0xc0000701e0?)
        /usr/lib/go/src/sync/waitgroup.go:116 +0x48 fp=0xc000054ee8 sp=0xc000054ec0 pc=0x4893a8
main.(*mainSrv).wait(...)
        /home/ide/develop/aprilsh/frontend/server/server.go:1539
main.main()
        /home/ide/develop/aprilsh/frontend/server/server.go:449 +0x24c fp=0xc000054f40 sp=0xc000054ee8 pc=0x68834c
runtime.main()
        /usr/lib/go/src/runtime/proc.go:267 +0x2bb fp=0xc000054fe0 sp=0xc000054f40 pc=0x43e45b
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000054fe8 sp=0xc000054fe0 pc=0x46dfc1

goroutine 2 [force gc (idle), 73 minutes]:
runtime.gopark(0x1a0d96cbb3e69?, 0x0?, 0x0?, 0x0?, 0x0?)
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
runtime.gopark(0xa2171?, 0x89a0f?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000045f70 sp=0xc000045f50 pc=0x43e8ae
runtime.goparkunlock(...)
        /usr/lib/go/src/runtime/proc.go:404
runtime.(*scavengerState).park(0x9e1880)
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
runtime.gopark(0x0?, 0x736038?, 0x60?, 0xa0?, 0x2000000020?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000044620 sp=0xc000044600 pc=0x43e8ae
runtime.runfinq()
        /usr/lib/go/src/runtime/mfinal.go:193 +0x107 fp=0xc0000447e0 sp=0xc000044620 pc=0x41e967
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000447e8 sp=0xc0000447e0 pc=0x46dfc1
created by runtime.createfing in goroutine 1
        /usr/lib/go/src/runtime/mfinal.go:163 +0x3d

goroutine 6 [GC worker (idle), 86 minutes]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000046750 sp=0xc000046730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000467e0 sp=0xc000046750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000467e8 sp=0xc0000467e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 18 [GC worker (idle), 86 minutes]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000040750 sp=0xc000040730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000407e0 sp=0xc000040750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000407e8 sp=0xc0000407e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 34 [GC worker (idle)]:
runtime.gopark(0x1a105fe12e232?, 0x1?, 0xc?, 0xe3?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000588750 sp=0xc000588730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0005887e0 sp=0xc000588750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0005887e8 sp=0xc0005887e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 7 [GC worker (idle)]:
runtime.gopark(0x1a1072ae38929?, 0x3?, 0x8a?, 0x4e?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000046f50 sp=0xc000046f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000046fe0 sp=0xc000046f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000046fe8 sp=0xc000046fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 19 [GC worker (idle)]:
runtime.gopark(0x1a105fe12dee7?, 0x3?, 0x25?, 0xbc?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000040f50 sp=0xc000040f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000040fe0 sp=0xc000040f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000040fe8 sp=0xc000040fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 8 [GC worker (idle)]:
runtime.gopark(0x1a1072aefe3c3?, 0x1?, 0x33?, 0x7d?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000047750 sp=0xc000047730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000477e0 sp=0xc000047750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000477e8 sp=0xc0000477e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 20 [GC worker (idle)]:
runtime.gopark(0x1a1072ae41897?, 0x3?, 0x4?, 0x21?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000041750 sp=0xc000041730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000417e0 sp=0xc000041750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000417e8 sp=0xc0000417e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 35 [GC worker (idle)]:
runtime.gopark(0x1a1072af51af2?, 0x1?, 0x48?, 0x25?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000588f50 sp=0xc000588f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000588fe0 sp=0xc000588f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000588fe8 sp=0xc000588fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

## run() ReadFromUDP()
goroutine 36 [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000059980 sp=0xc000059960 pc=0x43e8ae
runtime.netpollblock(0x0?, 0x409326?, 0x0?)
        /usr/lib/go/src/runtime/netpoll.go:564 +0xf7 fp=0xc0000599b8 sp=0xc000059980 pc=0x437317
internal/poll.runtime_pollWait(0x7f09b90fde80, 0x72)
        /usr/lib/go/src/runtime/netpoll.go:343 +0x85 fp=0xc0000599d8 sp=0xc0000599b8 pc=0x468c85
internal/poll.(*pollDesc).wait(0xc00058c100?, 0xc000059ce0?, 0x0)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000059a00 sp=0xc0000599d8 pc=0x4d4247
internal/poll.(*pollDesc).waitRead(...)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadFromInet6(0xc00058c100, {0xc000059ce0, 0x80, 0x80}, 0x7f09b90fdec8?)
        /usr/lib/go/src/internal/poll/fd_unix.go:274 +0x22b fp=0xc000059a98 sp=0xc000059a00 pc=0x4d626b
net.(*netFD).readFromInet6(0xc00058c100, {0xc000059ce0?, 0xffffffffffffffff?, 0xffffffffffffffff?}, 0x0?)
        /usr/lib/go/src/net/fd_posix.go:72 +0x25 fp=0xc000059ae8 sp=0xc000059a98 pc=0x5623a5
net.(*UDPConn).readFrom(0x30?, {0xc000059ce0?, 0xc000098810?, 0x0?}, 0xc000098810)
        /usr/lib/go/src/net/udpsock_posix.go:59 +0x79 fp=0xc000059bd8 sp=0xc000059ae8 pc=0x579219
net.(*UDPConn).readFromUDP(0xc000048070, {0xc000059ce0?, 0x9e1800?, 0x9e1800?}, 0x3?)
        /usr/lib/go/src/net/udpsock.go:149 +0x30 fp=0xc000059c30 sp=0xc000059bd8 pc=0x577610
net.(*UDPConn).ReadFromUDP(...)
        /usr/lib/go/src/net/udpsock.go:141
main.(*mainSrv).run(0xc0000701e0, 0xc0001183c0)
        /home/ide/develop/aprilsh/frontend/server/server.go:1391 +0x4fe fp=0xc000059fb8 sp=0xc000059c30 pc=0x68e81e
main.(*mainSrv).start.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:1266 +0x2a fp=0xc000059fe0 sp=0xc000059fb8 pc=0x68dfca
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000059fe8 sp=0xc000059fe0 pc=0x46dfc1
created by main.(*mainSrv).start in goroutine 1
        /home/ide/develop/aprilsh/frontend/server/server.go:1265 +0x159

goroutine 37 [select, 86 minutes, locked to thread]:
runtime.gopark(0xc00058b7a8?, 0x2?, 0x60?, 0xb6?, 0xc00058b7a4?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc00058b638 sp=0xc00058b618 pc=0x43e8ae
runtime.selectgo(0xc00058b7a8, 0xc00058b7a0, 0x0?, 0x0, 0x0?, 0x1)
        /usr/lib/go/src/runtime/select.go:327 +0x725 fp=0xc00058b758 sp=0xc00058b638 pc=0x44e3c5
runtime.ensureSigM.func1()
        /usr/lib/go/src/runtime/signal_unix.go:1014 +0x19f fp=0xc00058b7e0 sp=0xc00058b758 pc=0x4655df
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc00058b7e8 sp=0xc00058b7e0 pc=0x46dfc1
created by runtime.ensureSigM in goroutine 36
        /usr/lib/go/src/runtime/signal_unix.go:997 +0xc8

goroutine 21 [syscall, 86 minutes]:
runtime.notetsleepg(0x0?, 0x0?)
        /usr/lib/go/src/runtime/lock_futex.go:236 +0x29 fp=0xc0005867a0 sp=0xc000586768 pc=0x4117c9
os/signal.signal_recv()
        /usr/lib/go/src/runtime/sigqueue.go:152 +0x29 fp=0xc0005867c0 sp=0xc0005867a0 pc=0x46a9a9
os/signal.loop()
        /usr/lib/go/src/os/signal/signal_unix.go:23 +0x13 fp=0xc0005867e0 sp=0xc0005867c0 pc=0x582693
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0005867e8 sp=0xc0005867e0 pc=0x46dfc1
created by os/signal.Notify.func1.1 in goroutine 36
        /usr/lib/go/src/os/signal/signal.go:151 +0x1f

## shell.Wait()
goroutine 40 [syscall, 85 minutes]:
syscall.Syscall6(0x695a60?, 0xc00081bcb0?, 0x40b64c?, 0x6e2dc0?, 0x5a5394?, 0xc00081bcb0?, 0x40b41e?)
        /usr/lib/go/src/syscall/syscall_linux.go:91 +0x30 fp=0xc00081bc70 sp=0xc00081bbe8 pc=0x4bc970
os.(*Process).blockUntilWaitable(0xc0000fc1b0)
        /usr/lib/go/src/os/wait_waitid.go:32 +0x76 fp=0xc00081bd48 sp=0xc00081bc70 pc=0x4e4056
os.(*Process).wait(0xc0000fc1b0)
        /usr/lib/go/src/os/exec_unix.go:22 +0x25 fp=0xc00081bda8 sp=0xc00081bd48 pc=0x4dfd05
os.(*Process).Wait(...)
        /usr/lib/go/src/os/exec.go:134
main.runWorker(0xc00089c500, 0xc000070240, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:648 +0xa67 fp=0xc00081bf88 sp=0xc00081bda8 pc=0x6893a7
main.(*mainSrv).run.func2(0x0?, 0x0?, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:1434 +0x2e fp=0xc00081bfb8 sp=0xc00081bf88 pc=0x68faee
main.(*mainSrv).run.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:1436 +0x2f fp=0xc00081bfe0 sp=0xc00081bfb8 pc=0x68fa8f
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc00081bfe8 sp=0xc00081bfe0 pc=0x46dfc1
created by main.(*mainSrv).run in goroutine 36
        /home/ide/develop/aprilsh/frontend/server/server.go:1433 +0xa12

## ClearUtmpx()
goroutine 41 [syscall, 1 minutes]:
runtime.cgocall(0x695ae0, 0xc000815b20)
        /usr/lib/go/src/runtime/cgocall.go:157 +0x4b fp=0xc000815af8 sp=0xc000815ac0 pc=0x409b8b
github.com/ericwq/goutmp._Cfunc_write_uwtmp_record(0x0, 0x7f09ffdf4c60, 0x0, 0x8391, 0x0)
        _cgo_gotypes.go:171 +0x4b fp=0xc000815b20 sp=0xc000815af8 pc=0x5a3b6b
github.com/ericwq/goutmp.UtmpxRemoveRecord(0x6fd760?)
        /home/ide/develop/goutmp/goutmp_linux.go:191 +0xba fp=0xc000815b88 sp=0xc000815b20 pc=0x5a40da
github.com/ericwq/aprilsh/util.ClearUtmpx(...)
        /home/ide/develop/aprilsh/util/utmp_unix.go:21
main.serve(0xc0000a2020, 0xc0000a2030, 0xc0008943f0, 0xc00007aa00, 0x0, 0x0)
        /home/ide/develop/aprilsh/frontend/server/server.go:854 +0x17ec fp=0xc000815f98 sp=0xc000815b88 pc=0x68b10c
main.runWorker.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:621 +0x4a fp=0xc000815fe0 sp=0xc000815f98 pc=0x6896ea
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000815fe8 sp=0xc000815fe0 pc=0x46dfc1
created by main.runWorker in goroutine 40
        /home/ide/develop/aprilsh/frontend/server/server.go:619 +0x719

## ReadFromNetwork()
goroutine 52 [chan send, 1 minutes]:
runtime.gopark(0x987608?, 0x6b9ac0?, 0x0?, 0x6c?, 0x6b9c40?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000057e28 sp=0xc000057e08 pc=0x43e8ae
runtime.chansend(0xc000070540, 0xc000057f10, 0x1, 0xc00078bde0?)
        /usr/lib/go/src/runtime/chan.go:259 +0x3a5 fp=0xc000057e98 sp=0xc000057e28 pc=0x40b205
runtime.chansend1(0xc0005c00c0?, 0x1?)
        /usr/lib/go/src/runtime/chan.go:145 +0x17 fp=0xc000057ec8 sp=0xc000057e98 pc=0x40ae57
github.com/ericwq/aprilsh/frontend.ReadFromNetwork(0x0?, 0x0?, 0x0?, {0x7876e8, 0xc0005c00c0})
        /home/ide/develop/aprilsh/frontend/read.go:106 +0xf3 fp=0xc000057f40 sp=0xc000057ec8 pc=0x5d8793
main.serve.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:714 +0x38 fp=0xc000057f78 sp=0xc000057f40 pc=0x68ca78
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc000057fe0 sp=0xc000057f78 pc=0x686576
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000057fe8 sp=0xc000057fe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 41
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

## ReadFromFile()
goroutine 53 [syscall, 85 minutes]:
syscall.Syscall(0xdb59a5e0d?, 0x6e13c0?, 0x4d4127?, 0x7ffff800000?)
        /usr/lib/go/src/syscall/syscall_linux.go:69 +0x25 fp=0xc0000425b0 sp=0xc000042540 pc=0x4bc8e5
syscall.read(0xc000828120?, {0xc0008ee000?, 0x1?, 0x72?})
        /usr/lib/go/src/syscall/zsyscall_linux_amd64.go:721 +0x38 fp=0xc0000425f0 sp=0xc0000425b0 pc=0x4ba918
syscall.Read(...)
        /usr/lib/go/src/syscall/syscall_unix.go:181
internal/poll.ignoringEINTRIO(...)
        /usr/lib/go/src/internal/poll/fd_unix.go:736
internal/poll.(*FD).Read(0xc000828120, {0xc0008ee000, 0x4000, 0x4000})
        /usr/lib/go/src/internal/poll/fd_unix.go:160 +0x2ae fp=0xc000042688 sp=0xc0000425f0 pc=0x4d556e
os.(*File).read(...)
        /usr/lib/go/src/os/file_posix.go:29
os.(*File).Read(0xc0000a2020, {0xc0008ee000?, 0x9e1800?, 0x9e1800?})
        /usr/lib/go/src/os/file.go:118 +0x52 fp=0xc0000426c8 sp=0xc000042688 pc=0x4e0492
github.com/ericwq/aprilsh/frontend.ReadFromFile(0x1, 0x0?, 0x0?, {0x788268, 0xc0000a2020})
        /home/ide/develop/aprilsh/frontend/read.go:60 +0xb1 fp=0xc000042740 sp=0xc0000426c8 pc=0x5d8591
main.serve.func2()
        /home/ide/develop/aprilsh/frontend/server/server.go:720 +0x35 fp=0xc000042778 sp=0xc000042740 pc=0x68ca15
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc0000427e0 sp=0xc000042778 pc=0x686576
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000427e8 sp=0xc0000427e0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 41
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96
