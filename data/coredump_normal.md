goroutine 0 [idle]:
runtime.futex()
        /usr/lib/go/src/runtime/sys_linux_amd64.s:557 +0x21 fp=0x7fff8e7241c0 sp=0x7fff8e7241b8 pc=0x46fd81
runtime.futexsleep(0x4?, 0x0?, 0x7fff8e724238?)
        /usr/lib/go/src/runtime/os_linux.go:69 +0x30 fp=0x7fff8e724210 sp=0x7fff8e7241c0 pc=0x437f50
runtime.notesleep(0x9e2248)
        /usr/lib/go/src/runtime/lock_futex.go:160 +0x87 fp=0x7fff8e724248 sp=0x7fff8e724210 pc=0x4114e7
runtime.mPark(...)
        /usr/lib/go/src/runtime/proc.go:1632
runtime.stoplockedm()
        /usr/lib/go/src/runtime/proc.go:2780 +0x73 fp=0x7fff8e7242a0 sp=0x7fff8e724248 pc=0x442eb3
runtime.schedule()
        /usr/lib/go/src/runtime/proc.go:3561 +0x3a fp=0x7fff8e7242d8 sp=0x7fff8e7242a0 pc=0x444cfa
runtime.park_m(0xc000083380?)
        /usr/lib/go/src/runtime/proc.go:3745 +0x11f fp=0x7fff8e724320 sp=0x7fff8e7242d8 pc=0x44527f
traceback: unexpected SPWRITE function runtime.mcall
runtime.mcall()
        /usr/lib/go/src/runtime/asm_amd64.s:458 +0x4e fp=0x7fff8e724338 sp=0x7fff8e724320 pc=0x46bfce

