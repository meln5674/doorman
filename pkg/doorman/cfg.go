package doorman

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigFile contains the structure parsed from the YAML config file
type ConfigFile struct {
	Kubernetes *KubernetesConfigFile `json:"kubernetes"`
	Templates  []Template            `json:"templates"`
	Health     *HealthConfigFile     `json:"health"`
	Metrics    *MetricsConfigFile    `json:"metrics"`
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

// NodePoolConfigFile is the node pool section of the config field. For a node to be part of the pool, it must match one or more of the elements of the selector array.
type NodePoolConfigFile struct {
	Name          string     `json:"name"`
	TCPPorts      []int      `json:"tcpPorts"`
	UDPPorts      []int      `json:"udpPorts"`
	NodeSelectors []Selector `json:"nodeSelector"`
	AddressType   corev1.NodeAddressType
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
