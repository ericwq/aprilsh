## Changelog

<details>
<summary>2022</summary>
  
- 2022/Apr/01: start terminal emulator.
- 2022/Jul/31: finish the terminal emulator:
  - add scroll buffer support,
  - add color palette support,
  - refine UTF-8 support.
- 2022/Aug/04: start the prediction engine.
- 2022/Aug/29: finish the prediction engine.
  - refine UTF-8 support.
- 2022/Sep/20: finish the UDP network.
- 2022/Sep/28: finish user input state.
- 2022/Sep/29: refine cell width.
- 2022/Oct/02: add terminfo module.
- 2022/Oct/13: finish the Framebuffer for completeness.
- 2022/Oct/14: finish Complete state.
- 2022/Nov/04: finish Display.
- 2022/Nov/08: finish Complete testing.
- 2022/Nov/27: finish Transport and TransportSender.
- 2022/Dec/28: finish command-line parameter parsing and locale validation.
  
</details>
<details>
<summary>2023</summary>
  
- 2023/Mar/24: solve the locale problem in alpine.
- 2023/Apr/07: support concurrent UDP server.
- 2023/Apr/21: finish server start/stop part.
- 2023/May/01: study [s6](https://skarnet.org/software/s6/) as PID 1 process: [utmps](https://skarnet.org/software/utmps/) require s6, aprilsh should support openrc.
- 2023/May/16: finish [alpine container with openrc support](https://github.com/ericwq/s6)
- 2023/May/30: finish [eric/goutmp](https://github.com/ericwq/goutmp)
- 2023/Jun/07: upgrade to `ericwq/goutmp` v0.2.0.
- 2023/Jun/15: finish `warnUnattached()` part.
- 2023/Jun/21: finish serve() function.
- 2023/Jun/25: re-structure cmd directory.
- 2023/Jul/12: prepare client and server. fix bug in overlay.
- 2023/Jul/19: refine frontend, terminal, util package for test coverage.
- 2023/Jul/24: refine network package for test coverage.
- 2023/Aug/01: start integration test for client.
- 2023/Aug/07: add util.Log and rewrite log related part for other packages.
- 2023/Aug/14: accomplish `exit` command in running aprilsh client.
- 2023/Aug/22: add OSC 112, DECSCUR, XTWINOPS 22,23 support; study CSI u.
- 2023/Sep/15: improve the performance of client and server.
- 2023/Sep/28: fix bug for Display and add integration test for server.
- 2023/Oct/10: fix uncompress buffer size bug and fix max uint64 bug.
- 2023/Oct/17: fix Framebuffer.resize() resize bug.
- 2023/Oct/19: fix NewFrame() bug for alternate screen buffer.
- 2023/Oct/22: pass client Term to server.
- 2023/Oct/23: fix uncompress buffer overflow bug.
- 2023/Oct/27: fix window title bug.
- 2023/Nov/13: fix stream output mode display bug, #1.
- 2023/Nov/19: enhance stream mode to display over buffer size file, #2.
- 2023/Nov/29: enhance screen difference with mix sequence, fix bug #6,#7,#8.
- 2023/Dec/08: enhance title #14, limit concurrent user #17, fix bug #9,#10,#12,#14,#15,#16.
- 2023/Dec/13: fix bug #11, solve computer hibernate problem partly.
- 2023/Dec/28: enhance utmp access problem #17; fix read dead line problem #18.

</details>
<details>
<summary>2024</summary>
</details> 

- 2024/Jan/02: refine utmp access #22.
- 2024/Jan/09: refine for no connection shutdown #20, refine for no request time out,
- 2024/Jan/09: fix bug #19, #21, #23, #24.
- 2024/Jan/13: attacked by a terrible fever, the fever last more than 7 days, the cough last more than 10 days.
- 2024/Jan/25: finish unit test for Framebuffer and Emulator; fix bug #23, #27, #26.
- 2024/Feb/01: finish release workflow for source tar ball; finish build apk for alpine.
- 2024/Feb/09: aprilsh-openrc is ready #29; fix environment variable bug for login user #39.
- 2024/Feb/19: syslog support #37, customized ssh port #36, ssh authentication passphrase #41, 
- 2024/Feb/19: customized port #49, fetch key error handling #48, password support #45,
- 2024/Feb/19: refine hostkey callback #43.
- 2024/Mar/06: fix bug for server package #60, hide server command options, #59, fix bug for UDS name #61,
- 2024/Mar/06: check available port before use #51, main server listen on local port #58,
- 2024/Mar/06: child inherit options from parent #56, each client run on child process #55,
- 2024/Mar/06: fix bug for server quit #57.
- 2024/Mar/20: fix failed test #60,#61,#62,#63,#66; add container port mapping support #65,
- 2024/Mar/20: add supervisor for apshd #67; upgrade goutmp to 0.5.1.
- 2024/Mar/22: add logrotate for apshd, #68; fix test bug for #34; disable warnUnattached().
- 2024/Mar/23: prepare apk build files for aports publish, #34.
- 2024/Apr/01: skarnet rpm packaging: skalibs.
- 2024/Apr/02: skarnet rpm packaging: execline.
- 2024/Apr/11: skarnet rpm packaging: s6. with systemd service and journald support.
- 2024/Apr/14: skarnet rpm packaging: s6-dns.
- 2024/Apr/15: skarnet rpm packaging: s6-rc.
- 2024/Apr/15: skarnet rpm packaging: s6-networking.
- 2024/Apr/22: skarnet rpm packaging: tipidee.
- 2024/Apr/24: skarnet rpm packaging: sign rpm packages and publish skarnet yum/dnf repo.
- 2024/Apr/27: skarnet rpm packaging: move [ericwq/rpms](https://github.com/ericwq/rpms) project to [codeberg.org](https://codeberg.org/ericwq/rpms).
- 2024/Apr/30: update goutmp to support glibc based linux. build rpm package for 0.6.40.
- 2024/May/01: aprilsh rpm packaging.
- 2024/May/06: publish aprilsh yum/dnf repo.
- 2024/May/09: add homebrew aprilsh formula.
- 2024/May/13: add alpine private repo.
- 2024/May/15: fix alpine execute mode bug.
- 2024/May/23: fix ssh rsa public key login problem for alpine.
- 2024/May/24: add ssh container for fedora.
- 2024/May/27: refine apk build according to aports review (1st round)
- 2024/May/31: submit clean commit for aports review. add 3 new feat to issues.
- 2024/Jun/12: understand ssh-auth protocol and openssh implementation.
- 2024/Jun/15: aprilsh client support ssh auth methods: publickey and password.
- 2024/Jun/25: upgrade goutmp to 0.5.3; add aprilsh avatar.
- 2024/Jun/30: fix diagnostics warning for souce; refine APK according to aports review (2nd review).
- 2024/Jul/01: alpine aports approve aprilsh 0.7.5; thanks, Kevin Daudt.
- 2024/Jul/15: still working on prediction engine.
- 2024/Jul/17: overlay: add input echo mechanism to improve server side timeout.
- 2024/Jul/23: prediction engine works for slow network.
- 2024/Jul/30: support XTWINOPS 8, DECSET 12 and DECRST 12, Underline Style.
- 2024/Aug/07: support SGR 1006, synchronized output, OSC 8, XTGETTCAP, terminfo sub-package
- 2024/Aug/15: support terminal query, support CSI u basic, fix package list mode test problem
