package rules

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/khvedela/triage/internal/findings"
	"github.com/khvedela/triage/internal/kube"
)

func init() { Register(&podBadEnvRef{}) }

// TRG-POD-BAD-ENV-REF fires when a pod's env[].valueFrom.configMapKeyRef or
// secretKeyRef references a key that does not exist in an otherwise-present
// ConfigMap or Secret. This prevents the pod from starting with CreateContainerConfigError.
type podBadEnvRef struct{}

func (r *podBadEnvRef) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-BAD-ENV-REF",
		Title:    "Pod env var references a missing key in a ConfigMap or Secret",
		Category: findings.CategoryConfiguration,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `An env var in the pod spec uses valueFrom.configMapKeyRef or
valueFrom.secretKeyRef to reference a specific key inside a ConfigMap or Secret.
The ConfigMap/Secret exists, but the referenced key does not.

Kubernetes will refuse to start the container and report
"CreateContainerConfigError" with a message like:
  "couldn't find key KEY in ConfigMap NS/NAME"
  "couldn't find key KEY in Secret NS/NAME"

This is distinct from TRG-POD-MISSING-CONFIGMAP / TRG-POD-MISSING-SECRET, which
fire when the whole ConfigMap or Secret is absent.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#define-container-environment-variables-using-configmap-data",
		},
		Priority: 82,
	}
}

func (r *podBadEnvRef) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}

	ns := rc.Target.Namespace
	var out []findings.Finding

	allContainers := append(pod.Spec.InitContainers, pod.Spec.Containers...)
	for _, c := range allContainers {
		for _, env := range c.Env {
			if env.ValueFrom == nil {
				continue
			}
			if ref := env.ValueFrom.ConfigMapKeyRef; ref != nil {
				f := r.checkConfigMapKey(ctx, rc, pod, c.Name, env.Name, ns, ref)
				if f != nil {
					out = append(out, *f)
				}
			}
			if ref := env.ValueFrom.SecretKeyRef; ref != nil {
				f := r.checkSecretKey(ctx, rc, pod, c.Name, env.Name, ns, ref)
				if f != nil {
					out = append(out, *f)
				}
			}
		}
	}
	return out, nil
}

func (r *podBadEnvRef) checkConfigMapKey(
	ctx context.Context, rc *Context,
	pod *corev1.Pod, containerName, envName, ns string,
	ref *corev1.ConfigMapKeySelector,
) *findings.Finding {
	cm, cmFound, cmErr := rc.Cache.GetConfigMap(ctx, ns, ref.Name)
	if cmErr != nil {
		if kube.IsForbidden(cmErr) {
			return nil
		}
		return nil
	}
	if !cmFound {
		return nil // TRG-POD-MISSING-CONFIGMAP handles entirely-absent ConfigMaps
	}

	if _, ok := cm.Data[ref.Key]; ok {
		return nil
	}
	if _, ok := cm.BinaryData[ref.Key]; ok {
		return nil
	}

	// Key is missing. Check optional flag — if optional==true, Kubernetes skips it.
	if ref.Optional != nil && *ref.Optional {
		return nil
	}

	fieldPath := fmt.Sprintf("spec.containers[%s].env[%s].valueFrom.configMapKeyRef", containerName, envName)
	podNS, podName := rc.Target.Namespace, rc.Target.Name

	f := findings.Finding{
		ID:         "TRG-POD-BAD-ENV-REF",
		RuleID:     "TRG-POD-BAD-ENV-REF",
		Title:      fmt.Sprintf("Env var %q references missing key %q in ConfigMap %q", envName, ref.Key, ref.Name),
		Summary: fmt.Sprintf(
			"Container %q in pod %q/%q references ConfigMap %q key %q via env var %q, but that key does not exist in the ConfigMap. "+
				"The pod will fail to start with CreateContainerConfigError.",
			containerName, podNS, podName, ref.Name, ref.Key, envName,
		),
		Category:   findings.CategoryConfiguration,
		Severity:   findings.SeverityHigh,
		Confidence: findings.ConfidenceHigh,
		Target:     rc.Target,
		Related: []findings.ResourceRef{
			{APIVersion: "v1", Kind: "ConfigMap", Namespace: ns, Name: ref.Name},
		},
		Evidence: []findings.Evidence{
			{Kind: findings.EvidenceKindField, Source: fieldPath, Value: fmt.Sprintf("%s[%s]", ref.Name, ref.Key)},
			{Kind: findings.EvidenceKindComputed, Value: fmt.Sprintf("ConfigMap %q exists but does not contain key %q. Present keys: %s", ref.Name, ref.Key, configMapKeys(cm))},
		},
		Remediation: findings.Remediation{
			Explanation: fmt.Sprintf("The key %q does not exist in ConfigMap %q. Add it or fix the reference.", ref.Key, ref.Name),
			NextCommands: []string{
				fmt.Sprintf("kubectl get configmap -n %s %s -o yaml", ns, ref.Name),
				fmt.Sprintf("kubectl describe pod -n %s %s", podNS, podName),
			},
			SuggestedFix: fmt.Sprintf(
				"Either add the missing key to the ConfigMap:\n"+
					"  kubectl patch configmap %s -n %s --type merge -p '{\"data\":{%q:\"<value>\"}}'"+
					"\nOr update the pod spec to reference an existing key.",
				ref.Name, ns, ref.Key,
			),
			DocsLinks: []string{
				"https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#define-container-environment-variables-using-configmap-data",
			},
		},
	}
	return &f
}

func (r *podBadEnvRef) checkSecretKey(
	ctx context.Context, rc *Context,
	pod *corev1.Pod, containerName, envName, ns string,
	ref *corev1.SecretKeySelector,
) *findings.Finding {
	secret, secretFound, secretErr := rc.Cache.GetSecret(ctx, ns, ref.Name)
	if secretErr != nil {
		if kube.IsForbidden(secretErr) {
			return nil
		}
		return nil
	}
	if !secretFound {
		return nil // TRG-POD-MISSING-SECRET handles entirely-absent Secrets
	}

	if _, ok := secret.Data[ref.Key]; ok {
		return nil
	}
	if _, ok := secret.StringData[ref.Key]; ok {
		return nil
	}

	if ref.Optional != nil && *ref.Optional {
		return nil
	}

	fieldPath := fmt.Sprintf("spec.containers[%s].env[%s].valueFrom.secretKeyRef", containerName, envName)
	podNS, podName := rc.Target.Namespace, rc.Target.Name

	f := findings.Finding{
		ID:         "TRG-POD-BAD-ENV-REF",
		RuleID:     "TRG-POD-BAD-ENV-REF",
		Title:      fmt.Sprintf("Env var %q references missing key %q in Secret %q", envName, ref.Key, ref.Name),
		Summary: fmt.Sprintf(
			"Container %q in pod %q/%q references Secret %q key %q via env var %q, but that key does not exist in the Secret. "+
				"The pod will fail to start with CreateContainerConfigError.",
			containerName, podNS, podName, ref.Name, ref.Key, envName,
		),
		Category:   findings.CategoryConfiguration,
		Severity:   findings.SeverityHigh,
		Confidence: findings.ConfidenceHigh,
		Target:     rc.Target,
		Related: []findings.ResourceRef{
			{APIVersion: "v1", Kind: "Secret", Namespace: ns, Name: ref.Name},
		},
		Evidence: []findings.Evidence{
			{Kind: findings.EvidenceKindField, Source: fieldPath, Value: fmt.Sprintf("%s[%s]", ref.Name, ref.Key)},
			// Intentionally do NOT list secret keys — secrets are sensitive.
			{Kind: findings.EvidenceKindComputed, Value: fmt.Sprintf("Secret %q exists but does not contain key %q", ref.Name, ref.Key)},
		},
		Remediation: findings.Remediation{
			Explanation: fmt.Sprintf("The key %q does not exist in Secret %q. Add it or fix the reference.", ref.Key, ref.Name),
			NextCommands: []string{
				fmt.Sprintf("kubectl get secret -n %s %s -o jsonpath='{.data}' | python3 -c \"import sys,json; d=json.load(sys.stdin); print(list(d.keys()))\"", ns, ref.Name),
				fmt.Sprintf("kubectl describe pod -n %s %s", podNS, podName),
			},
			SuggestedFix: fmt.Sprintf(
				"Add the missing key to the Secret:\n"+
					"  kubectl patch secret %s -n %s --type merge -p '{\"stringData\":{%q:\"<value>\"}}'"+
					"\nOr update the pod spec to reference an existing key.",
				ref.Name, ns, ref.Key,
			),
			DocsLinks: []string{
				"https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-environment-variables",
			},
		},
	}
	return &f
}

func configMapKeys(cm *corev1.ConfigMap) string {
	keys := make([]string, 0, len(cm.Data)+len(cm.BinaryData))
	for k := range cm.Data {
		keys = append(keys, k)
	}
	for k := range cm.BinaryData {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return "(empty)"
	}
	result := ""
	for i, k := range keys {
		if i > 0 {
			result += ", "
		}
		result += k
	}
	return result
}
