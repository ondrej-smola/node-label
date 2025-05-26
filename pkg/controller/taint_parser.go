package controller

import (
	"fmt"
	"strings"
	
	corev1 "k8s.io/api/core/v1"
)

// parseTaints parses a string of comma-separated taints in the format:
// "key1=value1:effect1,key2=value2:effect2" or "key1:effect1" for taints without values
func parseTaints(taintValue string) ([]corev1.Taint, error) {
	var taints []corev1.Taint
	
	if taintValue == "" {
		return taints, nil
	}
	
	pairs := strings.Split(taintValue, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		
		// Split by colon to separate key=value from effect
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid taint format: %s (expected key=value:effect or key:effect)", pair)
		}
		
		keyValuePart := strings.TrimSpace(parts[0])
		effectPart := strings.TrimSpace(parts[1])
		
		// Validate effect
		var effect corev1.TaintEffect
		switch effectPart {
		case "NoSchedule":
			effect = corev1.TaintEffectNoSchedule
		case "PreferNoSchedule":
			effect = corev1.TaintEffectPreferNoSchedule
		case "NoExecute":
			effect = corev1.TaintEffectNoExecute
		default:
			return nil, fmt.Errorf("invalid taint effect: %s (must be NoSchedule, PreferNoSchedule, or NoExecute)", effectPart)
		}
		
		// Parse key and optional value
		taint := corev1.Taint{
			Effect: effect,
		}
		
		if strings.Contains(keyValuePart, "=") {
			// Has value
			kvParts := strings.SplitN(keyValuePart, "=", 2)
			taint.Key = strings.TrimSpace(kvParts[0])
			taint.Value = strings.TrimSpace(kvParts[1])
		} else {
			// No value
			taint.Key = keyValuePart
		}
		
		if taint.Key == "" {
			return nil, fmt.Errorf("empty key in taint: %s", pair)
		}
		
		taints = append(taints, taint)
	}
	
	return taints, nil
}