package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/dixudx/yacht"
	"github.com/dixudx/yacht/utils"
)

func main() {
	klog.InitFlags(flag.CommandLine)
	defer klog.Flush()

	var kubeconfig string
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.Parse()

	ctx := context.TODO()

	// 0. load kubeconfig and create clientset/informers/listers
	config, err := utils.LoadsKubeConfig(kubeconfig) // TODO: we need to use an explicit configfile when running locally
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	kubeClient := kubernetes.NewForConfigOrDie(config)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Hour*12)
	nsLister := kubeInformerFactory.Core().V1().Namespaces().Lister()

	// 1. create a controller for namespaces
	namespaceController := yacht.NewController("namespaces").
		WithWorkers(2).
		WithHandlerFunc(func(key interface{}) (requeueAfter *time.Duration, err error) {
			// We can use "WithEnqueueFunc" to set our own enqueueFunc, otherwise default namespacedKey will be used
			// Convert the namespace/name string into a distinct namespace and name
			_, name, err := cache.SplitMetaNamespaceKey(key.(string))
			if err != nil {
				utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
				return nil, err
			}

			// TODO: we can add our real logics here
			ns, err := nsLister.Get(name)
			if err != nil {
				return nil, err
			}
			klog.Infof("[mock] successfully processing namespace %s", ns.Name)
			return nil, nil

		})

	// 2. add event handler for namespaces on the addition/update/deletion
	kubeInformerFactory.Core().V1().Namespaces().Informer().AddEventHandler(namespaceController.DefaultResourceEventHandlerFuncs())

	// 3. start the informer factory
	kubeInformerFactory.Start(ctx.Done())

	// 4. run the controller
	namespaceController.WithCacheSynced(kubeInformerFactory.Core().V1().Namespaces().Informer().HasSynced).
		Run(ctx)
}
