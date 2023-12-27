goroutine 0 [idle]:
runtime.futex()
        /usr/lib/go/src/runtime/sys_linux_amd64.s:557 +0x21 fp=0x7ffde7b24be0 sp=0x7ffde7b24bd8 pc=0x46fd81
runtime.futexsleep(0x7ffde7b24c58?, 0x442cd6?, 0x7ffde7b24c58?)
        /usr/lib/go/src/runtime/os_linux.go:69 +0x30 fp=0x7ffde7b24c30 sp=0x7ffde7b24be0 pc=0x437f50
runtime.notesleep(0x9e2208)
        /usr/lib/go/src/runtime/lock_futex.go:160 +0x87 fp=0x7ffde7b24c68 sp=0x7ffde7b24c30 pc=0x4114e7
runtime.mPark(...)
        /usr/lib/go/src/runtime/proc.go:1632
runtime.stoplockedm()
        /usr/lib/go/src/runtime/proc.go:2780 +0x73 fp=0x7ffde7b24cc0 sp=0x7ffde7b24c68 pc=0x442eb3
runtime.schedule()
        /usr/lib/go/src/runtime/proc.go:3561 +0x3a fp=0x7ffde7b24cf8 sp=0x7ffde7b24cc0 pc=0x444cfa
runtime.park_m(0xc000107520?)
        /usr/lib/go/src/runtime/proc.go:3745 +0x11f fp=0x7ffde7b24d40 sp=0x7ffde7b24cf8 pc=0x44527f
traceback: unexpected SPWRITE function runtime.mcall
runtime.mcall()
        /usr/lib/go/src/runtime/asm_amd64.s:458 +0x4e fp=0x7ffde7b24d58 sp=0x7ffde7b24d40 pc=0x46bfce

