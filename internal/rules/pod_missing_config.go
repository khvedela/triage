package rules

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/OWNER/triage/internal/findings"
	"github.com/OWNER/triage/internal/kube"
)

func init() {
	Register(&podMissingConfigMap{})
	Register(&podMissingSecret{})
}

// ----- TRG-POD-MISSING-CONFIGMAP ------------------------------------------

type podMissingConfigMap struct{}

func (r *podMissingConfigMap) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-MISSING-CONFIGMAP",
		Title:    "Pod references a ConfigMap that does not exist",
		Category: findings.CategoryConfiguration,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `The pod spec references a ConfigMap (in a volume, envFrom, or env valueFrom)
that does not exist in the same namespace. The pod will stay in ContainerCreating
or Pending until the ConfigMap is created.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/configuration/configmap/",
		},
		Priority: 80,
	}
}

func (r *podMissingConfigMap) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	ns := rc.Target.Namespace
	var out []findings.Finding

	// Collect all referenced ConfigMap names from the pod spec.
	refs := configMapRefs(pod)
	seen := map[string]bool{}

	for _, ref := range refs {
		if seen[ref.name] {
			continue
		}
		seen[ref.name] = true

		cm, cmFound, cmErr := rc.Cache.GetConfigMap(ctx, ns, ref.name)
		if cmErr != nil {
			if kube.IsForbidden(cmErr) {
				continue // RBAC — already handled by TRG-ACCESS-INSUFFICIENT-READ
			}
			continue
		}
		if cmFound || cm != nil {
			continue // exists — no finding
		}

		out = append(out, findings.Finding{
			ID:         "TRG-POD-MISSING-CONFIGMAP",
			RuleID:     "TRG-POD-MISSING-CONFIGMAP",
			Title:      fmt.Sprintf("ConfigMap %q referenced by pod does not exist", ref.name),
			Summary:    fmt.Sprintf("Pod %q references ConfigMap %q (in %s) but it does not exist in namespace %q.", pod.Name, ref.name, ref.where, ns),
			Category:   findings.CategoryConfiguration,
			Severity:   findings.SeverityHigh,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Evidence: []findings.Evidence{
				{Kind: findings.EvidenceKindField, Source: ref.where, Value: ref.name},
				{Kind: findings.EvidenceKindComputed, Value: fmt.Sprintf("kubectl get configmap -n %s %s → NotFound", ns, ref.name)},
			},
			Remediation: findings.Remediation{
				Explanation: "Create the missing ConfigMap or correct the reference in the pod spec.",
				NextCommands: []string{
					fmt.Sprintf("kubectl get configmap -n %s", ns),
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, pod.Name),
				},
				SuggestedFix: fmt.Sprintf("Create the ConfigMap:\n"+
					"  kubectl create configmap %s -n %s --from-literal=key=value\n"+
					"Or fix the reference in the pod/deployment spec.", ref.name, ns),
				DocsLinks: []string{"https://kubernetes.io/docs/concepts/configuration/configmap/"},
			},
		})
	}
	return out, nil
}

// ----- TRG-POD-MISSING-SECRET --------------------------------------------

type podMissingSecret struct{}

func (r *podMissingSecret) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-MISSING-SECRET",
		Title:    "Pod references a Secret that does not exist",
		Category: findings.CategoryConfiguration,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `The pod spec references a Secret (in a volume, envFrom, env valueFrom, or
imagePullSecrets) that does not exist in the same namespace. The pod will remain
in ContainerCreating or Pending until the Secret is created.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/configuration/secret/",
		},
		Priority: 81,
	}
}

