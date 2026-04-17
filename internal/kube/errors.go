package kube

import (
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// ErrForbidden is returned by cache accessors when the API server denied
// the read due to RBAC. Distinct from "not found" so that rules can emit
// a TRG-ACCESS-INSUFFICIENT-READ finding instead of a false negative.
var ErrForbidden = errors.New("forbidden: RBAC denied access")

// IsForbidden reports whether err is a Kubernetes "Forbidden" error or our
// own ErrForbidden sentinel.
func IsForbidden(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrForbidden) {
		return true
	}
	return apierrors.IsForbidden(err)
}

// IsNotFound reports whether err is a Kubernetes "NotFound" error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return apierrors.IsNotFound(err)
}