## mainSrv.wait()
goroutine 1 [semacquire, 51 minutes]:
runtime.gopark(0x46c032?, 0xc00009ee60?, 0x0?, 0x0?, 0xc00009ee40?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc00009ee20 sp=0xc00009ee00 pc=0x43e8ae
runtime.goparkunlock(...)
        /usr/lib/go/src/runtime/proc.go:404
runtime.semacquire1(0xc00012e1d0, 0x1?, 0x1, 0x0, 0x40?)
        /usr/lib/go/src/runtime/sema.go:160 +0x218 fp=0xc00009ee88 sp=0xc00009ee20 pc=0x44f3f8
sync.runtime_Semacquire(0x722383?)
        /usr/lib/go/src/runtime/sema.go:62 +0x25 fp=0xc00009eec0 sp=0xc00009ee88 pc=0x46a665
sync.(*WaitGroup).Wait(0xc00012e180?)
        /usr/lib/go/src/sync/waitgroup.go:116 +0x48 fp=0xc00009eee8 sp=0xc00009eec0 pc=0x4893a8
main.(*mainSrv).wait(...)
        /home/ide/develop/aprilsh/frontend/server/server.go:1536
main.main()
        /home/ide/develop/aprilsh/frontend/server/server.go:449 +0x24c fp=0xc00009ef40 sp=0xc00009eee8 pc=0x68832c
runtime.main()
        /usr/lib/go/src/runtime/proc.go:267 +0x2bb fp=0xc00009efe0 sp=0xc00009ef40 pc=0x43e45b
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc00009efe8 sp=0xc00009efe0 pc=0x46dfc1

goroutine 2 [force gc (idle), 50 minutes]:
runtime.gopark(0x18fa9167e2710?, 0x0?, 0x0?, 0x0?, 0x0?)
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
runtime.gopark(0x18be5d?, 0xef158?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000045f70 sp=0xc000045f50 pc=0x43e8ae
runtime.goparkunlock(...)
        /usr/lib/go/src/runtime/proc.go:404
runtime.(*scavengerState).park(0x9e1860)
        /usr/lib/go/src/runtime/mgcscavenge.go:425 +0x49 fp=0xc000045fa0 sp=0xc000045f70 pc=0x428069
runtime.bgscavenge(0x0?)
        /usr/lib/go/src/runtime/mgcscavenge.go:658 +0x59 fp=0xc000045fc8 sp=0xc000045fa0 pc=0x428619
runtime.gcenable.func2()
        /usr/lib/go/src/runtime/mgc.go:201 +0x25 fp=0xc000045fe0 sp=0xc000045fc8 pc=0x41f8e5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000045fe8 sp=0xc000045fe0 pc=0x46dfc1
created by runtime.gcenable in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:201 +0xa5

goroutine 18 [finalizer wait, 51 minutes]:
runtime.gopark(0x198?, 0x71e5a0?, 0x1?, 0xfa?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000044620 sp=0xc000044600 pc=0x43e8ae
runtime.runfinq()
        /usr/lib/go/src/runtime/mfinal.go:193 +0x107 fp=0xc0000447e0 sp=0xc000044620 pc=0x41e967
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000447e8 sp=0xc0000447e0 pc=0x46dfc1
created by runtime.createfing in goroutine 1
        /usr/lib/go/src/runtime/mfinal.go:163 +0x3d

goroutine 19 [GC worker (idle), 51 minutes]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000040750 sp=0xc000040730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000407e0 sp=0xc000040750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000407e8 sp=0xc0000407e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 5 [GC worker (idle), 51 minutes]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000046750 sp=0xc000046730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000467e0 sp=0xc000046750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000467e8 sp=0xc0000467e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 34 [GC worker (idle)]:
runtime.gopark(0x18fb0d252a149?, 0x3?, 0x9f?, 0x92?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000094750 sp=0xc000094730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000947e0 sp=0xc000094750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000947e8 sp=0xc0000947e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 20 [GC worker (idle)]:
runtime.gopark(0x18fb0d252b6d9?, 0x3?, 0xa1?, 0x49?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000040f50 sp=0xc000040f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000040fe0 sp=0xc000040f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000040fe8 sp=0xc000040fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 6 [GC worker (idle)]:
runtime.gopark(0x18fb0d2565940?, 0x1?, 0x48?, 0x46?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000046f50 sp=0xc000046f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000046fe0 sp=0xc000046f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000046fe8 sp=0xc000046fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 21 [GC worker (idle)]:
runtime.gopark(0xa10ce0?, 0x1?, 0x42?, 0xcf?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000041750 sp=0xc000041730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000417e0 sp=0xc000041750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000417e8 sp=0xc0000417e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 22 [GC worker (idle)]:
runtime.gopark(0x18fad6b190541?, 0x1?, 0x94?, 0x39?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000041f50 sp=0xc000041f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000041fe0 sp=0xc000041f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000041fe8 sp=0xc000041fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 7 [GC worker (idle)]:
runtime.gopark(0x18fb0d253b97d?, 0x3?, 0x5d?, 0x7?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000047750 sp=0xc000047730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000477e0 sp=0xc000047750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000477e8 sp=0xc0000477e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

## mainSrv.run() read from UDP 
goroutine 23 [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc0000a1980 sp=0xc0000a1960 pc=0x43e8ae
runtime.netpollblock(0x0?, 0x409326?, 0x0?)
        /usr/lib/go/src/runtime/netpoll.go:564 +0xf7 fp=0xc0000a19b8 sp=0xc0000a1980 pc=0x437317
internal/poll.runtime_pollWait(0x7f3e13e67e28, 0x72)
        /usr/lib/go/src/runtime/netpoll.go:343 +0x85 fp=0xc0000a19d8 sp=0xc0000a19b8 pc=0x468c85
internal/poll.(*pollDesc).wait(0xc000098180?, 0xc0000a1ce0?, 0x0)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc0000a1a00 sp=0xc0000a19d8 pc=0x4d4247
internal/poll.(*pollDesc).waitRead(...)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadFromInet6(0xc000098180, {0xc0000a1ce0, 0x80, 0x80}, 0x7f3e13e67e70?)
        /usr/lib/go/src/internal/poll/fd_unix.go:274 +0x22b fp=0xc0000a1a98 sp=0xc0000a1a00 pc=0x4d626b
net.(*netFD).readFromInet6(0xc000098180, {0xc0000a1ce0?, 0xffffffffffffffff?, 0xffffffffffffffff?}, 0x0?)
        /usr/lib/go/src/net/fd_posix.go:72 +0x25 fp=0xc0000a1ae8 sp=0xc0000a1a98 pc=0x5623a5
net.(*UDPConn).readFrom(0x30?, {0xc0000a1ce0?, 0xc001005f80?, 0x0?}, 0xc001005f80)
        /usr/lib/go/src/net/udpsock_posix.go:59 +0x79 fp=0xc0000a1bd8 sp=0xc0000a1ae8 pc=0x579219
net.(*UDPConn).readFromUDP(0xc000126080, {0xc0000a1ce0?, 0x9e17e0?, 0x9e17e0?}, 0xc00013c140?)
        /usr/lib/go/src/net/udpsock.go:149 +0x30 fp=0xc0000a1c30 sp=0xc0000a1bd8 pc=0x577610
net.(*UDPConn).ReadFromUDP(...)
        /usr/lib/go/src/net/udpsock.go:141
main.(*mainSrv).run(0xc00012e180, 0xc00012a3c0)
        /home/ide/develop/aprilsh/frontend/server/server.go:1388 +0x4fe fp=0xc0000a1fb8 sp=0xc0000a1c30 pc=0x68e7be
main.(*mainSrv).start.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:1263 +0x2a fp=0xc0000a1fe0 sp=0xc0000a1fb8 pc=0x68df6a
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000a1fe8 sp=0xc0000a1fe0 pc=0x46dfc1
created by main.(*mainSrv).start in goroutine 1
        /home/ide/develop/aprilsh/frontend/server/server.go:1262 +0x159

goroutine 24 [select, 51 minutes, locked to thread]:
runtime.gopark(0xc000096fa8?, 0x2?, 0x60?, 0x6e?, 0xc000096fa4?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000096e38 sp=0xc000096e18 pc=0x43e8ae
runtime.selectgo(0xc000096fa8, 0xc000096fa0, 0x0?, 0x0, 0x0?, 0x1)
        /usr/lib/go/src/runtime/select.go:327 +0x725 fp=0xc000096f58 sp=0xc000096e38 pc=0x44e3c5
runtime.ensureSigM.func1()
        /usr/lib/go/src/runtime/signal_unix.go:1014 +0x19f fp=0xc000096fe0 sp=0xc000096f58 pc=0x4655df
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000096fe8 sp=0xc000096fe0 pc=0x46dfc1
created by runtime.ensureSigM in goroutine 23
        /usr/lib/go/src/runtime/signal_unix.go:997 +0xc8

goroutine 8 [syscall, 51 minutes]:
runtime.notetsleepg(0x0?, 0x0?)
        /usr/lib/go/src/runtime/lock_futex.go:236 +0x29 fp=0xc0000927a0 sp=0xc000092768 pc=0x4117c9
os/signal.signal_recv()
        /usr/lib/go/src/runtime/sigqueue.go:152 +0x29 fp=0xc0000927c0 sp=0xc0000927a0 pc=0x46a9a9
os/signal.loop()
        /usr/lib/go/src/os/signal/signal_unix.go:23 +0x13 fp=0xc0000927e0 sp=0xc0000927c0 pc=0x582693
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000927e8 sp=0xc0000927e0 pc=0x46dfc1
created by os/signal.Notify.func1.1 in goroutine 23
        /usr/lib/go/src/os/signal/signal.go:151 +0x1f

## runWorker() shell.Wait()
goroutine 9 [syscall, 51 minutes]:
syscall.Syscall6(0x695a00?, 0xc0000a3cb0?, 0x40b64c?, 0x6e2dc0?, 0x5a5394?, 0xc0000a3cb0?, 0x40b41e?)
        /usr/lib/go/src/syscall/syscall_linux.go:91 +0x30 fp=0xc0000a3c70 sp=0xc0000a3be8 pc=0x4bc970
os.(*Process).blockUntilWaitable(0xc00001e210)
        /usr/lib/go/src/os/wait_waitid.go:32 +0x76 fp=0xc0000a3d48 sp=0xc0000a3c70 pc=0x4e4056
os.(*Process).wait(0xc00001e210)
        /usr/lib/go/src/os/exec_unix.go:22 +0x25 fp=0xc0000a3da8 sp=0xc0000a3d48 pc=0x4dfd05
os.(*Process).Wait(...)
        /usr/lib/go/src/os/exec.go:134
main.runWorker(0xc00017e000, 0xc00012e1e0, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:648 +0xa67 fp=0xc0000a3f88 sp=0xc0000a3da8 pc=0x689387
main.(*mainSrv).run.func2(0x0?, 0x0?, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:1431 +0x2e fp=0xc0000a3fb8 sp=0xc0000a3f88 pc=0x68fa8e
main.(*mainSrv).run.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:1433 +0x2f fp=0xc0000a3fe0 sp=0xc0000a3fb8 pc=0x68fa2f
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000a3fe8 sp=0xc0000a3fe0 pc=0x46dfc1
created by main.(*mainSrv).run in goroutine 23
        /home/ide/develop/aprilsh/frontend/server/server.go:1430 +0xa12

## serve() util.ClearUtmpx().
goroutine 10 [syscall]:
runtime.cgocall(0x695a80, 0xc0001b5b20)
        /usr/lib/go/src/runtime/cgocall.go:157 +0x4b fp=0xc0001b5af8 sp=0xc0001b5ac0 pc=0x409b8b
github.com/ericwq/goutmp._Cfunc_write_uwtmp_record(0x0, 0x7f3e5ab37c60, 0x0, 0x7c78, 0x0)
        _cgo_gotypes.go:171 +0x4b fp=0xc0001b5b20 sp=0xc0001b5af8 pc=0x5a3b6b
github.com/ericwq/goutmp.UtmpxRemoveRecord(0x6fd760?)
        /home/ide/develop/goutmp/goutmp_linux.go:191 +0xba fp=0xc0001b5b88 sp=0xc0001b5b20 pc=0x5a40da
github.com/ericwq/aprilsh/util.ClearUtmpx(...)
        /home/ide/develop/aprilsh/util/utmp_unix.go:21
main.serve(0xc00007e018, 0xc00007e028, 0xc000190000, 0xc00016c280, 0x0, 0x0)
        /home/ide/develop/aprilsh/frontend/server/server.go:979 +0x2234 fp=0xc0001b5f98 sp=0xc0001b5b88 pc=0x68bb34
main.runWorker.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:621 +0x4a fp=0xc0001b5fe0 sp=0xc0001b5f98 pc=0x6896ca
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0001b5fe8 sp=0xc0001b5fe0 pc=0x46dfc1
created by main.runWorker in goroutine 9
        /home/ide/develop/aprilsh/frontend/server/server.go:619 +0x719

## serve() ReadFromNetwork() chan send
goroutine 35 [chan send]:
runtime.gopark(0x9875e8?, 0x6b9ac0?, 0xc0?, 0x6b?, 0x6b9c40?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc00009ae28 sp=0xc00009ae08 pc=0x43e8ae
runtime.chansend(0xc0000b2060, 0xc00009af10, 0x1, 0xc000fffde0?)
        /usr/lib/go/src/runtime/chan.go:259 +0x3a5 fp=0xc00009ae98 sp=0xc00009ae28 pc=0x40b205
runtime.chansend1(0xc000618000?, 0x1?)
        /usr/lib/go/src/runtime/chan.go:145 +0x17 fp=0xc00009aec8 sp=0xc00009ae98 pc=0x40ae57
github.com/ericwq/aprilsh/frontend.ReadFromNetwork(0x0?, 0x0?, 0x0?, {0x7876a8, 0xc000618000})
        /home/ide/develop/aprilsh/frontend/read.go:106 +0xf3 fp=0xc00009af40 sp=0xc00009aec8 pc=0x5d8793
main.serve.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:714 +0x38 fp=0xc00009af78 sp=0xc00009af40 pc=0x68ca18
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc00009afe0 sp=0xc00009af78 pc=0x686556
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc00009afe8 sp=0xc00009afe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 10
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

## serve() ReadFromFile()
goroutine 36 [syscall, 51 minutes]:
syscall.Syscall(0x3105dae58?, 0x6e13c0?, 0x4d4127?, 0x7ffff800000?)
        /usr/lib/go/src/syscall/syscall_linux.go:69 +0x25 fp=0xc00009cdb0 sp=0xc00009cd40 pc=0x4bc8e5
syscall.read(0xc000070120?, {0xc0007ba000?, 0x43e801?, 0x72?})
        /usr/lib/go/src/syscall/zsyscall_linux_amd64.go:721 +0x38 fp=0xc00009cdf0 sp=0xc00009cdb0 pc=0x4ba918
syscall.Read(...)
        /usr/lib/go/src/syscall/syscall_unix.go:181
internal/poll.ignoringEINTRIO(...)
        /usr/lib/go/src/internal/poll/fd_unix.go:736
internal/poll.(*FD).Read(0xc000070120, {0xc0007ba000, 0x4000, 0x4000})
        /usr/lib/go/src/internal/poll/fd_unix.go:160 +0x2ae fp=0xc00009ce88 sp=0xc00009cdf0 pc=0x4d556e
os.(*File).read(...)
        /usr/lib/go/src/os/file_posix.go:29
os.(*File).Read(0xc00007e018, {0xc0007ba000?, 0x9e17e0?, 0x9e17e0?})
        /usr/lib/go/src/os/file.go:118 +0x52 fp=0xc00009cec8 sp=0xc00009ce88 pc=0x4e0492
github.com/ericwq/aprilsh/frontend.ReadFromFile(0x1, 0x0?, 0x0?, {0x788228, 0xc00007e018})
        /home/ide/develop/aprilsh/frontend/read.go:60 +0xb1 fp=0xc00009cf40 sp=0xc00009cec8 pc=0x5d8591
main.serve.func2()
        /home/ide/develop/aprilsh/frontend/server/server.go:720 +0x35 fp=0xc00009cf78 sp=0xc00009cf40 pc=0x68c9b5
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc00009cfe0 sp=0xc00009cf78 pc=0x686556
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc00009cfe8 sp=0xc00009cfe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 10
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

## runWorker() shell.Wait()
goroutine 25 [syscall, 51 minutes]:
syscall.Syscall6(0x695a00?, 0xc0000a9cb0?, 0x40b64c?, 0x6e2dc0?, 0x5a5394?, 0xc0000a9cb0?, 0x40b41e?)
        /usr/lib/go/src/syscall/syscall_linux.go:91 +0x30 fp=0xc0000a9c70 sp=0xc0000a9be8 pc=0x4bc970
os.(*Process).blockUntilWaitable(0xc0009d01b0)
        /usr/lib/go/src/os/wait_waitid.go:32 +0x76 fp=0xc0000a9d48 sp=0xc0000a9c70 pc=0x4e4056
os.(*Process).wait(0xc0009d01b0)
        /usr/lib/go/src/os/exec_unix.go:22 +0x25 fp=0xc0000a9da8 sp=0xc0000a9d48 pc=0x4dfd05
os.(*Process).Wait(...)
        /usr/lib/go/src/os/exec.go:134
main.runWorker(0xc00012a6e0, 0xc00012e1e0, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:648 +0xa67 fp=0xc0000a9f88 sp=0xc0000a9da8 pc=0x689387
main.(*mainSrv).run.func2(0x0?, 0x0?, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:1431 +0x2e fp=0xc0000a9fb8 sp=0xc0000a9f88 pc=0x68fa8e
main.(*mainSrv).run.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:1433 +0x2f fp=0xc0000a9fe0 sp=0xc0000a9fb8 pc=0x68fa2f
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000a9fe8 sp=0xc0000a9fe0 pc=0x46dfc1
created by main.(*mainSrv).run in goroutine 23
        /home/ide/develop/aprilsh/frontend/server/server.go:1430 +0xa12

## serve() select
goroutine 26 [select]:
runtime.gopark(0xc0001b7f38?, 0x5?, 0x90?, 0x7a?, 0xc0001b7cb6?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc0001b7a68 sp=0xc0001b7a48 pc=0x43e8ae
runtime.selectgo(0xc0001b7f38, 0xc0001b7cac, 0xa0fdc0?, 0x0, 0x723426?, 0x1)
        /usr/lib/go/src/runtime/select.go:327 +0x725 fp=0xc0001b7b88 sp=0xc0001b7a68 pc=0x44e3c5
main.serve(0xc000126120, 0xc000126130, 0xc000190070, 0xc000b5c960, 0x0, 0x0)
        /home/ide/develop/aprilsh/frontend/server/server.go:769 +0x9df fp=0xc0001b7f98 sp=0xc0001b7b88 pc=0x68a2df
main.runWorker.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:621 +0x4a fp=0xc0001b7fe0 sp=0xc0001b7f98 pc=0x6896ca
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0001b7fe8 sp=0xc0001b7fe0 pc=0x46dfc1
created by main.runWorker in goroutine 25
        /home/ide/develop/aprilsh/frontend/server/server.go:619 +0x719

## serve() ReadFromNetwork()
goroutine 11 [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000fff800 sp=0xc000fff7e0 pc=0x43e8ae
runtime.netpollblock(0x0?, 0x409326?, 0x0?)
        /usr/lib/go/src/runtime/netpoll.go:564 +0xf7 fp=0xc000fff838 sp=0xc000fff800 pc=0x437317
internal/poll.runtime_pollWait(0x7f3e13e67b40, 0x72)
        /usr/lib/go/src/runtime/netpoll.go:343 +0x85 fp=0xc000fff858 sp=0xc000fff838 pc=0x468c85
internal/poll.(*pollDesc).wait(0xc00061c200?, 0xc000654f00?, 0x0)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000fff880 sp=0xc000fff858 pc=0x4d4247
internal/poll.(*pollDesc).waitRead(...)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadMsgInet6(0xc00061c200, {0xc000654f00, 0x4e4, 0x4e4}, {0xc00001f260, 0x28, 0x28}, 0x7f3e13daae18?, 0x0?)
        /usr/lib/go/src/internal/poll/fd_unix.go:355 +0x339 fp=0xc000fff960 sp=0xc000fff880 pc=0x4d7179
net.(*netFD).readMsgInet6(0xc00061c200, {0xc000654f00?, 0xc00011c048?, 0x0?}, {0xc00001f260?, 0x45?, 0xc000fffa50?}, 0x41c2e8?, 0x43628a?)
        /usr/lib/go/src/net/fd_posix.go:90 +0x31 fp=0xc000fff9e0 sp=0xc000fff960 pc=0x562771
net.(*UDPConn).readMsg(0x4130a5?, {0xc000654f00?, 0x6ecc40?, 0xc000fffb40?}, {0xc00001f260?, 0xc000098700?, 0x1?})
        /usr/lib/go/src/net/udpsock_posix.go:106 +0x9c fp=0xc000fffad0 sp=0xc000fff9e0 pc=0x5796fc
net.(*UDPConn).ReadMsgUDPAddrPort(0xc000126118, {0xc000654f00?, 0x7f3e5aac0108?, 0x30?}, {0xc00001f260?, 0xc00001f260?, 0x0?})
        /usr/lib/go/src/net/udpsock.go:203 +0x3e fp=0xc000fffb60 sp=0xc000fffad0 pc=0x577ade
net.(*UDPConn).ReadMsgUDP(0xc000fffbe8?, {0xc000654f00?, 0xffffffffffffffff?, 0x0?}, {0xc00001f260?, 0x7f3e13e67b70?, 0x2d5e8bfa8dd?})
        /usr/lib/go/src/net/udpsock.go:191 +0x25 fp=0xc000fffbd0 sp=0xc000fffb60 pc=0x5779e5
github.com/ericwq/aprilsh/network.(*Connection).recvOne(0xc0006180c0, {0x7f3e13d96ab8, 0xc000126118})
        /home/ide/develop/aprilsh/network/network.go:608 +0xb9 fp=0xc000fffd40 sp=0xc000fffbd0 pc=0x67e5f9
github.com/ericwq/aprilsh/network.(*Connection).Recv(0xc0006180c0, 0x1)
        /home/ide/develop/aprilsh/network/network.go:810 +0x16b fp=0xc000fffec8 sp=0xc000fffd40 pc=0x67fc4b
github.com/ericwq/aprilsh/frontend.ReadFromNetwork(0x0?, 0x0?, 0x0?, {0x7876a8, 0xc0006180c0})
        /home/ide/develop/aprilsh/frontend/read.go:94 +0x68 fp=0xc000ffff40 sp=0xc000fffec8 pc=0x5d8708
main.serve.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:714 +0x38 fp=0xc000ffff78 sp=0xc000ffff40 pc=0x68ca18
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc000ffffe0 sp=0xc000ffff78 pc=0x686556
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000ffffe8 sp=0xc000ffffe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 26
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

## serve() ReadFromFile()
goroutine 12 [syscall, 51 minutes]:
syscall.Syscall(0xa474098e4?, 0x6e13c0?, 0x4d4127?, 0x7ffff800000?)
        /usr/lib/go/src/syscall/syscall_linux.go:69 +0x25 fp=0xc000097db0 sp=0xc000097d40 pc=0x4bc8e5
syscall.read(0xc0007c4240?, {0xc000b82000?, 0x1?, 0x72?})
        /usr/lib/go/src/syscall/zsyscall_linux_amd64.go:721 +0x38 fp=0xc000097df0 sp=0xc000097db0 pc=0x4ba918
syscall.Read(...)
        /usr/lib/go/src/syscall/syscall_unix.go:181
internal/poll.ignoringEINTRIO(...)
        /usr/lib/go/src/internal/poll/fd_unix.go:736
internal/poll.(*FD).Read(0xc0007c4240, {0xc000b82000, 0x4000, 0x4000})
        /usr/lib/go/src/internal/poll/fd_unix.go:160 +0x2ae fp=0xc000097e88 sp=0xc000097df0 pc=0x4d556e
os.(*File).read(...)
        /usr/lib/go/src/os/file_posix.go:29
os.(*File).Read(0xc000126120, {0xc000b82000?, 0x9e17e0?, 0x9e17e0?})
        /usr/lib/go/src/os/file.go:118 +0x52 fp=0xc000097ec8 sp=0xc000097e88 pc=0x4e0492
github.com/ericwq/aprilsh/frontend.ReadFromFile(0x1, 0x0?, 0x0?, {0x788228, 0xc000126120})
        /home/ide/develop/aprilsh/frontend/read.go:60 +0xb1 fp=0xc000097f40 sp=0xc000097ec8 pc=0x5d8591
main.serve.func2()
        /home/ide/develop/aprilsh/frontend/server/server.go:720 +0x35 fp=0xc000097f78 sp=0xc000097f40 pc=0x68c9b5
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc000097fe0 sp=0xc000097f78 pc=0x686556
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000097fe8 sp=0xc000097fe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 26
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

## runWorker() shell.Wait()
goroutine 37 [syscall, 50 minutes]:
syscall.Syscall6(0x695a00?, 0xc0000a5cb0?, 0x40b64c?, 0x6e2dc0?, 0x5a5394?, 0xc0000a5cb0?, 0x40b41e?)
        /usr/lib/go/src/syscall/syscall_linux.go:91 +0x30 fp=0xc0000a5c70 sp=0xc0000a5be8 pc=0x4bc970
os.(*Process).blockUntilWaitable(0xc000b580f0)
        /usr/lib/go/src/os/wait_waitid.go:32 +0x76 fp=0xc0000a5d48 sp=0xc0000a5c70 pc=0x4e4056
os.(*Process).wait(0xc000b580f0)
        /usr/lib/go/src/os/exec_unix.go:22 +0x25 fp=0xc0000a5da8 sp=0xc0000a5d48 pc=0x4dfd05
os.(*Process).Wait(...)
        /usr/lib/go/src/os/exec.go:134
main.runWorker(0xc00012a820, 0xc00012e1e0, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:648 +0xa67 fp=0xc0000a5f88 sp=0xc0000a5da8 pc=0x689387
main.(*mainSrv).run.func2(0x0?, 0x0?, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:1431 +0x2e fp=0xc0000a5fb8 sp=0xc0000a5f88 pc=0x68fa8e
main.(*mainSrv).run.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:1433 +0x2f fp=0xc0000a5fe0 sp=0xc0000a5fb8 pc=0x68fa2f
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000a5fe8 sp=0xc0000a5fe0 pc=0x46dfc1
created by main.(*mainSrv).run in goroutine 23
        /home/ide/develop/aprilsh/frontend/server/server.go:1430 +0xa12

## serve() select
goroutine 38 [select]:
runtime.gopark(0xc000ffdf38?, 0x5?, 0x0?, 0x46?, 0xc000ffdcb6?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000ffda68 sp=0xc000ffda48 pc=0x43e8ae
runtime.selectgo(0xc000ffdf38, 0xc000ffdcac, 0xa0fdc0?, 0x0, 0x723426?, 0x1)
        /usr/lib/go/src/runtime/select.go:327 +0x725 fp=0xc000ffdb88 sp=0xc000ffda68 pc=0x44e3c5
main.serve(0xc0007be0d8, 0xc0007be0e8, 0xc00014a310, 0xc000b791d0, 0x0, 0x0)
        /home/ide/develop/aprilsh/frontend/server/server.go:769 +0x9df fp=0xc000ffdf98 sp=0xc000ffdb88 pc=0x68a2df
main.runWorker.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:621 +0x4a fp=0xc000ffdfe0 sp=0xc000ffdf98 pc=0x6896ca
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000ffdfe8 sp=0xc000ffdfe0 pc=0x46dfc1
created by main.runWorker in goroutine 37
        /home/ide/develop/aprilsh/frontend/server/server.go:619 +0x719

## serve() ReadFromNetwork()
goroutine 50 [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc0000a7800 sp=0xc0000a77e0 pc=0x43e8ae
runtime.netpollblock(0x0?, 0x409326?, 0x0?)
        /usr/lib/go/src/runtime/netpoll.go:564 +0xf7 fp=0xc0000a7838 sp=0xc0000a7800 pc=0x437317
internal/poll.runtime_pollWait(0x7f3e13e67950, 0x72)
        /usr/lib/go/src/runtime/netpoll.go:343 +0x85 fp=0xc0000a7858 sp=0xc0000a7838 pc=0x468c85
internal/poll.(*pollDesc).wait(0xc0007e0880?, 0xc0000f5900?, 0x0)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc0000a7880 sp=0xc0000a7858 pc=0x4d4247
internal/poll.(*pollDesc).waitRead(...)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadMsgInet6(0xc0007e0880, {0xc0000f5900, 0x4e4, 0x4e4}, {0xc001009ec0, 0x28, 0x28}, 0x0?, 0x0?)
        /usr/lib/go/src/internal/poll/fd_unix.go:355 +0x339 fp=0xc0000a7960 sp=0xc0000a7880 pc=0x4d7179
net.(*netFD).readMsgInet6(0xc0007e0880, {0xc0000f5900?, 0xc00011c048?, 0x0?}, {0xc001009ec0?, 0x45?, 0xc0000a7a50?}, 0x41c2e8?, 0x43628a?)
        /usr/lib/go/src/net/fd_posix.go:90 +0x31 fp=0xc0000a79e0 sp=0xc0000a7960 pc=0x562771
net.(*UDPConn).readMsg(0x4130a5?, {0xc0000f5900?, 0x6ecc40?, 0xc0000a7b40?}, {0xc001009ec0?, 0xc0007e6200?, 0x2?})
        /usr/lib/go/src/net/udpsock_posix.go:106 +0x9c fp=0xc0000a7ad0 sp=0xc0000a79e0 pc=0x5796fc
net.(*UDPConn).ReadMsgUDPAddrPort(0xc0007be0d0, {0xc0000f5900?, 0x7f3e5aac1878?, 0x30?}, {0xc001009ec0?, 0xc001009ec0?, 0x0?})
        /usr/lib/go/src/net/udpsock.go:203 +0x3e fp=0xc0000a7b60 sp=0xc0000a7ad0 pc=0x577ade
net.(*UDPConn).ReadMsgUDP(0xc0000a7be8?, {0xc0000f5900?, 0xffffffffffffffff?, 0x0?}, {0xc001009ec0?, 0x7f3e13e67980?, 0x2d5e8bd7303?})
        /usr/lib/go/src/net/udpsock.go:191 +0x25 fp=0xc0000a7bd0 sp=0xc0000a7b60 pc=0x5779e5
github.com/ericwq/aprilsh/network.(*Connection).recvOne(0xc000618180, {0x7f3e13d96ab8, 0xc0007be0d0})
        /home/ide/develop/aprilsh/network/network.go:608 +0xb9 fp=0xc0000a7d40 sp=0xc0000a7bd0 pc=0x67e5f9
github.com/ericwq/aprilsh/network.(*Connection).Recv(0xc000618180, 0x1)
        /home/ide/develop/aprilsh/network/network.go:810 +0x16b fp=0xc0000a7ec8 sp=0xc0000a7d40 pc=0x67fc4b
github.com/ericwq/aprilsh/frontend.ReadFromNetwork(0x0?, 0x0?, 0x0?, {0x7876a8, 0xc000618180})
        /home/ide/develop/aprilsh/frontend/read.go:94 +0x68 fp=0xc0000a7f40 sp=0xc0000a7ec8 pc=0x5d8708
main.serve.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:714 +0x38 fp=0xc0000a7f78 sp=0xc0000a7f40 pc=0x68ca18
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc0000a7fe0 sp=0xc0000a7f78 pc=0x686556
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000a7fe8 sp=0xc0000a7fe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 38
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

## serve() ReadFromFile()
goroutine 51 [syscall, 50 minutes]:
syscall.Syscall(0x103863ce53?, 0x6e13c0?, 0x4d4127?, 0x7ffff800000?)
        /usr/lib/go/src/syscall/syscall_linux.go:69 +0x25 fp=0xc0000975b0 sp=0xc000097540 pc=0x4bc8e5
syscall.read(0xc00012efc0?, {0xc001324000?, 0x1?, 0x72?})
        /usr/lib/go/src/syscall/zsyscall_linux_amd64.go:721 +0x38 fp=0xc0000975f0 sp=0xc0000975b0 pc=0x4ba918
syscall.Read(...)
        /usr/lib/go/src/syscall/syscall_unix.go:181
internal/poll.ignoringEINTRIO(...)
        /usr/lib/go/src/internal/poll/fd_unix.go:736
internal/poll.(*FD).Read(0xc00012efc0, {0xc001324000, 0x4000, 0x4000})
        /usr/lib/go/src/internal/poll/fd_unix.go:160 +0x2ae fp=0xc000097688 sp=0xc0000975f0 pc=0x4d556e
os.(*File).read(...)
        /usr/lib/go/src/os/file_posix.go:29
os.(*File).Read(0xc0007be0d8, {0xc001324000?, 0x9e17e0?, 0x9e17e0?})
        /usr/lib/go/src/os/file.go:118 +0x52 fp=0xc0000976c8 sp=0xc000097688 pc=0x4e0492
github.com/ericwq/aprilsh/frontend.ReadFromFile(0x1, 0x0?, 0x0?, {0x788228, 0xc0007be0d8})
        /home/ide/develop/aprilsh/frontend/read.go:60 +0xb1 fp=0xc000097740 sp=0xc0000976c8 pc=0x5d8591
main.serve.func2()
        /home/ide/develop/aprilsh/frontend/server/server.go:720 +0x35 fp=0xc000097778 sp=0xc000097740 pc=0x68c9b5
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc0000977e0 sp=0xc000097778 pc=0x686556
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000977e8 sp=0xc0000977e0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 38
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96
