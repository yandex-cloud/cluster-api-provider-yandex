package v1alpha1

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func Test_yandexClusterAdmitter_Handle(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Error(err)
		return
	}

	generateRuntimeObj := func(cluster *YandexCluster) runtime.RawExtension {
		if cluster == nil {
			return runtime.RawExtension{}
		}

		objRaw, err := json.Marshal(cluster)
		if err != nil {
			t.Error(err)
			return runtime.RawExtension{}
		}

		return runtime.RawExtension{
			Raw: objRaw,
		}
	}

	tests := []struct {
		name       string
		operation  admissionv1.Operation
		newCluster *YandexCluster
		oldCluster *YandexCluster
		wantPatch  *jsonpatch.JsonPatchOperation
		want       admission.Response
	}{
		{
			name:       "delete operation",
			operation:  admissionv1.Delete,
			newCluster: nil,
			oldCluster: nil,
			want:       admission.Allowed("allowed"),
		},
		{
			name:       "bad create request",
			operation:  admissionv1.Create,
			newCluster: nil,
			oldCluster: nil,
			want:       admission.Errored(400, errors.New("failed to decode request: there is no content to decode")),
		},
		{
			name:       "bad update request",
			operation:  admissionv1.Update,
			newCluster: &YandexCluster{},
			oldCluster: nil,
			want:       admission.Errored(400, errors.New("failed to decode request: there is no content to decode")),
		},
		{
			name:       "new cluster with identityRef",
			operation:  admissionv1.Create,
			newCluster: &YandexCluster{Spec: YandexClusterSpec{IdentityRef: &IdentityReference{Name: "name", Namespace: "namespace"}}},
			wantPatch: &jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      "/metadata/labels",
				Value:     map[string]string{"yandexidentity/namespace": "name"}},
		},
		{
			name:      "cluster with identityRef and existing label",
			operation: admissionv1.Create,
			newCluster: &YandexCluster{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"yandexidentity/namespace": "name"}},
				Spec:       YandexClusterSpec{IdentityRef: &IdentityReference{Name: "name", Namespace: "namespace"}},
			},
			want: admission.Allowed("identityRef label already exists"),
		},
		{
			name:       "cluster without identityRef and nil old cluster",
			operation:  admissionv1.Create,
			newCluster: &YandexCluster{},
			oldCluster: nil,
			want:       admission.Allowed("allowed"),
		},
		{
			name:       "cluster without identityRef and same old cluster",
			operation:  admissionv1.Update,
			newCluster: &YandexCluster{},
			oldCluster: &YandexCluster{},
			want:       admission.Allowed("allowed"),
		},
		{
			name:      "cluster without identityRef and old cluster with identityRef",
			operation: admissionv1.Update,
			newCluster: &YandexCluster{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"yandexidentity/namespace": "name"}},
			},
			oldCluster: &YandexCluster{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"yandexidentity/namespace": "name"}},
				Spec:       YandexClusterSpec{IdentityRef: &IdentityReference{Name: "name", Namespace: "namespace"}},
			},
			wantPatch: &jsonpatch.JsonPatchOperation{
				Operation: "remove", Path: "/metadata/labels"},
		},
	}

	m := &yandexClusterAdmitter{
		decoder: admission.NewDecoder(scheme),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: tt.operation,
					Object:    generateRuntimeObj(tt.newCluster),
					OldObject: generateRuntimeObj(tt.oldCluster),
				},
			}

			got := m.Handle(context.TODO(), req)

			if tt.wantPatch != nil {
				g := got.Patches[0].Json()
				w := tt.wantPatch.Json()
				if g == w {
					// ok
					return
				}

				t.Errorf("incorrect patch, got %s, want %s", got.Patches[0].Json(), tt.wantPatch.Json())
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("yandexClusterAdmitter.Handle() = %v, want %v", got, tt.want)
			}
		})
	}
}
