export type ScenarioOutput = {
  text: string;
  markdown: string;
  json: string;
};

export type Scenario = {
  slug: string;
  name: string;
  headline: string;
  summary: string;
  primaryRule: string;
  severity: "critical" | "high" | "medium" | "info";
  confidence: "high" | "medium" | "low";
  yaml: string;
  signals: string[];
  whyItWins: string[];
  nextCommands: string[];
  outputs: ScenarioOutput;
};

export const scenarios: Scenario[] = [
  {
    slug: "crashloop",
    name: "CrashLoopBackOff",
    headline: "A pod restarts fast enough that the incident starts as noise and becomes an outage.",
    summary:
      "kubediag promotes the runtime failure to the top, preserves the raw evidence, and gives the exact commands needed to confirm the crash cause.",
    primaryRule: "TRG-POD-CRASHLOOPBACKOFF",
    severity: "critical",
    confidence: "high",
    yaml: `apiVersion: v1
kind: Pod
metadata:
  name: crashloop-demo
  namespace: default
  labels:
    app: crashloop-demo
spec:
  containers:
    - name: app
      image: busybox:1.36
      command: ["sh", "-c", "echo 'starting...'; sleep 1; exit 1"]
      resources:
        requests:
          cpu: 10m
          memory: 16Mi
        limits:
          cpu: 100m
          memory: 64Mi`,
    signals: [
      "pod.status.containerStatuses[app].state.waiting.reason = CrashLoopBackOff",
      "pod.status.containerStatuses[app].restartCount = 5",
      "Warning BackOff: back-off restarting failed container",
      "Last 3 log lines show the process exits almost immediately"
    ],
    whyItWins: [
      "The waiting reason is explicit and high-confidence; kubediag can name the runtime symptom without guessing.",
      "Restart count confirms this is not a one-off crash but a repeating failure pattern.",
      "Related findings, like OOMKilled, stay visible but ranked below the primary incident driver."
    ],
    nextCommands: [
      "kubectl logs -n default crashloop-demo -c app --previous",
      "kubectl describe pod -n default crashloop-demo"
    ],
    outputs: {
      text: `▶ Pod default/crashloop-demo
  generated: 10:30:00 UTC  (42ms)
  overall: CRITICAL
  findings: 2 (1 critical, 1 high)

ⓧ CRITICAL  [high confidence]  TRG-POD-CRASHLOOPBACKOFF
   Container "app" is in CrashLoopBackOff (5 restarts)
   Container "app" has crashed 5 times. The kubelet is backing off restarts.

   Evidence:
     • pod.status.containerStatuses[app].state.waiting.reason = CrashLoopBackOff
     • pod.status.containerStatuses[app].restartCount = 5

   Next commands:
     $ kubectl logs -n default crashloop-demo -c app --previous
     $ kubectl describe pod -n default crashloop-demo

● HIGH      [high confidence]  TRG-POD-OOMKILLED
   Container "app" was OOMKilled
   Container "app" was killed by the kernel OOM killer.`,
      markdown: `# Kubediag report — Pod default/crashloop-demo

_Generated at 2024-01-15 10:30:00 UTC (42ms)_

**Overall severity:** \`critical\`  
**Findings:** 2

| # | Severity | Confidence | Rule | Title |
|---|----------|-----------|------|-------|
| 1 | critical | high | \`TRG-POD-CRASHLOOPBACKOFF\` | Container "app" is in CrashLoopBackOff (5 restarts) |
| 2 | high | high | \`TRG-POD-OOMKILLED\` | Container "app" was OOMKilled |

## 1. Container "app" is in CrashLoopBackOff (5 restarts)

- **Rule:** \`TRG-POD-CRASHLOOPBACKOFF\`
- **Severity:** \`critical\`
- **Confidence:** \`high\`
- **Category:** \`Runtime\`

Container "app" has crashed 5 times. The kubelet is backing off restarts.`,
      json: `{
  "target": { "kind": "Pod", "namespace": "default", "name": "crashloop-demo" },
  "generatedAt": "2024-01-15T10:30:00Z",
  "durationMs": 42,
  "findings": [
    {
      "ruleId": "TRG-POD-CRASHLOOPBACKOFF",
      "title": "Container \\"app\\" is in CrashLoopBackOff (5 restarts)",
      "severity": "critical",
      "confidence": "high"
    },
    {
      "ruleId": "TRG-POD-OOMKILLED",
      "title": "Container \\"app\\" was OOMKilled",
      "severity": "high",
      "confidence": "high"
    }
  ]
}`
    }
  },
  {
    slug: "missing-configmap",
    name: "Missing ConfigMap",
    headline: "A pod references config that was never created, so it never becomes runnable.",
    summary:
      "kubediag turns a generic startup failure into a concrete configuration diagnosis tied to the missing object name and the affected reference.",
    primaryRule: "TRG-POD-MISSING-CONFIGMAP",
    severity: "high",
    confidence: "high",
    yaml: `apiVersion: v1
kind: Pod
metadata:
  name: bad-configmap-demo
  namespace: default
spec:
  containers:
    - name: app
      image: busybox:1.36
      command: ["sh", "-c", "echo $MY_KEY; sleep 3600"]
      env:
        - name: MY_KEY
          valueFrom:
            configMapKeyRef:
              name: does-not-exist
              key: some-key
      resources:
        requests:
          cpu: 10m
          memory: 16Mi
        limits:
          cpu: 100m
          memory: 64Mi`,
    signals: [
      "spec.containers[app].env[MY_KEY].valueFrom.configMapKeyRef.name = does-not-exist",
      "ConfigMap default/does-not-exist was not found",
      "Pod remains Pending / ContainerCreating waiting on referenced config"
    ],
    whyItWins: [
      "The pod spec directly points to a named ConfigMap, so kubediag can verify existence instead of inferring from a vague event string.",
      "The missing object name becomes part of the diagnosis, which shortens the path from symptom to fix.",
      "This finding outranks generic startup symptoms because the broken dependency is explicit."
    ],
    nextCommands: [
      "kubectl get configmap -n default does-not-exist",
      "kubectl describe pod -n default bad-configmap-demo"
    ],
    outputs: {
      text: `▶ Pod default/bad-configmap-demo
  generated: 10:31:12 UTC  (31ms)
  overall: HIGH
  findings: 1 (1 high)

● HIGH      [high confidence]  TRG-POD-MISSING-CONFIGMAP
   Pod references a ConfigMap that does not exist
   The pod references ConfigMap "does-not-exist" in env valueFrom.

   Evidence:
     • spec.containers[app].env[MY_KEY].valueFrom.configMapKeyRef.name = does-not-exist
     • configmap default/does-not-exist = not found

   Next commands:
     $ kubectl get configmap -n default does-not-exist
     $ kubectl describe pod -n default bad-configmap-demo

   Suggested fix:
     Create the ConfigMap or update the pod spec to reference the correct object.`,
      markdown: `# Kubediag report — Pod default/bad-configmap-demo

**Overall severity:** \`high\`

## Pod references a ConfigMap that does not exist

- **Rule:** \`TRG-POD-MISSING-CONFIGMAP\`
- **Severity:** \`high\`
- **Confidence:** \`high\`

The pod references ConfigMap \`does-not-exist\` in env valueFrom.`,
      json: `{
  "target": { "kind": "Pod", "namespace": "default", "name": "bad-configmap-demo" },
  "findings": [
    {
      "ruleId": "TRG-POD-MISSING-CONFIGMAP",
      "severity": "high",
      "confidence": "high",
      "evidence": [
        "spec.containers[app].env[MY_KEY].valueFrom.configMapKeyRef.name = does-not-exist",
        "configmap default/does-not-exist = not found"
      ]
    }
  ]
}`
    }
  },
  {
    slug: "stuck-rollout",
    name: "Stuck rollout",
    headline: "A deployment keeps its old replicas because the new image cannot be pulled.",
    summary:
      "kubediag surfaces both the rollout-level failure and the underlying image issue so operators do not have to correlate controller and pod symptoms manually.",
    primaryRule: "TRG-DEPLOY-ROLLOUT-STUCK",
    severity: "critical",
    confidence: "high",
    yaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: stuck-rollout-demo
  namespace: default
spec:
  replicas: 3
  progressDeadlineSeconds: 30
  selector:
    matchLabels:
      app: stuck-rollout-demo
  template:
    metadata:
      labels:
        app: stuck-rollout-demo
    spec:
      containers:
        - name: app
          image: nginx:this-tag-does-not-exist
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
            limits:
              cpu: 100m
              memory: 64Mi`,
    signals: [
      "deployment.status.conditions[Progressing].reason = ProgressDeadlineExceeded",
      "New ReplicaSet pods emit ErrImagePull / ImagePullBackOff",
      "Registry returns manifest unknown for tag this-tag-does-not-exist"
    ],
    whyItWins: [
      "The deployment condition proves the rollout exceeded its progress deadline, which makes this a controller-level incident.",
      "kubediag keeps the image pull error attached as supporting evidence so the underlying fix stays obvious.",
      "Operators can see both what is broken at the rollout layer and why it is broken at the pod layer."
    ],
    nextCommands: [
      "kubectl describe deployment -n default stuck-rollout-demo",
      "kubectl describe pod -n default -l app=stuck-rollout-demo",
      "kubectl set image deployment/stuck-rollout-demo app=nginx:stable -n default"
    ],
    outputs: {
      text: `▶ Deployment default/stuck-rollout-demo
  generated: 10:33:09 UTC  (57ms)
  overall: CRITICAL
  findings: 2 (1 critical, 1 high)

ⓧ CRITICAL  [high confidence]  TRG-DEPLOY-ROLLOUT-STUCK
   Deployment rollout has exceeded its progress deadline
   The deployment has not made progress before progressDeadlineSeconds elapsed.

   Evidence:
     • deployment.status.conditions[Progressing].reason = ProgressDeadlineExceeded

● HIGH      [high confidence]  TRG-POD-IMAGE-NOT-FOUND
   Container image tag or repository does not exist
   The registry returned manifest not found for image nginx:this-tag-does-not-exist.`,
      markdown: `# Kubediag report — Deployment default/stuck-rollout-demo

**Overall severity:** \`critical\`

## Deployment rollout has exceeded its progress deadline

- **Rule:** \`TRG-DEPLOY-ROLLOUT-STUCK\`
- **Severity:** \`critical\`
- **Confidence:** \`high\`

The deployment has not made progress before \`progressDeadlineSeconds\` elapsed.`,
      json: `{
  "target": { "kind": "Deployment", "namespace": "default", "name": "stuck-rollout-demo" },
  "findings": [
    {
      "ruleId": "TRG-DEPLOY-ROLLOUT-STUCK",
      "severity": "critical",
      "confidence": "high"
    },
    {
      "ruleId": "TRG-POD-IMAGE-NOT-FOUND",
      "severity": "high",
      "confidence": "high"
    }
  ]
}`
    }
  },
  {
    slug: "oomkilled",
    name: "OOMKilled",
    headline: "The process survives long enough to start, then crosses its memory limit and gets killed.",
    summary:
      "kubediag distinguishes resource pressure from generic crash behavior and includes the configured memory limit directly in the evidence trail.",
    primaryRule: "TRG-POD-OOMKILLED",
    severity: "high",
    confidence: "high",
    yaml: `apiVersion: v1
kind: Pod
metadata:
  name: oom-pod
  namespace: default
spec:
  containers:
    - name: app
      image: ghcr.io/khvedela/memory-demo:latest
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
        limits:
          cpu: 200m
          memory: 256Mi`,
    signals: [
      "pod.status.containerStatuses[app].lastState.terminated.reason = OOMKilled",
      "resources.limits.memory = 256Mi",
      "Restart count increases after high memory consumption spikes"
    ],
    whyItWins: [
      "OOMKilled is an explicit Kubernetes termination reason, so the diagnosis can be high confidence without heuristic ranking.",
      "Including the memory limit in evidence makes the remediation concrete instead of generic.",
      "The rule stays separate from CrashLoopBackOff so capacity incidents are easy to spot in fleet scans."
    ],
    nextCommands: [
      "kubectl top pod -n default oom-pod --containers",
      "kubectl describe pod -n default oom-pod"
    ],
    outputs: {
      text: `▶ Pod default/oom-pod
  generated: 10:34:44 UTC  (24ms)
  overall: HIGH
  findings: 1 (1 high)

● HIGH      [high confidence]  TRG-POD-OOMKILLED
   Container "app" was OOMKilled
   Container "app" was killed by the kernel OOM killer.

   Evidence:
     • pod.status.containerStatuses[app].lastState.terminated.reason = OOMKilled
     • spec.containers[app].resources.limits.memory = 256Mi

   Next commands:
     $ kubectl top pod -n default oom-pod --containers
     $ kubectl describe pod -n default oom-pod

   Suggested fix:
     Increase resources.limits.memory.`,
      markdown: `# Kubediag report — Pod default/oom-pod

**Overall severity:** \`high\`

## Container "app" was OOMKilled

- **Rule:** \`TRG-POD-OOMKILLED\`
- **Severity:** \`high\`
- **Confidence:** \`high\`

The container exceeded its memory limit and was killed by the kernel OOM killer.`,
      json: `{
  "target": { "kind": "Pod", "namespace": "default", "name": "oom-pod" },
  "findings": [
    {
      "ruleId": "TRG-POD-OOMKILLED",
      "severity": "high",
      "confidence": "high",
      "evidence": [
        "pod.status.containerStatuses[app].lastState.terminated.reason = OOMKilled",
        "spec.containers[app].resources.limits.memory = 256Mi"
      ]
    }
  ]
}`
    }
  }
];
