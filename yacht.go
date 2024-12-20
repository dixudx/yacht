package yacht

import (
	"context"
	"fmt"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	rl "k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	utilpointer "k8s.io/utils/pointer"

	"github.com/dixudx/yacht/utils"
)

type Controller struct {
	// name is the name of this controller
	name string
	// workers indicates the number of workers
	workers *int
	// enqueueFunc defines the function to enqueue the work item
	enqueueFunc EnqueueFunc
	// enqueueFilterFunc defines the filter function before enqueueing the work item
	enqueueFilterFunc EnqueueFilterFunc
	// queue is a rate limited work queue.
	queue workqueue.RateLimitingInterface
	// informersSynced records a group of cacheSyncs
	// The workers will not start working before all the caches are synced successfully
	informersSynced []cache.InformerSynced
	// handlerContextFunc defines the handler to process the work item
	handlerContextFunc HandlerContextFunc
	// le specifies the LeaderElector to use
	le *leaderelection.LeaderElector

	// runFlag indicates whether the workers start working
	runFlag bool

	once sync.Once
}

var _ Interface = &Controller{}

// NewController creates a new Controller
func NewController(name string) *Controller {
	return &Controller{
		name:        name,
		workers:     utilpointer.Int(2),
		enqueueFunc: DefaultEnqueueFunc,
		queue: workqueue.NewRateLimitingQueueWithConfig(
			workqueue.DefaultControllerRateLimiter(),
			workqueue.RateLimitingQueueConfig{
				Name: name,
			}),
		informersSynced: []cache.InformerSynced{},
	}
}

// WithWorkers sets the number of workers to process work items off work queue
func (c *Controller) WithWorkers(workers int) *Controller {
	if c.runFlag {
		panic(fmt.Errorf("can not mutate workers when controller %s is running", c.name))
	}
	if workers < 0 {
		panic(fmt.Errorf("can not set negative workers %d", workers))
	}

	c.workers = utilpointer.Int(workers)
	return c
}

// WithQueue replaces the default queue with the desired one to store work items.
func (c *Controller) WithQueue(queue workqueue.RateLimitingInterface) *Controller {
	if c.runFlag {
		panic(fmt.Errorf("can not mutate queue when controller %s is running", c.name))
	}

	c.queue = queue
	return c
}

// WithEnqueueFilterFunc sets customize enqueueFilterFunc
func (c *Controller) WithEnqueueFilterFunc(enqueueFilterFunc EnqueueFilterFunc) *Controller {
	if c.runFlag {
		panic(fmt.Errorf("can not mutate enqueueFilterFunc when controller %s is running", c.name))
	}

	c.enqueueFilterFunc = enqueueFilterFunc
	return c
}

// WithEnqueueFunc sets customize enqueueFunc
func (c *Controller) WithEnqueueFunc(enqueueFunc EnqueueFunc) *Controller {
	if c.runFlag {
		panic(fmt.Errorf("can not mutate enqueueFunc when controller %s is running", c.name))
	}

	if enqueueFunc != nil {
		c.enqueueFunc = enqueueFunc
	}
	return c
}

