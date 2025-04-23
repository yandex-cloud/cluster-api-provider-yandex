package v1alpha1

import (
	"context"
	"encoding/json"
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

	generateRequestWithIdentity := func(identity *YandexIdentity) admission.Request {
		objRaw, err := json.Marshal(identity)
		if err != nil {
			t.Error(err)
			return admission.Request{}
		}

		return admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Operation: admissionv1.Delete,
				OldObject: runtime.RawExtension{
					Raw: objRaw,
				},
			},
		}
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
			req: generateRequestWithIdentity(&YandexIdentity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
			}),
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
			req: generateRequestWithIdentity(&YandexIdentity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
			}),
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
			req: generateRequestWithIdentity(&YandexIdentity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
			}),
			want: admission.Allowed("identity is not linked to any cluster"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build()

			m := &yandexIdentityAdmitter{
				platformClient: fakeClient,
				decoder:        admission.NewDecoder(scheme),
			}
			if got := m.Handle(context.TODO(), tt.req); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("yandexIdentityAdmitter.Handle() = %v, want %v", got, tt.want)
			}
		})
	}

	// special cases
	t.Run("create request", func(t *testing.T) {
		wantCode := int32(200)
		m := &yandexIdentityAdmitter{
			platformClient: fake.NewClientBuilder().Build(),
			decoder:        admission.NewDecoder(runtime.NewScheme()),
		}

		got := m.Handle(context.TODO(), admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				// empty object to fail decoding
			},
		})

		if got.Result.Code != wantCode {
			t.Errorf("yandexIdentityAdmitter.Handle() = %v, want code %d", got, wantCode)
		}
	})

	t.Run("update request", func(t *testing.T) {
		wantCode := int32(200)
		m := &yandexIdentityAdmitter{
			platformClient: fake.NewClientBuilder().Build(),
			decoder:        admission.NewDecoder(runtime.NewScheme()),
		}

		got := m.Handle(context.TODO(), admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				// empty object to fail decoding
			},
		})

		if got.Result.Code != wantCode {
			t.Errorf("yandexIdentityAdmitter.Handle() = %v, want code %d", got, wantCode)
		}
	})

	t.Run("failed to decode request", func(t *testing.T) {
		wantCode := int32(400)
		m := &yandexIdentityAdmitter{
			platformClient: fake.NewClientBuilder().Build(),
			decoder:        admission.NewDecoder(runtime.NewScheme()),
		}

		got := m.Handle(context.TODO(), admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Operation: admissionv1.Delete,
				// empty object to fail decoding
			},
		})

		if got.Result.Code != wantCode {
			t.Errorf("yandexIdentityAdmitter.Handle() = %v, want code %d", got, wantCode)
		}
	})

	t.Run("failed to list clusters", func(t *testing.T) {
		wantCode := int32(500)
		m := &yandexIdentityAdmitter{
			platformClient: fake.NewClientBuilder().Build(),
			decoder:        admission.NewDecoder(scheme),
		}

		got := m.Handle(context.TODO(), generateRequestWithIdentity(&YandexIdentity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}))

		if got.Result.Code != wantCode {
			t.Errorf("yandexIdentityAdmitter.Handle() = %v, want code %d", got, wantCode)
		}
	})
}