goroutine 1 [semacquire, 3 minutes]:
## mainSrv.wait()
runtime.gopark(0x46c032?, 0xc0005b2e60?, 0xa0?, 0x42?, 0xc0005b2e40?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc0005b2e20 sp=0xc0005b2e00 pc=0x43e8ae
runtime.goparkunlock(...)
        /usr/lib/go/src/runtime/proc.go:404
runtime.semacquire1(0xc0000b41d0, 0x1?, 0x1, 0x0, 0x40?)
        /usr/lib/go/src/runtime/sema.go:160 +0x218 fp=0xc0005b2e88 sp=0xc0005b2e20 pc=0x44f3f8
sync.runtime_Semacquire(0x722383?)
        /usr/lib/go/src/runtime/sema.go:62 +0x25 fp=0xc0005b2ec0 sp=0xc0005b2e88 pc=0x46a665
sync.(*WaitGroup).Wait(0xc0000b4180?)
        /usr/lib/go/src/sync/waitgroup.go:116 +0x48 fp=0xc0005b2ee8 sp=0xc0005b2ec0 pc=0x4893a8
main.(*mainSrv).wait(...)
        /home/ide/develop/aprilsh/frontend/server/server.go:1536
main.main()
        /home/ide/develop/aprilsh/frontend/server/server.go:449 +0x24c fp=0xc0005b2f40 sp=0xc0005b2ee8 pc=0x68836c
runtime.main()
        /usr/lib/go/src/runtime/proc.go:267 +0x2bb fp=0xc0005b2fe0 sp=0xc0005b2f40 pc=0x43e45b
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0005b2fe8 sp=0xc0005b2fe0 pc=0x46dfc1

goroutine 2 [force gc (idle), 3 minutes]:
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
runtime.gopark(0x139d91?, 0x117875?, 0x0?, 0x0?, 0x0?)
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

goroutine 18 [finalizer wait, 3 minutes]:
runtime.gopark(0x198?, 0x71e5a0?, 0x1?, 0xfa?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000044620 sp=0xc000044600 pc=0x43e8ae
runtime.runfinq()
        /usr/lib/go/src/runtime/mfinal.go:193 +0x107 fp=0xc0000447e0 sp=0xc000044620 pc=0x41e967
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000447e8 sp=0xc0000447e0 pc=0x46dfc1
created by runtime.createfing in goroutine 1
        /usr/lib/go/src/runtime/mfinal.go:163 +0x3d

goroutine 19 [GC worker (idle), 3 minutes]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000040750 sp=0xc000040730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000407e0 sp=0xc000040750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000407e8 sp=0xc0000407e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 20 [GC worker (idle)]:
runtime.gopark(0xa10d20?, 0x1?, 0x3d?, 0xd6?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000040f50 sp=0xc000040f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000040fe0 sp=0xc000040f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000040fe8 sp=0xc000040fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 5 [GC worker (idle)]:
runtime.gopark(0x1e578a112e419?, 0x3?, 0xfa?, 0x6d?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000046750 sp=0xc000046730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000467e0 sp=0xc000046750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000467e8 sp=0xc0000467e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 6 [GC worker (idle)]:
runtime.gopark(0x1e57829521220?, 0x3?, 0xc4?, 0xfa?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000046f50 sp=0xc000046f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000046fe0 sp=0xc000046f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000046fe8 sp=0xc000046fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 34 [GC worker (idle)]:
runtime.gopark(0x1e57449cbbbcf?, 0x3?, 0x2c?, 0x7a?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000588750 sp=0xc000588730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0005887e0 sp=0xc000588750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0005887e8 sp=0xc0005887e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 7 [GC worker (idle)]:
runtime.gopark(0x1e573b10036d1?, 0x1?, 0x56?, 0x3d?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000047750 sp=0xc000047730 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc0000477e0 sp=0xc000047750 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0000477e8 sp=0xc0000477e0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 35 [GC worker (idle)]:
runtime.gopark(0x1e578a109d900?, 0x3?, 0x4b?, 0xd7?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000588f50 sp=0xc000588f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000588fe0 sp=0xc000588f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000588fe8 sp=0xc000588fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 8 [GC worker (idle)]:
runtime.gopark(0x1e578a1142e00?, 0x1?, 0x7b?, 0x43?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000047f50 sp=0xc000047f30 pc=0x43e8ae
runtime.gcBgMarkWorker()
        /usr/lib/go/src/runtime/mgc.go:1295 +0xe5 fp=0xc000047fe0 sp=0xc000047f50 pc=0x4214c5
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000047fe8 sp=0xc000047fe0 pc=0x46dfc1
created by runtime.gcBgMarkStartWorkers in goroutine 1
        /usr/lib/go/src/runtime/mgc.go:1219 +0x1c

goroutine 36 [IO wait]:
## mainSrv.run() read from UDP
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc0005b4980 sp=0xc0005b4960 pc=0x43e8ae
runtime.netpollblock(0x0?, 0x409326?, 0x0?)
        /usr/lib/go/src/runtime/netpoll.go:564 +0xf7 fp=0xc0005b49b8 sp=0xc0005b4980 pc=0x437317
internal/poll.runtime_pollWait(0x7feed82a0e28, 0x72)
        /usr/lib/go/src/runtime/netpoll.go:343 +0x85 fp=0xc0005b49d8 sp=0xc0005b49b8 pc=0x468c85
internal/poll.(*pollDesc).wait(0xc0005aa100?, 0xc0005b4ce0?, 0x0)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc0005b4a00 sp=0xc0005b49d8 pc=0x4d4247
internal/poll.(*pollDesc).waitRead(...)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadFromInet6(0xc0005aa100, {0xc0005b4ce0, 0x80, 0x80}, 0x7feed82a0e70?)
        /usr/lib/go/src/internal/poll/fd_unix.go:274 +0x22b fp=0xc0005b4a98 sp=0xc0005b4a00 pc=0x4d626b
net.(*netFD).readFromInet6(0xc0005aa100, {0xc0005b4ce0?, 0xffffffffffffffff?, 0xffffffffffffffff?}, 0x0?)
        /usr/lib/go/src/net/fd_posix.go:72 +0x25 fp=0xc0005b4ae8 sp=0xc0005b4a98 pc=0x5623a5
net.(*UDPConn).readFrom(0x30?, {0xc0005b4ce0?, 0xc00106a090?, 0x0?}, 0xc00106a090)
        /usr/lib/go/src/net/udpsock_posix.go:59 +0x79 fp=0xc0005b4bd8 sp=0xc0005b4ae8 pc=0x579219
net.(*UDPConn).readFromUDP(0xc0000a4068, {0xc0005b4ce0?, 0x9e1820?, 0x9e1820?}, 0xc00108d350?)
        /usr/lib/go/src/net/udpsock.go:149 +0x30 fp=0xc0005b4c30 sp=0xc0005b4bd8 pc=0x577610
net.(*UDPConn).ReadFromUDP(...)
        /usr/lib/go/src/net/udpsock.go:141
main.(*mainSrv).run(0xc0000b4180, 0xc0000b03c0)
        /home/ide/develop/aprilsh/frontend/server/server.go:1388 +0x4fe fp=0xc0005b4fb8 sp=0xc0005b4c30 pc=0x68e7fe
main.(*mainSrv).start.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:1263 +0x2a fp=0xc0005b4fe0 sp=0xc0005b4fb8 pc=0x68dfaa
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0005b4fe8 sp=0xc0005b4fe0 pc=0x46dfc1
created by main.(*mainSrv).start in goroutine 1
        /home/ide/develop/aprilsh/frontend/server/server.go:1262 +0x159

goroutine 37 [select, 3 minutes, locked to thread]:
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

goroutine 9 [syscall, 3 minutes]:
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

goroutine 21 [syscall, 3 minutes]:
## runWorker() shell.Wait()
syscall.Syscall6(0x695a40?, 0xc00011bcb0?, 0x40b64c?, 0x6e2dc0?, 0x5a5394?, 0xc00011bcb0?, 0x40b41e?)
        /usr/lib/go/src/syscall/syscall_linux.go:91 +0x30 fp=0xc00011bc70 sp=0xc00011bbe8 pc=0x4bc970
os.(*Process).blockUntilWaitable(0xc000804090)
        /usr/lib/go/src/os/wait_waitid.go:32 +0x76 fp=0xc00011bd48 sp=0xc00011bc70 pc=0x4e4056
os.(*Process).wait(0xc000804090)
        /usr/lib/go/src/os/exec_unix.go:22 +0x25 fp=0xc00011bda8 sp=0xc00011bd48 pc=0x4dfd05
os.(*Process).Wait(...)
        /usr/lib/go/src/os/exec.go:134
main.runWorker(0xc000132000, 0xc0000b41e0, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:648 +0xa67 fp=0xc00011bf88 sp=0xc00011bda8 pc=0x6893c7
main.(*mainSrv).run.func2(0x0?, 0x0?, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:1431 +0x2e fp=0xc00011bfb8 sp=0xc00011bf88 pc=0x68face
main.(*mainSrv).run.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:1433 +0x2f fp=0xc00011bfe0 sp=0xc00011bfb8 pc=0x68fa6f
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc00011bfe8 sp=0xc00011bfe0 pc=0x46dfc1
created by main.(*mainSrv).run in goroutine 36
        /home/ide/develop/aprilsh/frontend/server/server.go:1430 +0xa12

goroutine 22 [select]:
## serve() select
runtime.gopark(0xc000117f38?, 0x5?, 0x60?, 0x15?, 0xc000117cb6?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000117a68 sp=0xc000117a48 pc=0x43e8ae
runtime.selectgo(0xc000117f38, 0xc000117cac, 0xa0fe00?, 0x0, 0x723426?, 0x1)
        /usr/lib/go/src/runtime/select.go:327 +0x725 fp=0xc000117b88 sp=0xc000117a68 pc=0x44e3c5
main.serve(0xc00059e040, 0xc00059e050, 0xc00012a070, 0xc0001317c0, 0x0, 0x0)
        /home/ide/develop/aprilsh/frontend/server/server.go:769 +0x9df fp=0xc000117f98 sp=0xc000117b88 pc=0x68a31f
main.runWorker.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:621 +0x4a fp=0xc000117fe0 sp=0xc000117f98 pc=0x68970a
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000117fe8 sp=0xc000117fe0 pc=0x46dfc1
created by main.runWorker in goroutine 21
        /home/ide/develop/aprilsh/frontend/server/server.go:619 +0x719

goroutine 38 [IO wait]:
## serve() ReadFromNetwork()
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc000619800 sp=0xc0006197e0 pc=0x43e8ae
runtime.netpollblock(0x0?, 0x409326?, 0x0?)
        /usr/lib/go/src/runtime/netpoll.go:564 +0xf7 fp=0xc000619838 sp=0xc000619800 pc=0x437317
internal/poll.runtime_pollWait(0x7feed82a0d30, 0x72)
        /usr/lib/go/src/runtime/netpoll.go:343 +0x85 fp=0xc000619858 sp=0xc000619838 pc=0x468c85
internal/poll.(*pollDesc).wait(0xc00007c100?, 0xc0010c9400?, 0x0)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000619880 sp=0xc000619858 pc=0x4d4247
internal/poll.(*pollDesc).waitRead(...)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadMsgInet6(0xc00007c100, {0xc0010c9400, 0x4e4, 0x4e4}, {0xc001091c80, 0x28, 0x28}, 0x0?, 0x0?)
        /usr/lib/go/src/internal/poll/fd_unix.go:355 +0x339 fp=0xc000619960 sp=0xc000619880 pc=0x4d7179
net.(*netFD).readMsgInet6(0xc00007c100, {0xc0010c9400?, 0xc0000a2060?, 0x0?}, {0xc001091c80?, 0x45?, 0xc000619a50?}, 0x41c2e8?, 0x43628a?)
        /usr/lib/go/src/net/fd_posix.go:90 +0x31 fp=0xc0006199e0 sp=0xc000619960 pc=0x562771
net.(*UDPConn).readMsg(0x4130a5?, {0xc0010c9400?, 0x6ecc40?, 0xc000619b40?}, {0xc001091c80?, 0xc00007c600?, 0x2?})
        /usr/lib/go/src/net/udpsock_posix.go:106 +0x9c fp=0xc000619ad0 sp=0xc0006199e0 pc=0x5796fc
net.(*UDPConn).ReadMsgUDPAddrPort(0xc00059e038, {0xc0010c9400?, 0x7fef1ee965b8?, 0x30?}, {0xc001091c80?, 0xc001091c80?, 0x0?})
        /usr/lib/go/src/net/udpsock.go:203 +0x3e fp=0xc000619b60 sp=0xc000619ad0 pc=0x577ade
net.(*UDPConn).ReadMsgUDP(0xc000619be8?, {0xc0010c9400?, 0xffffffffffffffff?, 0x0?}, {0xc001091c80?, 0x7feed82a0d60?, 0x2fe50f729d?})
        /usr/lib/go/src/net/udpsock.go:191 +0x25 fp=0xc000619bd0 sp=0xc000619b60 pc=0x5779e5
github.com/ericwq/aprilsh/network.(*Connection).recvOne(0xc000144000, {0x7feed81a45e0, 0xc00059e038})
        /home/ide/develop/aprilsh/network/network.go:608 +0xb9 fp=0xc000619d40 sp=0xc000619bd0 pc=0x67e5f9
github.com/ericwq/aprilsh/network.(*Connection).Recv(0xc000144000, 0x1)
        /home/ide/develop/aprilsh/network/network.go:810 +0x16b fp=0xc000619ec8 sp=0xc000619d40 pc=0x67fc8b
github.com/ericwq/aprilsh/frontend.ReadFromNetwork(0x0?, 0x0?, 0x0?, {0x7876a8, 0xc000144000})
        /home/ide/develop/aprilsh/frontend/read.go:94 +0x68 fp=0xc000619f40 sp=0xc000619ec8 pc=0x5d8708
main.serve.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:714 +0x38 fp=0xc000619f78 sp=0xc000619f40 pc=0x68ca58
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc000619fe0 sp=0xc000619f78 pc=0x686596
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000619fe8 sp=0xc000619fe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 22
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

goroutine 39 [syscall, 3 minutes]:
## serve() ReadFromFile
syscall.Syscall(0x1ba672401?, 0x6e13c0?, 0x4d4127?, 0x7ffff800000?)
        /usr/lib/go/src/syscall/syscall_linux.go:69 +0x25 fp=0xc000054db0 sp=0xc000054d40 pc=0x4bc8e5
syscall.read(0xc0000700c0?, {0xc00058c000?, 0x1?, 0x72?})
        /usr/lib/go/src/syscall/zsyscall_linux_amd64.go:721 +0x38 fp=0xc000054df0 sp=0xc000054db0 pc=0x4ba918
syscall.Read(...)
        /usr/lib/go/src/syscall/syscall_unix.go:181
internal/poll.ignoringEINTRIO(...)
        /usr/lib/go/src/internal/poll/fd_unix.go:736
internal/poll.(*FD).Read(0xc0000700c0, {0xc00058c000, 0x4000, 0x4000})
        /usr/lib/go/src/internal/poll/fd_unix.go:160 +0x2ae fp=0xc000054e88 sp=0xc000054df0 pc=0x4d556e
os.(*File).read(...)
        /usr/lib/go/src/os/file_posix.go:29
os.(*File).Read(0xc00059e040, {0xc00058c000?, 0x9e1820?, 0x9e1820?})
        /usr/lib/go/src/os/file.go:118 +0x52 fp=0xc000054ec8 sp=0xc000054e88 pc=0x4e0492
github.com/ericwq/aprilsh/frontend.ReadFromFile(0x1, 0x0?, 0x0?, {0x788228, 0xc00059e040})
        /home/ide/develop/aprilsh/frontend/read.go:60 +0xb1 fp=0xc000054f40 sp=0xc000054ec8 pc=0x5d8591
main.serve.func2()
        /home/ide/develop/aprilsh/frontend/server/server.go:720 +0x35 fp=0xc000054f78 sp=0xc000054f40 pc=0x68c9f5
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc000054fe0 sp=0xc000054f78 pc=0x686596
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000054fe8 sp=0xc000054fe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 22
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

goroutine 23 [syscall, 3 minutes]:
## runWorker() shell.Wait()
syscall.Syscall6(0x695a40?, 0xc000119cb0?, 0x40b64c?, 0x6e2dc0?, 0x5a5394?, 0xc000119cb0?, 0x40b41e?)
        /usr/lib/go/src/syscall/syscall_linux.go:91 +0x30 fp=0xc000119c70 sp=0xc000119be8 pc=0x4bc970
os.(*Process).blockUntilWaitable(0xc000cc8240)
        /usr/lib/go/src/os/wait_waitid.go:32 +0x76 fp=0xc000119d48 sp=0xc000119c70 pc=0x4e4056
os.(*Process).wait(0xc000cc8240)
        /usr/lib/go/src/os/exec_unix.go:22 +0x25 fp=0xc000119da8 sp=0xc000119d48 pc=0x4dfd05
os.(*Process).Wait(...)
        /usr/lib/go/src/os/exec.go:134
main.runWorker(0xc0001325a0, 0xc0000b41e0, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:648 +0xa67 fp=0xc000119f88 sp=0xc000119da8 pc=0x6893c7
main.(*mainSrv).run.func2(0x0?, 0x0?, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:1431 +0x2e fp=0xc000119fb8 sp=0xc000119f88 pc=0x68face
main.(*mainSrv).run.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:1433 +0x2f fp=0xc000119fe0 sp=0xc000119fb8 pc=0x68fa6f
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc000119fe8 sp=0xc000119fe0 pc=0x46dfc1
created by main.(*mainSrv).run in goroutine 36
        /home/ide/develop/aprilsh/frontend/server/server.go:1430 +0xa12

goroutine 40 [select]:
## serve() select
runtime.gopark(0xc0007aff38?, 0x5?, 0x80?, 0x19?, 0xc0007afcb6?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc0007afa68 sp=0xc0007afa48 pc=0x43e8ae
runtime.selectgo(0xc0007aff38, 0xc0007afcac, 0xa0fe00?, 0x0, 0x723426?, 0x1)
        /usr/lib/go/src/runtime/select.go:327 +0x725 fp=0xc0007afb88 sp=0xc0007afa68 pc=0x44e3c5
main.serve(0xc00059e070, 0xc00059e0b8, 0xc0008545b0, 0xc000cc11d0, 0x0, 0x0)
        /home/ide/develop/aprilsh/frontend/server/server.go:769 +0x9df fp=0xc0007aff98 sp=0xc0007afb88 pc=0x68a31f
main.runWorker.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:621 +0x4a fp=0xc0007affe0 sp=0xc0007aff98 pc=0x68970a
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0007affe8 sp=0xc0007affe0 pc=0x46dfc1
created by main.runWorker in goroutine 23
        /home/ide/develop/aprilsh/frontend/server/server.go:619 +0x719

goroutine 50 [IO wait]:
## serve() ReadFromNetwork()
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc0007a9800 sp=0xc0007a97e0 pc=0x43e8ae
runtime.netpollblock(0x0?, 0x409326?, 0x0?)
        /usr/lib/go/src/runtime/netpoll.go:564 +0xf7 fp=0xc0007a9838 sp=0xc0007a9800 pc=0x437317
internal/poll.runtime_pollWait(0x7feed82a0b40, 0x72)
        /usr/lib/go/src/runtime/netpoll.go:343 +0x85 fp=0xc0007a9858 sp=0xc0007a9838 pc=0x468c85
internal/poll.(*pollDesc).wait(0xc0005ea800?, 0xc0010c8f00?, 0x0)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc0007a9880 sp=0xc0007a9858 pc=0x4d4247
internal/poll.(*pollDesc).waitRead(...)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadMsgInet6(0xc0005ea800, {0xc0010c8f00, 0x4e4, 0x4e4}, {0xc001091c50, 0x28, 0x28}, 0x0?, 0x0?)
        /usr/lib/go/src/internal/poll/fd_unix.go:355 +0x339 fp=0xc0007a9960 sp=0xc0007a9880 pc=0x4d7179
net.(*netFD).readMsgInet6(0xc0005ea800, {0xc0010c8f00?, 0xc0000a2060?, 0x0?}, {0xc001091c50?, 0x45?, 0xc0007a9a50?}, 0x41c2e8?, 0x43628a?)
        /usr/lib/go/src/net/fd_posix.go:90 +0x31 fp=0xc0007a99e0 sp=0xc0007a9960 pc=0x562771
net.(*UDPConn).readMsg(0x4130a5?, {0xc0010c8f00?, 0x6ecc40?, 0xc0007a9b40?}, {0xc001091c50?, 0xc00007c600?, 0x1?})
        /usr/lib/go/src/net/udpsock_posix.go:106 +0x9c fp=0xc0007a9ad0 sp=0xc0007a99e0 pc=0x5796fc
net.(*UDPConn).ReadMsgUDPAddrPort(0xc0000a4138, {0xc0010c8f00?, 0x7fef1ee965b8?, 0x30?}, {0xc001091c50?, 0xc001091c50?, 0x0?})
        /usr/lib/go/src/net/udpsock.go:203 +0x3e fp=0xc0007a9b60 sp=0xc0007a9ad0 pc=0x577ade
net.(*UDPConn).ReadMsgUDP(0xc0007a9be8?, {0xc0010c8f00?, 0xffffffffffffffff?, 0x0?}, {0xc001091c50?, 0x7feed82a0b70?, 0x2fe50f4028?})
        /usr/lib/go/src/net/udpsock.go:191 +0x25 fp=0xc0007a9bd0 sp=0xc0007a9b60 pc=0x5779e5
github.com/ericwq/aprilsh/network.(*Connection).recvOne(0xc0001440c0, {0x7feed81a45e0, 0xc0000a4138})
        /home/ide/develop/aprilsh/network/network.go:608 +0xb9 fp=0xc0007a9d40 sp=0xc0007a9bd0 pc=0x67e5f9
github.com/ericwq/aprilsh/network.(*Connection).Recv(0xc0001440c0, 0x1)
        /home/ide/develop/aprilsh/network/network.go:810 +0x16b fp=0xc0007a9ec8 sp=0xc0007a9d40 pc=0x67fc8b
github.com/ericwq/aprilsh/frontend.ReadFromNetwork(0x0?, 0x0?, 0x0?, {0x7876a8, 0xc0001440c0})
        /home/ide/develop/aprilsh/frontend/read.go:94 +0x68 fp=0xc0007a9f40 sp=0xc0007a9ec8 pc=0x5d8708
main.serve.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:714 +0x38 fp=0xc0007a9f78 sp=0xc0007a9f40 pc=0x68ca58
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc0007a9fe0 sp=0xc0007a9f78 pc=0x686596
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0007a9fe8 sp=0xc0007a9fe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 40
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

goroutine 51 [syscall, 3 minutes]:
## serve() ReadFromFile()
syscall.Syscall(0x22c85d862?, 0x6e13c0?, 0x4d4127?, 0x7ffff800000?)
        /usr/lib/go/src/syscall/syscall_linux.go:69 +0x25 fp=0xc00110d5b0 sp=0xc00110d540 pc=0x4bc8e5
syscall.read(0xc000070540?, {0xc001112000?, 0x1?, 0x72?})
        /usr/lib/go/src/syscall/zsyscall_linux_amd64.go:721 +0x38 fp=0xc00110d5f0 sp=0xc00110d5b0 pc=0x4ba918
syscall.Read(...)
        /usr/lib/go/src/syscall/syscall_unix.go:181
internal/poll.ignoringEINTRIO(...)
        /usr/lib/go/src/internal/poll/fd_unix.go:736
internal/poll.(*FD).Read(0xc000070540, {0xc001112000, 0x4000, 0x4000})
        /usr/lib/go/src/internal/poll/fd_unix.go:160 +0x2ae fp=0xc00110d688 sp=0xc00110d5f0 pc=0x4d556e
os.(*File).read(...)
        /usr/lib/go/src/os/file_posix.go:29
os.(*File).Read(0xc00059e070, {0xc001112000?, 0x9e1820?, 0x9e1820?})
        /usr/lib/go/src/os/file.go:118 +0x52 fp=0xc00110d6c8 sp=0xc00110d688 pc=0x4e0492
github.com/ericwq/aprilsh/frontend.ReadFromFile(0x1, 0x0?, 0x0?, {0x788228, 0xc00059e070})
        /home/ide/develop/aprilsh/frontend/read.go:60 +0xb1 fp=0xc00110d740 sp=0xc00110d6c8 pc=0x5d8591
main.serve.func2()
        /home/ide/develop/aprilsh/frontend/server/server.go:720 +0x35 fp=0xc00110d778 sp=0xc00110d740 pc=0x68c9f5
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc00110d7e0 sp=0xc00110d778 pc=0x686596
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc00110d7e8 sp=0xc00110d7e0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 40
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

goroutine 10 [syscall, 3 minutes]:
##  runWorker() shell.Wait()
syscall.Syscall6(0x695a40?, 0xc00011dcb0?, 0x40b64c?, 0x6e2dc0?, 0x5a5394?, 0xc00011dcb0?, 0x40b41e?)
        /usr/lib/go/src/syscall/syscall_linux.go:91 +0x30 fp=0xc00011dc70 sp=0xc00011dbe8 pc=0x4bc970
os.(*Process).blockUntilWaitable(0xc0000c63f0)
        /usr/lib/go/src/os/wait_waitid.go:32 +0x76 fp=0xc00011dd48 sp=0xc00011dc70 pc=0x4e4056
os.(*Process).wait(0xc0000c63f0)
        /usr/lib/go/src/os/exec_unix.go:22 +0x25 fp=0xc00011dda8 sp=0xc00011dd48 pc=0x4dfd05
os.(*Process).Wait(...)
        /usr/lib/go/src/os/exec.go:134
main.runWorker(0xc0001326e0, 0xc0000b41e0, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:648 +0xa67 fp=0xc00011df88 sp=0xc00011dda8 pc=0x6893c7
main.(*mainSrv).run.func2(0x0?, 0x0?, 0x0?)
        /home/ide/develop/aprilsh/frontend/server/server.go:1431 +0x2e fp=0xc00011dfb8 sp=0xc00011df88 pc=0x68face
main.(*mainSrv).run.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:1433 +0x2f fp=0xc00011dfe0 sp=0xc00011dfb8 pc=0x68fa6f
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc00011dfe8 sp=0xc00011dfe0 pc=0x46dfc1
created by main.(*mainSrv).run in goroutine 36
        /home/ide/develop/aprilsh/frontend/server/server.go:1430 +0xa12

goroutine 11 [select]:
## serve() select
runtime.gopark(0xc00061df38?, 0x5?, 0x60?, 0x18?, 0xc00061dcb6?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc00061da68 sp=0xc00061da48 pc=0x43e8ae
runtime.selectgo(0xc00061df38, 0xc00061dcac, 0xa0fe00?, 0x0, 0x723426?, 0x1)
        /usr/lib/go/src/runtime/select.go:327 +0x725 fp=0xc00061db88 sp=0xc00061da68 pc=0x44e3c5
main.serve(0xc000680020, 0xc000680030, 0xc0008544d0, 0xc0000e4960, 0x0, 0x0)
        /home/ide/develop/aprilsh/frontend/server/server.go:769 +0x9df fp=0xc00061df98 sp=0xc00061db88 pc=0x68a31f
main.runWorker.func3()
        /home/ide/develop/aprilsh/frontend/server/server.go:621 +0x4a fp=0xc00061dfe0 sp=0xc00061df98 pc=0x68970a
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc00061dfe8 sp=0xc00061dfe0 pc=0x46dfc1
created by main.runWorker in goroutine 10
        /home/ide/develop/aprilsh/frontend/server/server.go:619 +0x719

goroutine 66 [IO wait]:
##  serve() ReadFromNetwork()
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
        /usr/lib/go/src/runtime/proc.go:398 +0xce fp=0xc0007ab800 sp=0xc0007ab7e0 pc=0x43e8ae
runtime.netpollblock(0x0?, 0x409326?, 0x0?)
        /usr/lib/go/src/runtime/netpoll.go:564 +0xf7 fp=0xc0007ab838 sp=0xc0007ab800 pc=0x437317
internal/poll.runtime_pollWait(0x7feed82a0950, 0x72)
        /usr/lib/go/src/runtime/netpoll.go:343 +0x85 fp=0xc0007ab858 sp=0xc0007ab838 pc=0x468c85
internal/poll.(*pollDesc).wait(0xc00081e000?, 0xc001074500?, 0x0)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc0007ab880 sp=0xc0007ab858 pc=0x4d4247
internal/poll.(*pollDesc).waitRead(...)
        /usr/lib/go/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadMsgInet6(0xc00081e000, {0xc001074500, 0x4e4, 0x4e4}, {0xc001053050, 0x28, 0x28}, 0x7feed8161518?, 0x0?)
        /usr/lib/go/src/internal/poll/fd_unix.go:355 +0x339 fp=0xc0007ab960 sp=0xc0007ab880 pc=0x4d7179
net.(*netFD).readMsgInet6(0xc00081e000, {0xc001074500?, 0xc0000a2060?, 0x0?}, {0xc001053050?, 0x45?, 0xc0007aba50?}, 0x41c2e8?, 0x43628a?)
        /usr/lib/go/src/net/fd_posix.go:90 +0x31 fp=0xc0007ab9e0 sp=0xc0007ab960 pc=0x562771
net.(*UDPConn).readMsg(0x4130a5?, {0xc001074500?, 0x6ecc40?, 0xc0007abb40?}, {0xc001053050?, 0xc00007c780?, 0x2?})
        /usr/lib/go/src/net/udpsock_posix.go:106 +0x9c fp=0xc0007abad0 sp=0xc0007ab9e0 pc=0x5796fc
net.(*UDPConn).ReadMsgUDPAddrPort(0xc000680018, {0xc001074500?, 0x7fef1ee96a68?, 0x30?}, {0xc001053050?, 0xc001053050?, 0x0?})
        /usr/lib/go/src/net/udpsock.go:203 +0x3e fp=0xc0007abb60 sp=0xc0007abad0 pc=0x577ade
net.(*UDPConn).ReadMsgUDP(0xc0007abbe8?, {0xc001074500?, 0xffffffffffffffff?, 0x0?}, {0xc001053050?, 0x7feed82a0980?, 0x2fe5105621?})
        /usr/lib/go/src/net/udpsock.go:191 +0x25 fp=0xc0007abbd0 sp=0xc0007abb60 pc=0x5779e5
github.com/ericwq/aprilsh/network.(*Connection).recvOne(0xc000144180, {0x7feed81a45e0, 0xc000680018})
        /home/ide/develop/aprilsh/network/network.go:608 +0xb9 fp=0xc0007abd40 sp=0xc0007abbd0 pc=0x67e5f9
github.com/ericwq/aprilsh/network.(*Connection).Recv(0xc000144180, 0x1)
        /home/ide/develop/aprilsh/network/network.go:810 +0x16b fp=0xc0007abec8 sp=0xc0007abd40 pc=0x67fc8b
github.com/ericwq/aprilsh/frontend.ReadFromNetwork(0x0?, 0x0?, 0x0?, {0x7876a8, 0xc000144180})
        /home/ide/develop/aprilsh/frontend/read.go:94 +0x68 fp=0xc0007abf40 sp=0xc0007abec8 pc=0x5d8708
main.serve.func1()
        /home/ide/develop/aprilsh/frontend/server/server.go:714 +0x38 fp=0xc0007abf78 sp=0xc0007abf40 pc=0x68ca58
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc0007abfe0 sp=0xc0007abf78 pc=0x686596
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0007abfe8 sp=0xc0007abfe0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 11
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96

goroutine 67 [syscall, 3 minutes]:
## serve() ReadFromFile()
syscall.Syscall(0x2b3bdcdd2?, 0x6e13c0?, 0x4d4127?, 0x7ffff800000?)
        /usr/lib/go/src/syscall/syscall_linux.go:69 +0x25 fp=0xc0005895b0 sp=0xc000589540 pc=0x4bc8e5
syscall.read(0xc000590360?, {0xc000162000?, 0x1?, 0x72?})
        /usr/lib/go/src/syscall/zsyscall_linux_amd64.go:721 +0x38 fp=0xc0005895f0 sp=0xc0005895b0 pc=0x4ba918
syscall.Read(...)
        /usr/lib/go/src/syscall/syscall_unix.go:181
internal/poll.ignoringEINTRIO(...)
        /usr/lib/go/src/internal/poll/fd_unix.go:736
internal/poll.(*FD).Read(0xc000590360, {0xc000162000, 0x4000, 0x4000})
        /usr/lib/go/src/internal/poll/fd_unix.go:160 +0x2ae fp=0xc000589688 sp=0xc0005895f0 pc=0x4d556e
os.(*File).read(...)
        /usr/lib/go/src/os/file_posix.go:29
os.(*File).Read(0xc000680020, {0xc000162000?, 0x9e1820?, 0x9e1820?})
        /usr/lib/go/src/os/file.go:118 +0x52 fp=0xc0005896c8 sp=0xc000589688 pc=0x4e0492
github.com/ericwq/aprilsh/frontend.ReadFromFile(0x1, 0x0?, 0x0?, {0x788228, 0xc000680020})
        /home/ide/develop/aprilsh/frontend/read.go:60 +0xb1 fp=0xc000589740 sp=0xc0005896c8 pc=0x5d8591
main.serve.func2()
        /home/ide/develop/aprilsh/frontend/server/server.go:720 +0x35 fp=0xc000589778 sp=0xc000589740 pc=0x68c9f5
golang.org/x/sync/errgroup.(*Group).Go.func1()
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:75 +0x56 fp=0xc0005897e0 sp=0xc000589778 pc=0x686596
runtime.goexit()
        /usr/lib/go/src/runtime/asm_amd64.s:1650 +0x1 fp=0xc0005897e8 sp=0xc0005897e0 pc=0x46dfc1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 11
        /go/pkg/mod/golang.org/x/sync@v0.1.0/errgroup/errgroup.go:72 +0x96
