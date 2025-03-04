package v1alpha1

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func Test_yandexClusterAdmitter_Handle(t *testing.T) {
	errConversion := errors.New("failed to convert runtime Object to YandexCluster")

	tests := []struct {
		name       string
		newCluster runtime.Object
		oldCluster runtime.Object
		want       admission.Response
	}{
		{
			name:       "nil cluster",
			newCluster: nil,
			want:       admission.Errored(http.StatusBadRequest, errConversion),
		},
		{
			name:       "new cluster with identityRef",
			newCluster: &YandexCluster{Spec: YandexClusterSpec{IdentityRef: &IdentityReference{Name: "name", Namespace: "namespace"}}},
			want: admission.Patched("added label for identityRef",
				jsonpatch.JsonPatchOperation{
					Operation: "add", Path: "/metadata/labels/yandexidentity/namespace", Value: "name"}),
		},
		{
			name: "cluster with identityRef and existing label",
			newCluster: &YandexCluster{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"yandexidentity/namespace": "name"}},
				Spec:       YandexClusterSpec{IdentityRef: &IdentityReference{Name: "name", Namespace: "namespace"}},
			},
			want: admission.Allowed("identityRef label already exists"),
		},
		{
			name:       "cluster without identityRef and nil old cluster",
			newCluster: &YandexCluster{},
			oldCluster: nil,
			want:       admission.Allowed("allowed"),
		},
		{
			name:       "cluster without identityRef and same old cluster",
			newCluster: &YandexCluster{},
			oldCluster: &YandexCluster{},
			want:       admission.Allowed("allowed"),
		},
		{
			name:       "cluster without identityRef and old cluster with identityRef",
			newCluster: &YandexCluster{},
			oldCluster: &YandexCluster{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"yandexidentity/namespace": "name"}},
				Spec:       YandexClusterSpec{IdentityRef: &IdentityReference{Name: "name", Namespace: "namespace"}},
			},
			want: admission.Patched("removed label for identityRef",
				jsonpatch.JsonPatchOperation{
					Operation: "remove", Path: "/metadata/labels/yandexidentity/namespace"}),
		},
		{
			name:       "incorrect old cluster",
			newCluster: &YandexCluster{},
			oldCluster: &YandexIdentity{},
			want:       admission.Errored(http.StatusBadRequest, errConversion),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &yandexClusterAdmitter{}
			req := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object:    runtime.RawExtension{Object: tt.newCluster},
					OldObject: runtime.RawExtension{Object: tt.oldCluster},
				},
			}
			if got := m.Handle(context.TODO(), req); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("yandexClusterAdmitter.Handle() = %v, want %v", got, tt.want)
			}
		})
	}
}
