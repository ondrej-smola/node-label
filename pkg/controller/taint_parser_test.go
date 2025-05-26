package controller

import (
	"reflect"
	"testing"
	
	corev1 "k8s.io/api/core/v1"
)

func TestParseTaints(t *testing.T) {
	tests := []struct {
		name        string
		taintValue  string
		expected    []corev1.Taint
		expectError bool
	}{
		{
			name:       "valid single taint with value",
			taintValue: "key1=value1:NoSchedule",
			expected: []corev1.Taint{
				{
					Key:    "key1",
					Value:  "value1",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
		},
		{
			name:       "valid single taint without value",
			taintValue: "key1:NoSchedule",
			expected: []corev1.Taint{
				{
					Key:    "key1",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
		},
		{
			name:       "multiple taints with different effects",
			taintValue: "key1=value1:NoSchedule,key2:PreferNoSchedule,key3=value3:NoExecute",
			expected: []corev1.Taint{
				{
					Key:    "key1",
					Value:  "value1",
					Effect: corev1.TaintEffectNoSchedule,
				},
				{
					Key:    "key2",
					Effect: corev1.TaintEffectPreferNoSchedule,
				},
				{
					Key:    "key3",
					Value:  "value3",
					Effect: corev1.TaintEffectNoExecute,
				},
			},
		},
		{
			name:       "taints with spaces",
			taintValue: " key1 = value1 : NoSchedule , key2 : PreferNoSchedule ",
			expected: []corev1.Taint{
				{
					Key:    "key1",
					Value:  "value1",
					Effect: corev1.TaintEffectNoSchedule,
				},
				{
					Key:    "key2",
					Effect: corev1.TaintEffectPreferNoSchedule,
				},
			},
		},
		{
			name:       "empty taint value",
			taintValue: "",
			expected:   []corev1.Taint{},
		},
		{
			name:       "trailing comma",
			taintValue: "key1:NoSchedule,",
			expected: []corev1.Taint{
				{
					Key:    "key1",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
		},
		{
			name:       "taint with equals sign in value",
			taintValue: "config=key1=value1:NoSchedule",
			expected: []corev1.Taint{
				{
					Key:    "config",
					Value:  "key1=value1",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
		},
		{
			name:       "all taint effects",
			taintValue: "key1:NoSchedule,key2:PreferNoSchedule,key3:NoExecute",
			expected: []corev1.Taint{
				{
					Key:    "key1",
					Effect: corev1.TaintEffectNoSchedule,
				},
				{
					Key:    "key2",
					Effect: corev1.TaintEffectPreferNoSchedule,
				},
				{
					Key:    "key3",
					Effect: corev1.TaintEffectNoExecute,
				},
			},
		},
		{
			name:        "invalid format - no effect",
			taintValue:  "key1=value1",
			expectError: true,
		},
		{
			name:        "invalid format - missing colon",
			taintValue:  "key1-value1-NoSchedule",
			expectError: true,
		},
		{
			name:        "invalid effect",
			taintValue:  "key1:InvalidEffect",
			expectError: true,
		},
		{
			name:        "empty key",
			taintValue:  "=value1:NoSchedule",
			expectError: true,
		},
		{
			name:        "invalid effect case",
			taintValue:  "key1:noschedule",
			expectError: true,
		},
		{
			name:       "taint with special characters in key",
			taintValue: "example.com/special-key=value:NoSchedule",
			expected: []corev1.Taint{
				{
					Key:    "example.com/special-key",
					Value:  "value",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
		},
		{
			name:       "empty value with equals",
			taintValue: "key1=:NoSchedule",
			expected: []corev1.Taint{
				{
					Key:    "key1",
					Value:  "",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTaints(tt.taintValue)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}