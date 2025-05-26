package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (c *Controller) applyLabels(ctx context.Context, node *corev1.Node, labels map[string]string) error {
	newLabels := make(map[string]string)
	changed := false
	
	for k, v := range labels {
		if currentValue, exists := node.Labels[k]; !exists || currentValue != v {
			newLabels[k] = v
			changed = true
			slog.Debug("Label will be set on node", "label", k, "value", v, "node", node.Name)
		}
	}
	
	if !changed {
		slog.Debug("No label changes needed for node", "node", node.Name)
		return nil
	}
	
	patch := map[string]any{
		"metadata": map[string]any{
			"labels": newLabels,
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
	
	slog.Info("Successfully applied labels to node", "count", len(newLabels), "node", node.Name)
	return nil
}

func (c *Controller) applyTaints(ctx context.Context, node *corev1.Node, taints []corev1.Taint) error {
	// Build a map of desired taints for easy lookup
	desiredTaints := make(map[string]corev1.Taint)
	for _, taint := range taints {
		key := taintKey(taint)
		desiredTaints[key] = taint
	}
	
	// Check existing taints
	existingTaints := make(map[string]corev1.Taint)
	for _, taint := range node.Spec.Taints {
		key := taintKey(taint)
		existingTaints[key] = taint
	}
	
	// Determine which taints need to be added
	var taintsToAdd []corev1.Taint
	for key, desiredTaint := range desiredTaints {
		if _, exists := existingTaints[key]; !exists {
			taintsToAdd = append(taintsToAdd, desiredTaint)
			slog.Debug("Taint will be added to node", "taint", desiredTaint, "node", node.Name)
		}
	}
	
	if len(taintsToAdd) == 0 {
		slog.Debug("No taint changes needed for node", "node", node.Name)
		return nil
	}
	
	// Create patch for adding taints
	// We need to use a JSON patch to add to the taints array
	var patches []map[string]any
	
	// If no taints exist, we need to create the array
	if len(node.Spec.Taints) == 0 {
		patches = append(patches, map[string]any{
			"op":    "add",
			"path":  "/spec/taints",
			"value": taintsToAdd,
		})
	} else {
		// Add each taint individually
		for _, taint := range taintsToAdd {
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  "/spec/taints/-",
				"value": taint,
			})
		}
	}
	
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return fmt.Errorf("failed to marshal taint patch: %v", err)
	}
	
	_, err = c.clientset.CoreV1().Nodes().Patch(
		ctx,
		node.Name,
		types.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	
	if err != nil {
		return fmt.Errorf("failed to patch node with taints: %v", err)
	}
	
	slog.Info("Successfully applied taints to node", "count", len(taintsToAdd), "node", node.Name)
	return nil
}

// taintKey generates a unique key for a taint based on its key and effect
func taintKey(taint corev1.Taint) string {
	return fmt.Sprintf("%s:%s", taint.Key, taint.Effect)
}