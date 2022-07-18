package yacht

import (
	"time"

	"k8s.io/client-go/tools/cache"
	rl "k8s.io/client-go/tools/leaderelection/resourcelock"
)

type Interface interface {
	Enqueue(obj interface{})
	WithEnqueueFunc(EnqueueFunc) *Controller
	WithHandlerFunc(HandlerFunc) *Controller
	WithLeaderElection(leaseLock rl.Interface, leaseDuration, renewDeadline, retryPeriod time.Duration) *Controller
	WithCacheSynced(...cache.InformerSynced) *Controller
}

type HandlerFunc func(key interface{}) (requeueAfter *time.Duration, err error)

type EnqueueFunc func(obj interface{}) (interface{}, error)
