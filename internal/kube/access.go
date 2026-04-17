package kube

import (
	"context"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CanI performs a SelfSubjectAccessReview to determine whether the current
// user can perform `verb` on `group/resource` in `namespace`.
func (c *clientgoClient) CanI(ctx context.Context, verb, group, resource, namespace string) (bool, error) {
	r, err := c.cs.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Verb:      verb,
				Group:     group,
				Resource:  resource,
				Namespace: namespace,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return r.Status.Allowed, nil
}
