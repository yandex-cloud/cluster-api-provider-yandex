package scope

import (
	"crypto/md5"
	"encoding/hex"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"strings"
)

const (
	// Pair of labels used in Yandex analytics to differentiate VMs created by CAPI
	// CAPY means "Cluster API Provider Yandex"
	yaAnalyticsClusterIdLabel   string = "capy-cluster-id"
	yaAnalyticsNodeGroupIdLabel string = "capy-node-group-id"
)

// getMachineLabels prepares labels for machine in current scope
func (m *MachineScope) getMachineLabels() map[string]string {
	labels := map[string]string{
		"managed-by": "capy-controller-manager",
		"purpose":    "capy-test",
	}
	// Label value is the md5(folderId + clusterName) truncated to 20s
	labels[yaAnalyticsClusterIdLabel] = getYaAnalyticsLabelValue(m.YandexMachine.Spec.FolderID, m.YandexMachine.Labels[clusterv1.ClusterNameLabel])

	// Label value is the md5(folderId + clusterName + deploymentName) truncated to 20s
	deploymentName := m.YandexMachine.Labels[clusterv1.MachineDeploymentNameLabel]
	if deploymentName == "" {
		deploymentName = m.YandexMachine.Labels[clusterv1.MachineControlPlaneNameLabel]
	}
	labels[yaAnalyticsNodeGroupIdLabel] = getYaAnalyticsLabelValue(m.YandexMachine.Spec.FolderID, m.YandexMachine.Labels[clusterv1.ClusterNameLabel], deploymentName)

	return labels
}

// getYaAnalyticsLabelValue gets md5 from concatenated string and truncate to 20 symbols
func getYaAnalyticsLabelValue(parts ...string) string {
	valueLength := 20
	hasher := md5.New()
	hasher.Write([]byte(strings.Join(parts, "")))
	val := hex.EncodeToString(hasher.Sum(nil))

	if len(val) <= valueLength {
		return val
	}
	return val[:valueLength]
}
