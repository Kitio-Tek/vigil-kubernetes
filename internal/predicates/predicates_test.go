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

package predicates_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/predicates"
)

func podWithMeta(generation int64, labels, annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Generation:  generation,
			Labels:      labels,
			Annotations: annotations,
		},
	}
}

func TestResourceGenerationChanged_Update(t *testing.T) {
	p := predicates.ResourceGenerationChanged()

	t.Run("generation unchanged", func(t *testing.T) {
		e := event.UpdateEvent{
			ObjectOld: podWithMeta(1, nil, nil),
			ObjectNew: podWithMeta(1, nil, nil),
		}
		if p.Update(e) {
			t.Error("expected predicate to return false for unchanged generation")
		}
	})

	t.Run("generation changed", func(t *testing.T) {
		e := event.UpdateEvent{
			ObjectOld: podWithMeta(1, nil, nil),
			ObjectNew: podWithMeta(2, nil, nil),
		}
		if !p.Update(e) {
			t.Error("expected predicate to return true for changed generation")
		}
	})
}

func TestResourceGenerationChanged_CreateDeleteGeneric(t *testing.T) {
	p := predicates.ResourceGenerationChanged()
	pod := podWithMeta(1, nil, nil)

	if !p.Create(event.CreateEvent{Object: pod}) {
		t.Error("expected create to pass")
	}
	if !p.Delete(event.DeleteEvent{Object: pod}) {
		t.Error("expected delete to pass")
	}
	if !p.Generic(event.GenericEvent{Object: pod}) {
		t.Error("expected generic to pass")
	}
}

func TestAnnotationChangedOrGeneration_AnnotationChange(t *testing.T) {
	p := predicates.AnnotationChangedOrGeneration()

	old := podWithMeta(1, nil, map[string]string{"key": "old"})
	new := podWithMeta(1, nil, map[string]string{"key": "new"})

	e := event.UpdateEvent{ObjectOld: old, ObjectNew: new}
	if !p.Update(e) {
		t.Error("expected predicate to pass on annotation change")
	}
}

func TestAnnotationChangedOrGeneration_NoChange(t *testing.T) {
	p := predicates.AnnotationChangedOrGeneration()

	old := podWithMeta(1, nil, map[string]string{"key": "same"})
	new := podWithMeta(1, nil, map[string]string{"key": "same"})

	e := event.UpdateEvent{ObjectOld: old, ObjectNew: new}
	if p.Update(e) {
		t.Error("expected predicate to block when nothing changed")
	}
}

func TestNotPaused_PausedAnnotation(t *testing.T) {
	p := predicates.NotPaused()

	paused := podWithMeta(1, nil, map[string]string{"pg.vigil.io/paused": "true"})
	active := podWithMeta(1, nil, nil)

	if p.Create(event.CreateEvent{Object: paused}) {
		t.Error("expected create to be blocked for paused object")
	}
	if !p.Create(event.CreateEvent{Object: active}) {
		t.Error("expected create to pass for active object")
	}
}

func TestNotPaused_DeleteAlwaysPasses(t *testing.T) {
	p := predicates.NotPaused()
	paused := podWithMeta(1, nil, map[string]string{"pg.vigil.io/paused": "true"})

	if !p.Delete(event.DeleteEvent{Object: paused}) {
		t.Error("delete events should always pass NotPaused")
	}
}

func TestOwnedByCluster(t *testing.T) {
	p := predicates.OwnedByCluster("mycluster")

	owned := podWithMeta(1, map[string]string{"pg.vigil.io/cluster": "mycluster"}, nil)
	other := podWithMeta(1, map[string]string{"pg.vigil.io/cluster": "othercluster"}, nil)

	if !p.Create(event.CreateEvent{Object: owned}) {
		t.Error("expected owned object to pass")
	}
	if p.Create(event.CreateEvent{Object: other}) {
		t.Error("expected non-owned object to be blocked")
	}
}

func TestLabelChangedOrGeneration_LabelChange(t *testing.T) {
	p := predicates.LabelChangedOrGeneration()

	old := podWithMeta(1, map[string]string{"key": "old"}, nil)
	new := podWithMeta(1, map[string]string{"key": "new"}, nil)

	e := event.UpdateEvent{ObjectOld: old, ObjectNew: new}
	if !p.Update(e) {
		t.Error("expected predicate to pass on label change")
	}
}
