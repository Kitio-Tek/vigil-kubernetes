/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package predicates provides controller-runtime event predicates used by
// Athos controllers to filter reconcile triggers and reduce unnecessary work.
package predicates

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ResourceGenerationChanged returns a predicate that passes update events only
// when the resource generation has changed. It passes all create, delete, and
// generic events. This avoids re-queuing on status-only updates.
func ResourceGenerationChanged() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}
}

// AnnotationChangedOrGeneration returns a predicate that passes update events
// when the resource generation changed OR when annotations changed. This is
// useful for controllers that need to react to annotation-driven operations
// (e.g., manual failover triggers).
func AnnotationChangedOrGeneration() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration() {
				return true
			}
			return annotationsChanged(e.ObjectOld.GetAnnotations(), e.ObjectNew.GetAnnotations())
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}
}

// LabelChangedOrGeneration returns a predicate that passes update events when
// the resource generation changed OR when labels changed.
func LabelChangedOrGeneration() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration() {
				return true
			}
			return labelsChanged(e.ObjectOld.GetLabels(), e.ObjectNew.GetLabels())
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}
}

// NotPaused returns a predicate that blocks all events when the object carries
// the athos.io/paused=true annotation. Controllers that honour the annotation
// should use this predicate to skip reconciliation entirely.
func NotPaused() predicate.Predicate {
	const (
		pausedAnnotation = "pg.athos.io/paused"
		pausedValue      = "true"
	)
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetAnnotations()[pausedAnnotation] != pausedValue
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetAnnotations()[pausedAnnotation] != pausedValue
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return e.Object.GetAnnotations()[pausedAnnotation] != pausedValue
		},
	}
}

// OwnedByCluster returns a predicate that passes only when the object carries a
// pg.athos.io/cluster label matching the given cluster name. Use this when
// watching secondary resources (Services, Secrets) to avoid triggering
// reconciliation for resources belonging to a different cluster.
func OwnedByCluster(clusterName string) predicate.Predicate {
	const clusterLabel = "pg.athos.io/cluster"
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetLabels()[clusterLabel] == clusterName
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetLabels()[clusterLabel] == clusterName
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return e.Object.GetLabels()[clusterLabel] == clusterName
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return e.Object.GetLabels()[clusterLabel] == clusterName
		},
	}
}

func annotationsChanged(old, new map[string]string) bool {
	if len(old) != len(new) {
		return true
	}
	for k, v := range old {
		if new[k] != v {
			return true
		}
	}
	return false
}

func labelsChanged(old, new map[string]string) bool {
	if len(old) != len(new) {
		return true
	}
	for k, v := range old {
		if new[k] != v {
			return true
		}
	}
	return false
}
