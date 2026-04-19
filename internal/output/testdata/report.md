# Kubediag report — Pod default/crashloop-demo

_Generated at 2024-01-15 10:30:00 UTC (42ms)_

**Overall severity:** `critical`  
**Findings:** 2

## Contents

1. [Container "app" is in CrashLoopBackOff (5 restarts)](#1-container-app-is-in-crashloopbackoff-5-restarts)
2. [Container "app" was OOMKilled](#2-container-app-was-oomkilled)

## Summary

| # | Severity | Confidence | Rule | Title |
|---|----------|-----------|------|-------|
| 1 | critical | high | `TRG-POD-CRASHLOOPBACKOFF` | Container "app" is in CrashLoopBackOff (5 restarts) |
| 2 | high | high | `TRG-POD-OOMKILLED` | Container "app" was OOMKilled |

## Findings

## 1. Container "app" is in CrashLoopBackOff (5 restarts)

- **Rule:** `TRG-POD-CRASHLOOPBACKOFF`
- **Severity:** `critical`
- **Confidence:** `high`
- **Category:** `Runtime`
- **Target:** `pod/default/crashloop-demo`

Container "app" has crashed 5 times. The kubelet is backing off restarts.

### Evidence
- **Field** pod.status.containerStatuses[app].state.waiting.reason = CrashLoopBackOff
- **Field** pod.status.containerStatuses[app].restartCount = 5

### Next commands
```sh
kubectl logs -n default crashloop-demo -c app --previous
kubectl describe pod -n default crashloop-demo
```

## 2. Container "app" was OOMKilled

- **Rule:** `TRG-POD-OOMKILLED`
- **Severity:** `high`
- **Confidence:** `high`
- **Category:** `ResourcePressure`
- **Target:** `pod/default/crashloop-demo`

Container "app" was killed by the kernel OOM killer.

### Evidence
- **Field** pod.status.containerStatuses[app].lastState.terminated.reason = OOMKilled

### Next commands
```sh
kubectl top pod -n default crashloop-demo --containers
```

### Suggested fix
Increase resources.limits.memory.

