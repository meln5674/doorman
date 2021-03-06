package doorman

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigFile contains the structure parsed from the YAML config file
type ConfigFile struct {
	Kubernetes *KubernetesConfigFile `json:"kubernetes"`
	NodePools  []NodePoolConfigFile  `json:"nodePools"`
	Templates  []Template            `json:"templates"`
	Health     *HealthConfigFile     `json:"health"`
	Metrics    *MetricsConfigFile    `json:"metrics"`
	// TODO: Add ability to configure the post-template action(s)
}

// HealthConfigFile is the health endpoint section of the config file
type HealthConfigFile struct {
	Port int `json:"health"`
}

// MetricsConfigFile is the metrics section of the config file
type MetricsConfigFile struct {
	Port int `json:"health"`
}

// KubernetesConfigFile contains the configuration for reaching one or more Kubernetes master nodes
type KubernetesConfigFile struct {
	KubeconfigPaths []string `json:"kubeconfigPaths"`
	Contexts        []string `json:"contexts"`
}

// PortMapping is a mapping from a port on one host to a port on another. If dest is absent, the source is assumed to be the dest.
type PortMapping struct {
	Source int  `json:"src"`
	Dest   *int `json:"dest"`
}

// NodePoolConfigFile is the node pool section of the config field. For a node to be part of the pool, it must match one or more of the elements of the selector array.
type NodePoolConfigFile struct {
	Name          string                 `json:"name"`
	TCPPorts      []PortMapping          `json:"tcpPorts"`
	UDPPorts      []PortMapping          `json:"udpPorts"`
	NodeSelectors []Selector             `json:"nodeSelectors"`
	AddressType   corev1.NodeAddressType `json:"addressType"`
	// TODO: Add ability to specify nodeport range(s) to map to these nodes
}

// FieldSelector describes a kubernetes field selector
type FieldSelector struct {
	Key    string
	Value  string
	Negate bool
}

// AsString converts a field selector to the query param to provide to the API
func (f *FieldSelector) AsString() string {
	if f.Negate {
		return fmt.Sprintf("%s=%s", f.Key, f.Value)
	} else {
		return fmt.Sprintf("%s!=%s", f.Key, f.Value)
	}
}

// FieldSelectorsAsString converts an array of field selectors to the query param to provide to the API
func FieldSelectorsAsString(selectors []FieldSelector) string {
	if len(selectors) == 0 {
		return ""
	}
	str := selectors[0].AsString()
	for _, selector := range selectors {
		str += ","
		str += selector.AsString()
	}
	return str
}

// Selector is a label and/or field selector. For the selector to be matched, both selectors much match, and all components of those selectors must match.
type Selector struct {
	Labels *metav1.LabelSelector `json:"labels"`
	Fields *[]FieldSelector      `json:"fields"`
}

// Template contains the configuration for templating a file with node information
type Template struct {
	Template string `json:"template"`
	Path     string `json:"path"`
	Engine   string `json:"engine"`
}
