package scope

import (
	//nolint:gosec // not for security purpose
	"crypto/md5"
	"encoding/hex"
	"strings"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// labels mostly used in Yandex analytics to differentiate VMs created by CAPI
// CAPY means "Cluster API Provider Yandex".
const (
	// yaAnalyticsClusterNameLabel representation of CAPI "cluster.x-k8s.io/cluster-name" label.
	yaAnalyticsClusterNameLabel string = "yandex.cloud/capy-cluster-name"
	// yaAnalyticsFolderIDLabel Yandex Cloud folder Id.
	yaAnalyticsFolderIDLabel string = "yandex.cloud/folder-id"
	// yaAnalyticsControlPlaneNameLabel representation of CAPI "cluster.x-k8s.io/control-plane-name" label. Absent for workers.
	yaAnalyticsControlPlaneNameLabel string = "yandex.cloud/capy-control-plane-name"
	// yaAnalyticsMachineDeploymentNameLabel representation of CAPI "cluster.x-k8s.io/deployment-name" label. Absent for masters.
	yaAnalyticsMachineDeploymentNameLabel string = "yandex.cloud/capy-cluster-machine-deployment-name"
	// yaAnalyticsClusterHashLabel label value is the md5(folderId + clusterName) truncated to 20s.
	yaAnalyticsClusterHashLabel string = "yandex.cloud/capy-cluster-hash"
	// yaAnalyticsMachineDeploymentLabel label value is the md5(folderId + clusterName + deploymentName) truncated to 20s.
	yaAnalyticsMachineDeploymentLabel string = "yandex.cloud/capy-cluster-machine-deployment-hash"
	// managedByLabel label name identifies our controller as the VM owner.
	managedByLabel string = "yandex.cloud/managed-by"
	// capyControllerManagerName name of a capy controller manager deployment.
	capyControllerManagerName string = "capy-controller-manager"
)

// getMachineLabels prepares labels for machine in current scope.
func (m *MachineScope) getMachineLabels() map[string]string {
	labels := map[string]string{
		managedByLabel:              capyControllerManagerName,
		yaAnalyticsClusterNameLabel: m.YandexMachine.Labels[clusterv1.ClusterNameLabel],
		yaAnalyticsFolderIDLabel:    m.ClusterGetter.GetFolderID(),
	}
	deploymentName := ""
	if m.YandexMachine.Labels[clusterv1.MachineControlPlaneNameLabel] != "" {
		labels[yaAnalyticsControlPlaneNameLabel] = m.YandexMachine.Labels[clusterv1.MachineControlPlaneNameLabel]
		deploymentName = m.YandexMachine.Labels[clusterv1.MachineControlPlaneNameLabel]
	}
	if m.YandexMachine.Labels[clusterv1.MachineDeploymentNameLabel] != "" {
		labels[yaAnalyticsMachineDeploymentNameLabel] = m.YandexMachine.Labels[clusterv1.MachineDeploymentNameLabel]
		deploymentName = m.YandexMachine.Labels[clusterv1.MachineDeploymentNameLabel]
	}

	labels[yaAnalyticsClusterHashLabel] =
		getYaAnalyticsLabelHashValue(
			m.ClusterGetter.GetFolderID(),
			m.YandexMachine.Labels[clusterv1.ClusterNameLabel],
		)
	labels[yaAnalyticsMachineDeploymentLabel] =
		getYaAnalyticsLabelHashValue(
			m.ClusterGetter.GetFolderID(),
			m.YandexMachine.Labels[clusterv1.ClusterNameLabel],
			deploymentName,
		)

	return labels
}

// getYaAnalyticsLabelHashValue gets md5 from concatenated string and truncate to 20 symbols.
func getYaAnalyticsLabelHashValue(parts ...string) string {
	valueLength := 20
	//nolint:gosec // not for security purpose
	hasher := md5.New()
	hasher.Write([]byte(strings.Join(parts, "")))
	val := hex.EncodeToString(hasher.Sum(nil))

	if len(val) <= valueLength {
		return val
	}
	return val[:valueLength]
}
