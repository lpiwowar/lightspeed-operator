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

package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	common_helper "github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	apiv1beta1 "github.com/openstack-k8s-operators/lightspeed-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func makeExistingPVC(size string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PostgresDataPVCName,
			Namespace: "test-ns",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
		},
	}
}

func makeInstanceWithDBSize(size string) *apiv1beta1.OpenStackLightspeed {
	return &apiv1beta1.OpenStackLightspeed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "test-ns",
		},
		Spec: apiv1beta1.OpenStackLightspeedSpec{
			Database: &apiv1beta1.DatabaseSpec{
				Size: resource.MustParse(size),
			},
		},
	}
}

// Expanding the PVC patches its storage request to the larger value and returns no error.
func TestReconcilePostgresPVC_Expand(t *testing.T) {
	h := newTestHelper(t, makeExistingPVC("1Gi"))
	instance := makeInstanceWithDBSize("2Gi")

	ctx := context.Background()
	if err := reconcilePostgresPVC(h, ctx, instance); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := &corev1.PersistentVolumeClaim{}
	if err := h.GetClient().Get(ctx, client.ObjectKey{Name: PostgresDataPVCName, Namespace: "test-ns"}, updated); err != nil {
		t.Fatalf("failed to get PVC after expand: %v", err)
	}
	got := updated.Spec.Resources.Requests[corev1.ResourceStorage]
	want := resource.MustParse("2Gi")
	if got.Cmp(want) != 0 {
		t.Errorf("expected PVC storage %s after expand, got %s", want.String(), got.String())
	}
}

// Shrinking the PVC is rejected with ErrPostgresPVCSizeShrink; the PVC is not modified.
func TestReconcilePostgresPVC_Shrink(t *testing.T) {
	h := newTestHelper(t, makeExistingPVC("2Gi"))
	instance := makeInstanceWithDBSize("1Gi")

	ctx := context.Background()
	err := reconcilePostgresPVC(h, ctx, instance)
	if err == nil {
		t.Fatal("expected error when shrinking PVC, got nil")
	}
	if !errors.Is(err, ErrPostgresPVCSizeShrink) {
		t.Errorf("expected ErrPostgresPVCSizeShrink, got %v", err)
	}
}

// When the requested size matches the existing PVC, reconciliation is a no-op.
func TestReconcilePostgresPVC_NoOp(t *testing.T) {
	h := newTestHelper(t, makeExistingPVC("1Gi"))
	instance := makeInstanceWithDBSize("1Gi")

	ctx := context.Background()
	if err := reconcilePostgresPVC(h, ctx, instance); err != nil {
		t.Fatalf("unexpected error for matching size: %v", err)
	}
}

func newTestHelperWithPatchErr(t *testing.T, objs ...client.Object) *common_helper.Helper {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 to scheme: %v", err)
	}
	if err := apiv1beta1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add apiv1beta1 to scheme: %v", err)
	}

	patchErr := fmt.Errorf("forced patch failure")
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithInterceptorFuncs(interceptor.Funcs{
			Patch: func(_ context.Context, _ client.WithWatch, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
				return patchErr
			},
		}).
		Build()

	instance := &apiv1beta1.OpenStackLightspeed{
		ObjectMeta: metav1.ObjectMeta{Name: "test-instance", Namespace: "test-ns"},
	}
	logger := zap.New(zap.WriteTo(bytes.NewBuffer(nil)))
	h, err := common_helper.NewHelper(instance, fakeClient, nil, scheme, logger)
	if err != nil {
		t.Fatalf("failed to create helper: %v", err)
	}
	return h
}

// If the API server rejects the patch (e.g. StorageClass lacks allowVolumeExpansion), ErrPatchPostgresPVC is returned.
func TestReconcilePostgresPVC_PatchError(t *testing.T) {
	h := newTestHelperWithPatchErr(t, makeExistingPVC("1Gi"))
	instance := makeInstanceWithDBSize("2Gi")

	ctx := context.Background()
	err := reconcilePostgresPVC(h, ctx, instance)
	if err == nil {
		t.Fatal("expected error when patch fails, got nil")
	}
	if !errors.Is(err, ErrPatchPostgresPVC) {
		t.Errorf("expected ErrPatchPostgresPVC, got %v", err)
	}
}
