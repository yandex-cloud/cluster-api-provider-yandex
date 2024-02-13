package cloud

// ClusterGetter is an interface which can get cluster information.
type ClusterGetter interface {
	Cloud
}

// ClusterSetter is an interface which can set cluster information.
type ClusterSetter interface {
	SetReady()
}

// Cluster is an interface which can get and set cluster information.
type Cluster interface {
	ClusterGetter
	ClusterSetter
}
