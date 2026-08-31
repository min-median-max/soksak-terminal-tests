# 로컬 candidate 구성

로컬 candidate plan은 불변 릴리즈 문서를 선택합니다. 각 Contract, Plugin, Sidecar 항목에는
`id`, `version`, `releaseSha256`가 있습니다. 구성 명령은 선택한 `release.json`, 선언된 모든 파일,
target artifact, Plugin의 contract 구현, Plugin의 전체 runtime Sidecar 목록을 검증합니다.

구성 명령은 component 소스 저장소, package registry storage, build output, checkout 상태를 읽지
않습니다. 출력의 소스 저장소와 commit 필드는 검증된 릴리즈 문서에서 가져옵니다.

세 개의 절대 경로로 저장소 명령을 실행합니다.

```sh
make compose-local-candidate-plan \
  PLAN=/absolute/local-release-selection.json \
  STORE=/absolute/local/releases \
  OUT=/absolute/local/terminal-vision-candidate-plan.json
```

artifact 참조가 release-store owner 디렉터리 안에 있도록 output도 그 디렉터리에 둡니다. 같은 명령을
다시 실행하면 `unchanged`를 반환합니다. 같은 output 경로에 다른 bytes가 있으면 실패합니다.
