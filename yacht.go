package yacht

import (
	"context"
	"fmt"
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
)

type Controller struct {
	// name is the name of this controller
	name string
	// workers indicates the number of workers
	workers *int
	// enqueueFunc defines the function to enqueue the work item
	enqueueFunc EnqueueFunc
	// workqueue is a rate limited work queue.
	workqueue workqueue.RateLimitingInterface
	// informersSynced records a group of cacheSyncs
	// The workers will not start working before all the caches are synced successfully
	informersSynced []cache.InformerSynced
	// handlerFunc defines the handler to process the work item
	handlerFunc HandlerFunc
	// le specifies the LeaderElector to use
	le *leaderelection.LeaderElector

	// runFlag indicates whether the workers start working
	runFlag bool
}

var _ Interface = &Controller{}

// NewController creates a new Controller
func NewController(name string) *Controller {
	return &Controller{
		name:            name,
		workers:         utilpointer.Int(2),
		enqueueFunc:     DefaultEnqueueFunc,
		workqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name),
		informersSynced: []cache.InformerSynced{},
	}
}

// WithWorkers sets the number of workers to process work items off work queue
func (c *Controller) WithWorkers(workers int) *Controller {
	if c.runFlag && c.workers != nil {
		panic(fmt.Errorf("can not mutate workers when controller %s is running", c.name))
	}

	c.workers = utilpointer.Int(workers)
	return c
}

// WithEnqueueFunc sets customize enqueueFunc
func (c *Controller) WithEnqueueFunc(enqueueFunc EnqueueFunc) *Controller {
	if c.runFlag && c.enqueueFunc != nil {
		panic(fmt.Errorf("can not mutate enqueueFunc when controller %s is running", c.name))
	}

	if enqueueFunc != nil {
		c.enqueueFunc = enqueueFunc
	}
	return c
}

// WithHandlerFunc sets a handler function to process the work item off the work queue
func (c *Controller) WithHandlerFunc(handlerFunc HandlerFunc) *Controller {
	if c.runFlag && c.handlerFunc != nil {
		panic(fmt.Errorf("can not mutate handlerFunc when controller %s is running", c.name))
	}

	if handlerFunc != nil {
		c.handlerFunc = handlerFunc
	}
	return c
}

// WithLeaderElection uses leader election to get the lock
func (c *Controller) WithLeaderElection(leaseLock rl.Interface, leaseDuration, renewDeadline, retryPeriod time.Duration) *Controller {
	if c.runFlag && c.le != nil {
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
	c.workqueue.Add(key)
}

// Run will start multiple workers to process work items from work queue. It will block until ctx is closed.
func (c *Controller) Run(ctx context.Context) {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	if c.handlerFunc == nil {
		panic(fmt.Errorf("empty handlerFunc for controller %s", c.name))
	}

	if c.le != nil {
		wait.UntilWithContext(ctx, c.le.Run, time.Duration(0))
		return
	}

	c.run(ctx)
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
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
	klog.V(4).Infof("stopped %d workers for controller %s", *c.workers, c.name)
}

// runWorker starts an infinite loop on processing the work item until the work queue is shut down.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem reads a single work item from the work queue
func (c *Controller) processNextWorkItem() bool {
	item, quit := c.workqueue.Get()
	if quit {
		return false
	}
	defer c.workqueue.Done(item)

	requeueAfter, err := c.handlerFunc(item)
	if err == nil {
		c.workqueue.Forget(item)
		return true
	}

	if apierrors.IsNotFound(err) {
		c.workqueue.Forget(item)
		return true
	}

	utilruntime.HandleError(err)
	// put the item back on the work queue to handle any transient errors
	if requeueAfter != nil {
		c.workqueue.AddAfter(item, *requeueAfter)
	} else {
		c.workqueue.AddRateLimited(item)
	}
	return true
}

// DefaultEnqueueFunc uses a default namespacedKey as its KeyFunc.
// The key uses the format <namespace>/<name> unless <namespace> is empty, then
// it's just <name>.
func DefaultEnqueueFunc(obj interface{}) (interface{}, error) {
	return cache.MetaNamespaceKeyFunc(obj)
}
