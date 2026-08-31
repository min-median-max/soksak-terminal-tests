# soksak-terminal-tests

Soksak이 사용하는 터미널 구현을 검증하는 비제품 품질 저장소입니다. 설치 대상이나
registry 항목이 아닙니다. benchmark는 복원 구현 owner의 feed·직렬화·RSS를 비교하고 system은 설치된
PTY와 provider의 demand·gap·tail·end-to-end 처리량을 공개
명령으로 release component를 설치한 뒤 복원 시나리오를 검증합니다.

플러그인과 사이드카 저장소는 각자의 conformance와 release gate를 소유합니다. 이 저장소는
설치 artifact, 공개 command, contract report만 사용하며 다른 저장소의 source를 빌드하지
않습니다.

Darwin과 Linux는 plugin 7개, PTY sidecar, recovery sidecar 6개를 사용합니다. Windows는
Alacritty, Ghostty, VT100, WezTerm, Xterm plugin과 PTY 및 recovery sidecar 4개를 사용합니다.
`verify-fleet`가 immutable release byte를 먼저 검증합니다. System suite는 빈 home으로 Core를
시작하고 인증된 Registry plugin identity만 `plugin.install`에 전달합니다. 각 plugin release가
정확한 Sidecar closure를 선언하고 Core가 이를 설치합니다. Consent summary를 읽고
승인한 후 plugin을 활성화하며, 공개 control surface를 통해 Core가 만든 `environment.json`을
검증합니다. 이 저장소는 제품 설치 상태를 직접 작성하지 않습니다.

터미널 크기 변경마다 `SOKSAK_TEST_EVIDENCE` 아래에 `<plugin-id>-resize.json`을 기록합니다.
이 파일은 pane resize receipt, 공개 terminal root의 넓은/좁은 CSS pixel rect, PTY 요청 크기,
PTY 관측과 event sequence, 복원 관측과 event sequence, 렌더된 frame 크기를 포함합니다.
실패는 처음 진행하지 않은 경계를 명시하며 일반 timeout만으로 판정하지 않습니다.

System command는 선언된 target shell에 맞춰 생성합니다. Windows는 `cmd.exe` command와
UTF-16LE PowerShell `-EncodedCommand` payload를 사용하고 Darwin과 Linux는 POSIX shell
문법을 사용합니다. Restore marker는 기다리는 marker 문자열이 echo되는 command 자체에
나타나지 않게 예약하며, PTY process identity는 공개 `pty.status` 계약에서만 읽습니다.

macOS native runner는 Apple Silicon에서 universal application과 7개 plugin 전체 fleet를
실행합니다. Darwin Unix socket 주소 제한 때문에 control socket은 짧은 temporary path를
사용합니다. WebView2, ConPTY, Windows named pipe 동작은 Windows native runner만 판정합니다.

Native system test는 `SOKSAK_TEST_CLI`, `SOKSAK_TEST_APP`, `SOKSAK_TEST_HOME`,
`SOKSAK_TEST_RUNTIME`, `SOKSAK_TEST_WORKSPACE`, `SOKSAK_TEST_IDENTIFIER`,
`SOKSAK_TEST_EVIDENCE`를 절대 경로 또는 명시적 identity로 받습니다. Darwin에서는
`SOKSAK_TEST_RUNTIME`에 `/private/tmp` 아래의 짧은 절대 경로를 지정합니다.

## 설치된 Vision 6-engine matrix

`verify-installed-terminal-matrix`는 이미 실행 중인 설치 제품만 공개 `sok` 명령으로 검증합니다.
component를 설치하거나 제품 source를 읽지 않습니다. 호출자가 absolute CLI와 선택적 socket,
workspace window, host pane, fresh prompt marker, 기존 non-symlink snapshot directory, 그리고 6개
provider의 정확한 version과 executable SHA-256을 선언합니다.

