package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

// DepthLogging uses depth to determine which call frame to log.
func DepthLogging(err error, logType, msg string, obj interface{}, keysAndValues ...interface{}) {
	if u, ok := obj.(schema.ObjectKind); ok && u != nil {
		keysAndValues = append(keysAndValues,
			"Kind", u.GroupVersionKind().Kind,
			"APIVersion", u.GroupVersionKind().GroupVersion().String(),
		)
	}

	if u, ok := obj.(metav1.Object); ok && u != nil {
		keysAndValues = append(keysAndValues,
			"Namespace", u.GetNamespace(),
			"Name", u.GetName(),
			"UID", u.GetUID(),
		)
	} else {
		keysAndValues = append(keysAndValues,
			"object", obj,
		)
	}

	switch logType {
	case "info":
		klog.V(4).InfoS(msg, keysAndValues...)
	case "warning":
		// TODO: use WarningS
		klog.V(3).InfoS(msg, keysAndValues...)
	case "error":
		klog.V(2).ErrorS(err, msg, keysAndValues...)
	default:
		// no-op
	}
}
