/*
Copyright 2024.

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

func TestYandexMachine_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name       string
		oldMachine *infrav1.YandexMachine
		newMachine *infrav1.YandexMachine
		wantErr    bool
	}{
		{
			// mutabe fields: providerID
			name: "change in mutable fields",
			oldMachine: &infrav1.YandexMachine{
				ObjectMeta: v1.ObjectMeta{Name: "ym-test"},
				Spec: infrav1.YandexMachineSpec{
					ProviderID: nil,
				},
			},
			newMachine: &infrav1.YandexMachine{
				ObjectMeta: v1.ObjectMeta{Name: "ym-test"},
				Spec: infrav1.YandexMachineSpec{
					ProviderID: ptr.To("yandex://instance-id"),
				},
			},
			wantErr: false,
		},
		{
			name: "change in immutable fields",
			oldMachine: &infrav1.YandexMachine{
				ObjectMeta: v1.ObjectMeta{Name: "ym-test"},
				Spec: infrav1.YandexMachineSpec{
					ProviderID: nil,
					ZoneID:     ptr.To("ru-central1-a"),
				},
			},
			newMachine: &infrav1.YandexMachine{
				ObjectMeta: v1.ObjectMeta{Name: "ym-test"},
				Spec: infrav1.YandexMachineSpec{
					ProviderID: ptr.To("yandex://instance-id"),
					ZoneID:     ptr.To("ru-central1-b"),
				},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(_ *testing.T) {
			warn, err := test.newMachine.ValidateUpdate(test.oldMachine)
			if test.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(warn).To(BeNil())
		})
	}
}
