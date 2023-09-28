# Aprilsh

## Status

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
- 2023/Mar/24: solve the locale problem in alpine.
- 2023/Apr/07: support concurrent UDP server.
- 2023/Apr/21: finish server start/stop part.
- 2023/May/01: study [s6](https://skarnet.org/software/s6/) as PID 1 process: [utmps](https://skarnet.org/software/utmps/) require s6, aprilsh also require s6 or similar alternative.
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
