package scope

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SetNodeProviderID sets the providerID on the Kubernetes Node object.
func (m *MachineScope) SetNodeProviderID(ctx context.Context) error {
	cluster, err := util.GetClusterFromMetadata(ctx, m.client, m.Machine.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster from machine metadata")
	}

	if m.Machine.Spec.ProviderID == nil {
		return errors.New("providerid is not set for machine yet")
	}

	clusterClient, err := m.getManagedClusterClient(ctx, cluster)
	if err != nil {
		return err
	}

	nodelist := &corev1.NodeList{}
	if err = clusterClient.List(ctx, nodelist); err != nil {
		return errors.Wrap(err, "failed to list nodes")
	}

	var currentNode *corev1.Node
	for _, n := range nodelist.Items {
		// node hostname can contain domain name, but machineName name does not
		if strings.HasPrefix(n.Name, m.YandexMachine.Name+".") {
			currentNode = n.DeepCopy()

			break
			// else compare directly
		} else if n.Name == m.YandexMachine.Name {
			currentNode = n.DeepCopy()

			break
		}
	}

	if currentNode == nil {
		return errors.New("node not found")
	}

	if currentNode.Spec.ProviderID == "" {
		// KubeAPI will not allow to change .Spec.ProviderID if not empty
		original := currentNode.DeepCopy()
		currentNode.Spec.ProviderID = m.GetProviderID()

		if err := clusterClient.Patch(ctx, currentNode, client.MergeFrom(original)); err != nil {
			return errors.Wrap(err, "failed to patch node")
		}
	}

	return nil
}

// getManagedClusterClient returns a client for the managed cluster.
func (m *MachineScope) getManagedClusterClient(ctx context.Context, cluster *clusterv1.Cluster) (client.Client, error) {
	credsSecret := &corev1.Secret{}

	credsSecretName := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-kubeconfig",
	}

	if err := m.client.Get(ctx, credsSecretName, credsSecret); err != nil {
		return nil, errors.New("failed to get secret with cluster config")
	}

	rawKubeconfig, ok := credsSecret.Data["value"]
	if !ok {
		return nil, errors.New("failed to get kubeconfig from secret")
	}

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "failed to add corev1 to scheme")
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(rawKubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rest config from kubeconfig")
	}

	clusterClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client from rest config")
	}

	return clusterClient, nil
}
