package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	zoneOldLabel  = "failure-domain.beta.kubernetes.io/zone"
)

type Controller struct {
	clientset  kubernetes.Interface
	nodeLister listerscorev1.NodeLister
	nodeSynced cache.InformerSynced
	workqueue  workqueue.TypedRateLimitingInterface[string]
}

func NewController(clientset kubernetes.Interface) *Controller {
	nodeInformer := informerscorev1.NewNodeInformer(
		clientset,
		30*time.Second,
		cache.Indexers{},
	)

	controller := &Controller{
		clientset:  clientset,
		nodeLister: listerscorev1.NewNodeLister(nodeInformer.GetIndexer()),
		nodeSynced: nodeInformer.HasSynced,
		workqueue:  workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]()),
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
	if key, err := cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	} else {
		c.workqueue.Add(key)
	}
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
		key, ok := obj.(string)
		if !ok {
			c.workqueue.Forget("")
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		defer c.workqueue.Done(key)

		if err := c.syncHandler(ctx, key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(key)
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

	zone, ok := node.Labels[labelKey]
	if !ok || zone == "" {
		return nil
	}

	if err := c.applyZoneLabel(ctx, node, zone); err != nil {
		return fmt.Errorf("failed to apply labels to node %s: %v", node.Name, err)
	}

	return nil
}

func (c *Controller) applyZoneLabel(ctx context.Context, node *corev1.Node, zone string) error {
	// Only patch if at least one of the zone labels is missing or has a different value
	currentZone := node.Labels[zoneLabel]
	currentOldZone := node.Labels[zoneOldLabel]
	if currentZone == zone && currentOldZone == zone {
		slog.Debug("Zone labels already set correctly, skipping patch", "node", node.Name, "zone", zone)
		return nil
	}

	patch := map[string]any{
		"metadata": map[string]any{
			"labels": map[string]string{
				zoneLabel:    zone,
				zoneOldLabel: zone,
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %v", err)
	}

	_, err = c.clientset.CoreV1().Nodes().Patch(
		ctx,
		node.Name,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)

	if err != nil {
		return fmt.Errorf("failed to patch node: %v", err)
	}

	slog.Info("Successfully applied zone label to node", "node", node.Name, "zone", zone)
	return nil
}