func (r *podMissingSecret) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	ns := rc.Target.Namespace
	var out []findings.Finding

	refs := secretRefs(pod)
	seen := map[string]bool{}

	for _, ref := range refs {
		if seen[ref.name] {
			continue
		}
		seen[ref.name] = true

		_, secretFound, secretErr := rc.Cache.GetSecret(ctx, ns, ref.name)
		if secretErr != nil {
			if kube.IsForbidden(secretErr) {
				continue
			}
			continue
		}
		if secretFound {
			continue
		}

		out = append(out, findings.Finding{
			ID:         "TRG-POD-MISSING-SECRET",
			RuleID:     "TRG-POD-MISSING-SECRET",
			Title:      fmt.Sprintf("Secret %q referenced by pod does not exist", ref.name),
			Summary:    fmt.Sprintf("Pod %q references Secret %q (in %s) but it does not exist in namespace %q.", pod.Name, ref.name, ref.where, ns),
			Category:   findings.CategoryConfiguration,
			Severity:   findings.SeverityHigh,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Evidence: []findings.Evidence{
				{Kind: findings.EvidenceKindField, Source: ref.where, Value: ref.name},
				{Kind: findings.EvidenceKindComputed, Value: fmt.Sprintf("kubectl get secret -n %s %s → NotFound", ns, ref.name)},
			},
			Remediation: findings.Remediation{
				Explanation: "Create the missing Secret or correct the reference in the pod spec.",
				NextCommands: []string{
					fmt.Sprintf("kubectl get secret -n %s", ns),
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, pod.Name),
				},
				SuggestedFix: fmt.Sprintf("Create the Secret:\n"+
					"  kubectl create secret generic %s -n %s --from-literal=key=value\n"+
					"Or fix the reference in the pod/deployment spec.", ref.name, ns),
				DocsLinks: []string{"https://kubernetes.io/docs/concepts/configuration/secret/"},
			},
		})
	}
	return out, nil
}

// ----- spec reference extractors ------------------------------------------

type specRef struct {
	name  string
	where string // human-readable field path for evidence
}

func configMapRefs(pod *corev1.Pod) []specRef {
	var refs []specRef

	// Volumes
	for _, v := range pod.Spec.Volumes {
		if v.ConfigMap != nil {
			refs = append(refs, specRef{v.ConfigMap.Name, fmt.Sprintf("spec.volumes[%s].configMap.name", v.Name)})
		}
		if v.Projected != nil {
			for _, src := range v.Projected.Sources {
				if src.ConfigMap != nil {
					refs = append(refs, specRef{src.ConfigMap.Name, fmt.Sprintf("spec.volumes[%s].projected.sources[].configMap.name", v.Name)})
				}
			}
		}
	}

	// envFrom / env in all containers
	allContainers := append(pod.Spec.InitContainers, pod.Spec.Containers...)
	for _, c := range allContainers {
		for _, ef := range c.EnvFrom {
			if ef.ConfigMapRef != nil {
				refs = append(refs, specRef{ef.ConfigMapRef.Name, fmt.Sprintf("spec.containers[%s].envFrom[].configMapRef.name", c.Name)})
			}
		}
		for _, env := range c.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
				refs = append(refs, specRef{env.ValueFrom.ConfigMapKeyRef.Name, fmt.Sprintf("spec.containers[%s].env[%s].valueFrom.configMapKeyRef.name", c.Name, env.Name)})
			}
		}
	}

	return refs
}

func secretRefs(pod *corev1.Pod) []specRef {
	var refs []specRef

	// imagePullSecrets
	for _, s := range pod.Spec.ImagePullSecrets {
		refs = append(refs, specRef{s.Name, "spec.imagePullSecrets[].name"})
	}

	// Volumes
	for _, v := range pod.Spec.Volumes {
		if v.Secret != nil {
			refs = append(refs, specRef{v.Secret.SecretName, fmt.Sprintf("spec.volumes[%s].secret.secretName", v.Name)})
		}
		if v.Projected != nil {
			for _, src := range v.Projected.Sources {
				if src.Secret != nil {
					refs = append(refs, specRef{src.Secret.Name, fmt.Sprintf("spec.volumes[%s].projected.sources[].secret.name", v.Name)})
				}
			}
		}
	}

	// envFrom / env
	allContainers := append(pod.Spec.InitContainers, pod.Spec.Containers...)
	for _, c := range allContainers {
		for _, ef := range c.EnvFrom {
			if ef.SecretRef != nil {
				refs = append(refs, specRef{ef.SecretRef.Name, fmt.Sprintf("spec.containers[%s].envFrom[].secretRef.name", c.Name)})
			}
		}
		for _, env := range c.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				refs = append(refs, specRef{env.ValueFrom.SecretKeyRef.Name, fmt.Sprintf("spec.containers[%s].env[%s].valueFrom.secretKeyRef.name", c.Name, env.Name)})
			}
		}
	}

	// TLS secrets (not explicitly in pod spec, but commonly referenced via service account tokens)
	if pod.Spec.ServiceAccountName != "" {
		// Don't check the service account token secret; it's auto-created.
		_ = pod.Spec.ServiceAccountName
	}

	return refs
}

