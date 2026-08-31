# 시스템 테스트 수명주기

시스템 테스트는 두 종료 동작을 구분합니다.

- `Shutdown`은 일반 애플리케이션 종료를 요청합니다. 제품은 재시작 가능한 Sidecar 프로세스를 유지할 수 있습니다.
- `Finish`는 테스트 리소스를 종료합니다. Plugin 탭을 닫고, 활성 Plugin을 비활성화하고,
  `sidecar_status`가 반환한 모든 Sidecar에 `sidecar_stop`을 호출합니다.

`Finish`는 각 종료 응답에 `running:false`가 있는지 확인합니다. 이후 `sidecar_status`의
`open`과 `recorded`가 모두 비어 있는지 확인하고, 초기 상태의 모든 PID가 더 이상 존재하지 않는지
확인한 다음 애플리케이션을 종료합니다. PID는 한 번 확인합니다. `sidecar_stop`이 운영체제의 프로세스
종료 알림을 이미 기다립니다. 시스템 테스트는 완료 시 `Finish`를 호출해야 합니다.

수명주기 실행 결과는 선언된 증거 디렉터리의 `process-window-ownership.json`에 기록됩니다.