```sh
go run ./cmd/verify-installed-terminal-matrix \
  --cli /absolute/path/to/sok \
  --socket /absolute/path/to/control.sock \
  --window w-installed \
  --pane g-host \
  --prompt-marker 'FRESH_PROMPT_MARKER' \
  --snapshot-dir /absolute/path/to/matrix-images \
  --expect "alacritty=0.0.42@${ALACRITTY_SHA256}" \
  --expect "ghostty=0.0.40@${GHOSTTY_SHA256}" \
  --expect "kitty=0.0.36@${KITTY_SHA256}" \
  --expect "shitty=0.0.37@${SHITTY_SHA256}" \
  --expect "vt100=0.0.39@${VT100_SHA256}" \
  --expect "wezterm=0.0.38@${WEZTERM_SHA256}"
```

각 `*_SHA256` 변수는 실행 전에 예상 executable의 lowercase 64-character digest로 선언해야 하며,
snapshot directory도 미리 존재해야 합니다.

각 transaction은 Vision engine 선택, fresh `terminal-vision` tab open, `phase=live`와 prompt
marker를 함께 요구하는 event 기반 wait, view/pane/engine/failure/gap 검증, `lines` 없는 전체
prompt read, exact 4-field `sidecar.status`와
선언된 executable digest 비교를 수행합니다. 이어 layout settled와 composition zero를 확인하고
tab을 activate하거나 window focus를 바꾸지 않은 채 `window.snapshot`을 요청합니다. 출력 JSON의
PNG 경로는 사람이 직접 확인합니다. polling, sleep, Enter workaround, PATH 변경, provider별 예외는
없습니다.

## 검증 identity

GREEN은 Core commit, 이 저장소 commit, Registry asset digest, 선택한 모든 release.json digest,
OS/architecture가 같은 하나의 immutable 입력 묶음에만 유효합니다. 하나라도 바뀌면 이전
결과는 무효이며 canonical local gate부터 다시 실행합니다. 같은 묶음의 결과가 달라지면
멱등성 또는 관측 결함이며 성공할 때까지 재실행한 결과는 인정하지 않습니다. provider 예외,
timeout 증가, retry loop는 이 기준을 충족하지 못합니다.

## 빌드와 검증 명령 계약

Go 버전 정본은 `go.mod` 하나이며 owner test는 다음 명령으로 실행합니다.

```sh
make verify
SOKSAK_BENCH_REPORTS=/absolute/path/to/reports make benchmark
```

`make fleet TARGET=<target>`은 다른 host에서도 immutable artifact fleet을 정적으로 감사할 수
있습니다. 반면 `make system-commands TARGET=<native-target>`와
`make system-restore TARGET=<native-target>`는 target과 실제 Go host OS/architecture가 같아야
합니다. cross-compile된 test binary는 native runtime 증거로 인정하지 않습니다. GitHub의
Darwin/Linux/Windows reusable workflow도 private Go test 이름을 직접 호출하지 않고 이 공개 Make
시나리오만 실행합니다.

`make system-native-focus`, `make system-native-cursor`, `make system-native-keyboard`에
`TARGET=aarch64-apple-darwin`을 지정한 세 명령이 순서가 고정된 Darwin AppKit 인증 gate입니다.
WebKit은 비활성 non-key window에 native key event를 전달하지 않으므로 이 gate는 격리된 test
application을 의도적으로 활성화합니다. 사람이 사용 중인 local desktop에서는 실행하지 않고,
local capture-only 개발은 `system-theme`과 `system-performance`로 사용자 focus를 보존합니다. Gate는 공개 node 좌표에
AppKit mouse/key event를 보내고 각 terminal과 PTY까지 도달했는지 증명합니다.
`SOKSAK_TEST_CANDIDATE_PLAN`이 있으면 candidate closure를 사용하고, 없으면 Core를 통해 선언된
immutable fleet을 설치합니다.

Candidate workflow 순서는 `system-theme → system-native-focus → system-native-cursor →
system-native-keyboard → system-visibility →
system-performance → system-commands`입니다. 따라서 뒤의 성능 RED가 색상이나 native input 인증을
막지 않습니다.
