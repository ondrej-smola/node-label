package controller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	labelKey      = "altinity.cloud/auto-zone"
	taintLabelKey = "altinity.cloud/auto-taint"
	zoneLabel     = "topology.kubernetes.io/zone"
)

type Controller struct {
	clientset  kubernetes.Interface
	nodeLister listerscorev1.NodeLister
	nodeSynced cache.InformerSynced
	workqueue  workqueue.RateLimitingInterface
}

func NewController(clientset kubernetes.Interface) *Controller {
	nodeInformer := informerscorev1.NewNodeInformer(
		clientset,
		time.Second*30,
		cache.Indexers{},
	)

	controller := &Controller{
		clientset:  clientset,
		nodeLister: listerscorev1.NewNodeLister(nodeInformer.GetIndexer()),
		nodeSynced: nodeInformer.HasSynced,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Nodes"),
	}

	_, err := nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNode,
		UpdateFunc: func(old, new interface{}) {
			oldNode := old.(*corev1.Node)
			newNode := new.(*corev1.Node)
			if oldNode.ResourceVersion != newNode.ResourceVersion {
				controller.enqueueNode(new)
			}
		},
	})
	if err != nil {
		runtime.HandleError(fmt.Errorf("failed to add event handler: %v", err))
		return nil
	}

	go nodeInformer.Run(context.Background().Done())

	return controller
}

func (c *Controller) enqueueNode(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) Run(ctx context.Context, workers int) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	slog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.nodeSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	slog.Info("Starting workers")
	for range workers {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	slog.Info("Started workers")
	<-ctx.Done()
	slog.Info("Shutting down workers")

	return nil
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj any) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(ctx, key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(obj)
		slog.Info("Successfully synced", "key", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandler(ctx context.Context, key string) error {
	node, err := c.nodeLister.Get(key)
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("node '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	// Always add altinity.cloud/use=anywhere label
	labels := map[string]string{
		"altinity.cloud/use": "anywhere",
	}
	if err := c.applyLabels(ctx, node, labels); err != nil {
		return fmt.Errorf("failed to apply 'altinity.cloud/use=anywhere' label to node %s: %v", node.Name, err)
	}

	// Handle zone label
	zoneValue, hasZone := node.Labels[labelKey]
	if hasZone && zoneValue != "" {
		labels := map[string]string{
			zoneLabel: zoneValue,
		}
		if err := c.applyLabels(ctx, node, labels); err != nil {
			return fmt.Errorf("failed to apply zone label to node %s: %v", node.Name, err)
		}
	}

	// Handle taints
	taintValue, hasTaints := node.Labels[taintLabelKey]
	if hasTaints && taintValue != "" {
		var taints []corev1.Taint
		switch taintValue {
		case "clickhouse":
			taints = []corev1.Taint{{
				Key:    "dedicated",
				Value:  "clickhouse",
				Effect: corev1.TaintEffectNoSchedule,
			}}
		case "zookeeper":
			taints = []corev1.Taint{{
				Key:    "dedicated",
				Value:  "zookeeper",
				Effect: corev1.TaintEffectNoSchedule,
			}}
		default:
			slog.Error("invalid value for taint label", "label", taintLabelKey, "value", taintValue, "node", node.Name)
			// do nothing
		}

		if len(taints) > 0 {
			if err := c.applyTaints(ctx, node, taints); err != nil {
				return fmt.Errorf("failed to apply taints to node %s: %v", node.Name, err)
			}
		}
	}

	// If neither zone nor taints are present, skip
	if !hasZone && !hasTaints {
		slog.Debug("Node does not have zone or taint annotations, skipping", "node", node.Name)
	}

	return nil
}
