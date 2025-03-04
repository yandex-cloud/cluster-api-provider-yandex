package v1alpha1

import (
	"context"
	"reflect"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func Test_yandexIdentityAdmitter_Handle(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Error(err)
		return
	}

	tests := []struct {
		name    string
		objects []runtime.Object
		req     admission.Request
		want    admission.Response
	}{
		{
			name:    "identity is not linked to any cluster",
			objects: []runtime.Object{},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Object: &YandexIdentity{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "test",
							},
						},
					},
				},
			},
			want: admission.Allowed("identity is not linked to any cluster"),
		},
		{
			name: "identity is linked to cluster",
			objects: []runtime.Object{
				&YandexCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
						Labels: map[string]string{
							"yandexidentity/test": "test",
						},
					},
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Object: &YandexIdentity{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "test",
							},
						},
					},
				},
			},
			want: admission.Denied("identity is linked to clusters"),
		},
		{
			name: "identity is linked to existing  cluster",
			objects: []runtime.Object{
				&YandexCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
						Labels: map[string]string{
							"yandexidentity/another": "another",
						},
					},
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Object: &YandexIdentity{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "test",
							},
						},
					},
				},
			},
			want: admission.Allowed("identity is not linked to any cluster"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build()

			m := &yandexIdentityAdmitter{
				platformClient: fakeClient,
			}
			if got := m.Handle(context.TODO(), tt.req); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("yandexIdentityAdmitter.Handle() = %v, want %v", got, tt.want)
			}
		})
	}
}
