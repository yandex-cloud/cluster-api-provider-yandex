/*
Copyright 2023 The Kubernetes Authors.

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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func TestYandexCluster_Default(t *testing.T) {
	tests := []struct {
		name           string
		cluster        *infrav1.YandexCluster
		expectedZoneID string
		expectedID     string
	}{
		{
			name: "internal listener with failureDomains fills subnet from first FD",
			cluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					FailureDomains: []infrav1.FailureDomainSpec{
						{ZoneID: "ru-central1-a", SubnetID: "subnet-a"},
						{ZoneID: "ru-central1-b", SubnetID: "subnet-b"},
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Internal: true,
						},
					},
				},
			},
			expectedZoneID: "ru-central1-a",
			expectedID:     "subnet-a",
		},
		{
			name: "internal listener with explicit subnet keeps it",
			cluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					FailureDomains: []infrav1.FailureDomainSpec{
						{ZoneID: "ru-central1-a", SubnetID: "subnet-a"},
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Internal: true,
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-b",
								ID:     "explicit-subnet",
							},
						},
					},
				},
			},
			expectedZoneID: "ru-central1-b",
			expectedID:     "explicit-subnet",
		},
		{
			name: "external listener does not fill subnet",
			cluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					FailureDomains: []infrav1.FailureDomainSpec{
						{ZoneID: "ru-central1-a", SubnetID: "subnet-a"},
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Internal: false,
						},
					},
				},
			},
			expectedZoneID: "",
			expectedID:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			tt.cluster.Default()
			g.Expect(tt.cluster.Spec.LoadBalancer.Listener.Subnet.ZoneID).To(Equal(tt.expectedZoneID))
			g.Expect(tt.cluster.Spec.LoadBalancer.Listener.Subnet.ID).To(Equal(tt.expectedID))
		})
	}
}

func TestYandexCluster_ValidateCreate(t *testing.T) {
	y := NewWithT(t)
	tests := []struct {
		name string
		*infrav1.YandexCluster
		wantErr bool
	}{
		{
			name: "YandexCluster with FQDN - valid",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "ya.ru",
						Port: 8443,
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "YandexCluster with IPv4 - valid",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "10.10.10.10",
						Port: 8443,
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "YandexCluster with incorrect port",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "10.10.10.10",
						Port: 844355,
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with incorrect IPv4 endpoint",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "1000.10.10.10",
						Port: 8443,
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with incorrect FQDN endpoint - 1",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "yaru",
						Port: 8443,
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with incorrect FQDN endpoint - 2",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "yaru.",
						Port: 8443,
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},

		{
			name: "YandexCluster with incorrect IPv6 - 1",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "2a02:6b8:a::a",
						Port: 8443,
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with incorrect IPv6 - 2",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "2a02:06b8:000a:0000:0000:0000:0000:000a",
						Port: 8443,
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with empty host endpoint",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "",
						Port: 8443,
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with empty port endpoint",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "ya.ru",
						Port: 0,
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with application load balancer",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Type: infrav1.LoadBalancerTypeALB,
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "YandexCluster with network load balancer",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Type: infrav1.LoadBalancerTypeNLB,
						Listener: infrav1.ListenerSpec{
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with correct IP address in loadbalancer listener spec",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Type: infrav1.LoadBalancerTypeALB,
						Listener: infrav1.ListenerSpec{
							Address: "1.1.1.1",
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "YandexCluster with incorrect IPv6 address in loadbalancer listener spec",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Type: infrav1.LoadBalancerTypeALB,
						Listener: infrav1.ListenerSpec{
							Address: "2a02:6b8:a::a",
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with incorrect IP address in loadbalancer listener spec",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Type: infrav1.LoadBalancerTypeALB,
						Listener: infrav1.ListenerSpec{
							Address: "1.1.1",
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with incorrect fqdn in loadbalancer listener spec",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Type: infrav1.LoadBalancerTypeALB,
						Listener: infrav1.ListenerSpec{
							Address: "ya.ru",
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "some-subnet-id",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "internal listener without subnet and without failureDomains - error",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Type: infrav1.LoadBalancerTypeALB,
						Listener: infrav1.ListenerSpec{
							Internal: true,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "internal listener without subnet but with failureDomains - valid",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					FailureDomains: []infrav1.FailureDomainSpec{
						{ZoneID: "ru-central1-a", SubnetID: "subnet-a"},
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Type: infrav1.LoadBalancerTypeALB,
						Listener: infrav1.ListenerSpec{
							Internal: true,
							Subnet: infrav1.SubnetSpec{
								ZoneID: "ru-central1-a",
								ID:     "subnet-a",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "external listener without subnet - valid",
			YandexCluster: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					FailureDomains: []infrav1.FailureDomainSpec{
						{ZoneID: "ru-central1-a", SubnetID: "subnet-a"},
					},
					LoadBalancer: infrav1.LoadBalancerSpec{
						Type: infrav1.LoadBalancerTypeALB,
						Listener: infrav1.ListenerSpec{
							Internal: false,
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(_ *testing.T) {
			warn, err := test.YandexCluster.ValidateCreate()
			if test.wantErr {
				y.Expect(err).To(HaveOccurred())
			} else {
				y.Expect(err).NotTo(HaveOccurred())
			}
			y.Expect(warn).To(BeNil())
		})
	}
}

func TestYandexCluster_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name        string
		newTemplate *infrav1.YandexCluster
		oldTemplate *infrav1.YandexCluster
		wantErr     bool
	}{
		{
			name: "YandexCluster with no changes in immutable fields",
			newTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "ya.ru",
						Port: 8443,
					},
				},
			},
			oldTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "ya.ru",
						Port: 8443,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "YandexCluster with changes in immutable field loadBalancer.name",
			newTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Name: "new-name",
					},
				},
			},
			oldTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Name: "old-name",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with changes in immutable field loadBalancer.listener",
			newTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Internal: false,
						},
					},
				},
			},
			oldTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Listener: infrav1.ListenerSpec{
							Internal: true,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with changes in immutable field loadBalancer.healthcheck",
			newTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Healthcheck: infrav1.HealtcheckSpec{
							HealthcheckThreshold: 1,
						},
					},
				},
			},
			oldTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					LoadBalancer: infrav1.LoadBalancerSpec{
						Healthcheck: infrav1.HealtcheckSpec{
							HealthcheckThreshold: 3,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with changes in empty field controlPlaneEndpoint",
			newTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "ya.ru",
						Port: 8443,
					},
				},
			},
			oldTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{},
			},
			wantErr: false,
		},
		{
			name: "YandexCluster change empty host to incorrect IPv4",
			newTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "1000.10.10.10",
						Port: 8443,
					},
				},
			},
			oldTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster change empty host to incorrect IPv6",
			newTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "2a02:6b8:a::a",
						Port: 8443,
					},
				},
			},
			oldTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster change empty host to incorrect FQDN",
			newTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "yaru",
						Port: 8443,
					},
				},
			},
			oldTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster change empty host to incorrect port",
			newTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "ya.ru",
						Port: 844355,
					},
				},
			},
			oldTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with changes in non empty field controlPlaneEndpoint",
			newTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "ya.ru",
						Port: 8443,
					},
				},
			},
			oldTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "yandex.com",
						Port: 8444,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "YandexCluster with empty update",
			newTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{},
				},
			},
			oldTemplate: &infrav1.YandexCluster{
				Spec: infrav1.YandexClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{},
				},
			},
			wantErr: false,
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
