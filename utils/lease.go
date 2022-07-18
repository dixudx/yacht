package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coordinationv1client "k8s.io/client-go/kubernetes/typed/coordination/v1"
	rl "k8s.io/client-go/tools/leaderelection/resourcelock"
)

// NewLeaseLock creates a LeaseLock
func NewLeaseLock(leaseName, leaseNamespace, identity string, client coordinationv1client.LeasesGetter) rl.Interface {
	return &rl.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: leaseNamespace,
		},
		Client: client,
		LockConfig: rl.ResourceLockConfig{
			Identity: identity,
		},
	}
}