func (c *Controller) DefaultResourceEventHandlerFuncs() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if c.applyEnqueueFilterFunc(nil, obj, cache.Added) {
				c.Enqueue(obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if c.applyEnqueueFilterFunc(oldObj, newObj, cache.Updated) {
				c.Enqueue(newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if c.applyEnqueueFilterFunc(obj, nil, cache.Deleted) {
				c.Enqueue(obj)
			}
		},
	}
}

func (c *Controller) applyEnqueueFilterFunc(oldObj, newObj interface{}, operation cache.DeltaType) bool {
	if c.enqueueFilterFunc == nil {
		obj := oldObj
		if obj == nil {
			obj = newObj
		}
		utils.DepthLogging(nil, "info", fmt.Sprintf("[%s] enqueue resource", operation), obj)
		return true
	}

	var err error
	var ok bool
	switch operation {
	case cache.Added:
		ok, err = c.enqueueFilterFunc(nil, newObj)
	case cache.Deleted:
		ok, err = c.enqueueFilterFunc(oldObj, nil)
	case cache.Updated:
		ok, err = c.enqueueFilterFunc(oldObj, newObj)
	default:
		utils.DepthLogging(nil, "error", fmt.Sprintf("[%s] unexpected resource event type", operation), oldObj)
		return false
	}

	if err != nil {
		utils.DepthLogging(err, "error", fmt.Sprintf("[%s] failed to apply enqueueFilterFunc", operation), oldObj)
		return false
	}

	if !ok {
		utils.DepthLogging(nil, "warning", fmt.Sprintf("[%s] not enqueue resource", operation), oldObj)
		return false
	}

	if operation == cache.Deleted {
		utils.DepthLogging(nil, "info", fmt.Sprintf("[%s] enqueue resource", operation), oldObj)
	} else {
		utils.DepthLogging(nil, "info", fmt.Sprintf("[%s] enqueue resource", operation), newObj)
	}
	return true
}

// WithHandlerFunc sets a handler function to process the work item off the work queue
// Deprecated: Use WithHandlerContextFunc instead.
func (c *Controller) WithHandlerFunc(handlerFunc HandlerFunc) *Controller {
	if c.runFlag {
		panic(fmt.Errorf("can not mutate handlerContextFunc when controller %s is running", c.name))
	}

	if handlerFunc != nil {
		c.handlerContextFunc = func(ctx context.Context, key interface{}) (requeueAfter *time.Duration, err error) {
			select {
			case <-ctx.Done():
				return
			default:
				return handlerFunc(key)
			}
		}
	}
	return c
}

// WithHandlerContextFunc sets a handler function to process the work item off the work queue
func (c *Controller) WithHandlerContextFunc(handlerContextFunc HandlerContextFunc) *Controller {
	if c.runFlag {
		panic(fmt.Errorf("can not mutate handlerContextFunc when controller %s is running", c.name))
	}

	if handlerContextFunc != nil {
		c.handlerContextFunc = handlerContextFunc
	}
	return c
}

// WithLeaderElection uses leader election to get the lock
func (c *Controller) WithLeaderElection(leaseLock rl.Interface, leaseDuration, renewDeadline, retryPeriod time.Duration) *Controller {
	if c.runFlag {
		panic(fmt.Errorf("can not mutate leaderElection when controller %s is running", c.name))
	}

	lec := leaderelection.LeaderElectionConfig{
		Lock: leaseLock,
		// IMPORTANT: you MUST ensure that any code you have that is protected by the lease must terminate **before**
		// you call cancel. Otherwise, you could have a background loop still running and another process could
		// get elected before your background loop finished, violating the stated goal of the lease.
		ReleaseOnCancel: true,
		LeaseDuration:   leaseDuration,
		RenewDeadline:   renewDeadline,
		RetryPeriod:     retryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				c.run(ctx)
			},
			OnStoppedLeading: func() {
				klog.Errorf("leader election got lost for controller %s", c.name)
			},
			OnNewLeader: func(identity string) {
				// gets notified when new leader is elected
				if identity == leaseLock.Identity() {
					// I just got the lock
					return
				}
				klog.Infof("new leader %s is elected for controller %s", identity, c.name)
			},
		},
	}

	le, err := leaderelection.NewLeaderElector(lec)
	if err != nil {
		panic(fmt.Errorf("failed to create a LeaderElector for controller %s: %v", c.name, err))
	}
	c.le = le
	return c
}

// WithCacheSynced sets all the resource cacheSynced
func (c *Controller) WithCacheSynced(informersSynced ...cache.InformerSynced) *Controller {
	c.informersSynced = append(c.informersSynced, informersSynced...)
	return c
}

// Enqueue takes an object and converts it into a key (could be a string, or a struct) which is then put onto the
// work queue.
func (c *Controller) Enqueue(obj interface{}) {
	key, err := c.enqueueFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.queue.Add(key)
}

// Run will start multiple workers to process work items from work queue. It will block until ctx is closed.
func (c *Controller) Run(ctx context.Context) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	if c.handlerContextFunc == nil {
		panic(fmt.Errorf("please set handlerContextFunc for controller %s", c.name))
	}

	c.once.Do(func() {
		if c.le != nil {
			wait.UntilWithContext(ctx, c.le.Run, time.Duration(0))
			return
		}
		c.run(ctx)
	})
}

func (c *Controller) run(ctx context.Context) {
	klog.Infof("starting controller %s", c.name)
	defer klog.Infof("shutting down controller %s", c.name)
	c.runFlag = true

	// Wait for all the caches to be synced before starting workers
	if !cache.WaitForNamedCacheSync(c.name, ctx.Done(), c.informersSynced...) {
		return
	}

	klog.V(4).Infof("starting %d workers for controller %s", *c.workers, c.name)
	// Launch workers to process work items from queue
	for i := 0; i < *c.workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	<-ctx.Done()
	klog.V(4).Infof("stopped %d workers for controller %s", *c.workers, c.name)
}

// runWorker starts an infinite loop on processing the work item until the work queue is shut down.
func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem reads a single work item from the work queue
func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	item, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(item)

	requeueAfter, err := c.handlerContextFunc(ctx, item)
	if err == nil {
		c.queue.Forget(item)
		if requeueAfter != nil {
			// Sometimes we may want to re-visit this object after a while.
			// Put the item back on the work queue with delay.
			c.queue.AddAfter(item, *requeueAfter)
		}
		return true
	}

	if apierrors.IsNotFound(err) {
		c.queue.Forget(item)
		return true
	}

	utilruntime.HandleError(err)
	// put the item back on the work queue to handle any transient errors
	if requeueAfter != nil {
		c.queue.AddAfter(item, *requeueAfter)
	} else {
		c.queue.AddRateLimited(item)
	}
	return true
}

// DefaultEnqueueFunc uses a default namespacedKey as its KeyFunc.
// The key uses the format <namespace>/<name> unless <namespace> is empty, then
// it's just <name>.
func DefaultEnqueueFunc(obj interface{}) (interface{}, error) {
	return cache.MetaNamespaceKeyFunc(obj)
}
