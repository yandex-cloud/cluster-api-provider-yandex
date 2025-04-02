/*
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

package v1alpha1_test

import (
	"testing"

	. "github.com/onsi/gomega"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestYandexMachineTemplate_ValidateCreate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name     string
		template *infrav1.YandexMachineTemplate
		wantErr  bool
	}{
		{
			name: "valid template create",
			template: &infrav1.YandexMachineTemplate{
				ObjectMeta: v1.ObjectMeta{Name: "ymt-valid"},
				Spec: infrav1.YandexMachineTemplateSpec{
					Template: infrav1.YandexMachineTemplateResource{
						Spec: infrav1.YandexMachineSpec{
							ZoneID: ptr.To("ru-central1-a"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "template with invalid symbols in name create",
			template: &infrav1.YandexMachineTemplate{
				ObjectMeta: v1.ObjectMeta{Name: "ymt-invalid-v1.0"},
				Spec: infrav1.YandexMachineTemplateSpec{
					Template: infrav1.YandexMachineTemplateResource{
						Spec: infrav1.YandexMachineSpec{
							ZoneID: ptr.To("ru-central1-a"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "template with invalid length name create",
			template: &infrav1.YandexMachineTemplate{
				ObjectMeta: v1.ObjectMeta{Name: "ymt-very-looooooooooooooooooooooooooooooooooooooooooooooong-name"},
				Spec: infrav1.YandexMachineTemplateSpec{
					Template: infrav1.YandexMachineTemplateResource{
						Spec: infrav1.YandexMachineSpec{
							ZoneID: ptr.To("ru-central1-a"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "template with invalid first charter name create",
			template: &infrav1.YandexMachineTemplate{
				ObjectMeta: v1.ObjectMeta{Name: "8yml-invalid"},
				Spec: infrav1.YandexMachineTemplateSpec{
					Template: infrav1.YandexMachineTemplateResource{
						Spec: infrav1.YandexMachineSpec{
							ZoneID: ptr.To("ru-central1-a"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "template with invalid last charter name create",
			template: &infrav1.YandexMachineTemplate{
				ObjectMeta: v1.ObjectMeta{Name: "yml-invalid-"},
				Spec: infrav1.YandexMachineTemplateSpec{
					Template: infrav1.YandexMachineTemplateResource{
						Spec: infrav1.YandexMachineSpec{
							ZoneID: ptr.To("ru-central1-a"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "template with invalid providerID create",
			template: &infrav1.YandexMachineTemplate{
				ObjectMeta: v1.ObjectMeta{Name: "ymt-valid"},
				Spec: infrav1.YandexMachineTemplateSpec{
					Template: infrav1.YandexMachineTemplateResource{
						Spec: infrav1.YandexMachineSpec{
							ZoneID:     ptr.To("ru-central1-a"),
							ProviderID: ptr.To("yandex://1234567"),
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(_ *testing.T) {
			warn, err := test.template.ValidateCreate()

			if test.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(warn).To(BeNil())
		})
	}
}

func TestYandexMachineTemplate_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name        string
		oldTemplate *infrav1.YandexMachineTemplate
		newTemplate *infrav1.YandexMachineTemplate
		wantErr     bool
	}{
		{
			name: "change in mutable fields",
			oldTemplate: &infrav1.YandexMachineTemplate{
				ObjectMeta: v1.ObjectMeta{Name: "ymt-old"},
				Spec: infrav1.YandexMachineTemplateSpec{
					Template: infrav1.YandexMachineTemplateResource{
						Spec: infrav1.YandexMachineSpec{
							ZoneID: ptr.To("ru-central1-a"),
						},
					},
				},
			},
			newTemplate: &infrav1.YandexMachineTemplate{
				ObjectMeta: v1.ObjectMeta{Name: "ymt-new"},
				Spec: infrav1.YandexMachineTemplateSpec{
					Template: infrav1.YandexMachineTemplateResource{
						Spec: infrav1.YandexMachineSpec{
							ZoneID: ptr.To("ru-central1-a"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change in immutable fields",
			oldTemplate: &infrav1.YandexMachineTemplate{
				ObjectMeta: v1.ObjectMeta{Name: "ymt-test"},
				Spec: infrav1.YandexMachineTemplateSpec{
					Template: infrav1.YandexMachineTemplateResource{
						Spec: infrav1.YandexMachineSpec{
							ZoneID: ptr.To("ru-central1-a"),
						},
					},
				},
			},
			newTemplate: &infrav1.YandexMachineTemplate{
				ObjectMeta: v1.ObjectMeta{Name: "ymt-test"},
				Spec: infrav1.YandexMachineTemplateSpec{
					Template: infrav1.YandexMachineTemplateResource{
						Spec: infrav1.YandexMachineSpec{
							ZoneID: ptr.To("ru-central1-b"),
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(_ *testing.T) {
			warn, err := test.newTemplate.ValidateUpdate(test.oldTemplate)

			if test.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(warn).To(BeNil())
		})
	}
}
