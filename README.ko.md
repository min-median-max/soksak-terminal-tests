# soksak-terminal-tests

Soksak이 사용하는 터미널 구현을 검증하는 비제품 품질 저장소입니다. 설치 대상이나
registry 항목이 아닙니다. benchmark는 복원 구현의 성능·RSS·손실을 비교하고 system은 공개
명령으로 release component를 설치한 뒤 복원 시나리오를 검증합니다.

플러그인과 사이드카 저장소는 각자의 conformance와 release gate를 소유합니다. 이 저장소는
설치 artifact, 공개 command, contract report만 사용하며 다른 저장소의 source를 빌드하지
않습니다.

Darwin과 Linux는 plugin 7개, PTY sidecar, recovery sidecar 6개를 사용합니다. Windows는
Alacritty, Ghostty, VT100, WezTerm, Xterm plugin과 PTY 및 recovery sidecar 4개를 사용합니다.
`verify-fleet`가 immutable release byte를 먼저 검증합니다. System suite는 빈 home으로 Core를
시작하고 `plugin.install`에 `pty`와 `recovery` binding을 명시합니다. Consent summary를 읽고
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
