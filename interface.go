package yacht

import (
	"context"
	"time"

	"k8s.io/client-go/tools/cache"
	rl "k8s.io/client-go/tools/leaderelection/resourcelock"
)

type Interface interface {
	Enqueue(obj interface{})
	WithEnqueueFunc(EnqueueFunc) *Controller
	// Deprecated: Use WithHandlerContextFunc instead.
	WithHandlerFunc(HandlerFunc) *Controller
	WithHandlerContextFunc(HandlerContextFunc) *Controller
	WithLeaderElection(leaseLock rl.Interface, leaseDuration, renewDeadline, retryPeriod time.Duration) *Controller
	WithCacheSynced(...cache.InformerSynced) *Controller
}

// Deprecated: Use HandlerContextFunc instead.
type HandlerFunc func(key interface{}) (requeueAfter *time.Duration, err error)

type HandlerContextFunc func(ctx context.Context, key interface{}) (requeueAfter *time.Duration, err error)

type EnqueueFunc func(obj interface{}) (interface{}, error)

type EnqueueFilterFunc func(oldObj, newObj interface{}) (bool, error)
