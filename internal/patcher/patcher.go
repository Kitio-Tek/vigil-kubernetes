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

// Package patcher provides utilities for server-side apply and merge patching
// of Kubernetes objects managed by the athos-kubernetes operator.
package patcher

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Patcher wraps a controller-runtime client and provides higher-level patching
// methods used throughout the reconciliation loop.
type Patcher struct {
	client client.Client
}

// New creates a new Patcher backed by the provided client.
func New(c client.Client) *Patcher {
	return &Patcher{client: c}
}

// Apply performs a server-side apply for the given object using the specified
// field manager name. The object must have its TypeMeta set so that the API
// server can identify the resource type.
func (p *Patcher) Apply(ctx context.Context, obj client.Object, fieldOwner string) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("marshal object for server-side apply: %w", err)
	}
	return p.client.Patch(ctx, obj, client.RawPatch(types.ApplyPatchType, data),
		client.FieldOwner(fieldOwner),
		client.ForceOwnership,
	)
}

// Patch creates a merge patch between base and modified and applies it to the
// live object. The base argument should be the unmodified object as returned by
// a GET, and modified should be the desired state. The fieldOwner parameter is
// used as the field manager name for the patch.
func (p *Patcher) Patch(ctx context.Context, base, modified client.Object, fieldOwner string) error {
	baseBytes, err := json.Marshal(base)
	if err != nil {
		return fmt.Errorf("marshal base object: %w", err)
	}
	modifiedBytes, err := json.Marshal(modified)
	if err != nil {
		return fmt.Errorf("marshal modified object: %w", err)
	}
	// Build a simple merge patch by computing the JSON diff.
	patch, err := createMergePatch(baseBytes, modifiedBytes)
	if err != nil {
		return fmt.Errorf("compute merge patch: %w", err)
	}
	if string(patch) == "{}" {
		return nil
	}
	return p.client.Patch(ctx, modified, client.RawPatch(types.MergePatchType, patch),
		client.FieldOwner(fieldOwner),
	)
}

// EnsureOwnerRef sets a controller owner reference on obj pointing to owner.
// It uses the scheme to resolve the GVK for owner. If an equivalent reference
// already exists, this is a no-op.
func EnsureOwnerRef(obj, owner metav1.Object, scheme *runtime.Scheme) error {
	ownerObj, ok := owner.(runtime.Object)
	if !ok {
		return fmt.Errorf("owner does not implement runtime.Object")
	}
	gvks, _, err := scheme.ObjectKinds(ownerObj)
	if err != nil {
		return fmt.Errorf("resolve GVK for owner: %w", err)
	}
	if len(gvks) == 0 {
		return fmt.Errorf("no GVK found for owner type")
	}
	gvk := gvks[0]
	return setOwnerReference(obj, owner, gvk)
}

// setOwnerReference adds an owner reference to obj using the provided GVK.
func setOwnerReference(obj, owner metav1.Object, gvk schema.GroupVersionKind) error {
	t := true
	ref := metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               owner.GetName(),
		UID:                owner.GetUID(),
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}
	existing := obj.GetOwnerReferences()
	for _, r := range existing {
		if r.UID == ref.UID {
			return nil
		}
	}
	obj.SetOwnerReferences(append(existing, ref))
	return nil
}

// createMergePatch computes the JSON merge patch required to transform original
// into modified. It returns an empty object ("{}") when there are no differences.
func createMergePatch(original, modified []byte) ([]byte, error) {
	var origMap, modMap map[string]interface{}
	if err := json.Unmarshal(original, &origMap); err != nil {
		return nil, fmt.Errorf("unmarshal original: %w", err)
	}
	if err := json.Unmarshal(modified, &modMap); err != nil {
		return nil, fmt.Errorf("unmarshal modified: %w", err)
	}
	patch := diffMaps(origMap, modMap)
	return json.Marshal(patch)
}

// diffMaps returns a map containing only the keys from modified that differ
// from original, suitable for use as a JSON merge patch.
func diffMaps(original, modified map[string]interface{}) map[string]interface{} {
	patch := make(map[string]interface{})
	for key, modVal := range modified {
		origVal, exists := original[key]
		if !exists {
			patch[key] = modVal
			continue
		}
		modMap, modIsMap := modVal.(map[string]interface{})
		origMap, origIsMap := origVal.(map[string]interface{})
		if modIsMap && origIsMap {
			sub := diffMaps(origMap, modMap)
			if len(sub) > 0 {
				patch[key] = sub
			}
			continue
		}
		if fmt.Sprintf("%v", origVal) != fmt.Sprintf("%v", modVal) {
			patch[key] = modVal
		}
	}
	// Mark keys removed from modified as null in the patch.
	for key := range original {
		if _, exists := modified[key]; !exists {
			patch[key] = nil
		}
	}
	return patch
}

// MustControllerReference is a convenience wrapper around
// controllerutil.SetControllerReference that panics on error. It is intended
// for use in test helpers where the arguments are always valid.
func MustControllerReference(owner, obj client.Object, scheme *runtime.Scheme) {
	if err := controllerutil.SetControllerReference(owner, obj, scheme); err != nil {
		panic(fmt.Sprintf("MustControllerReference: %v", err))
	}
}
