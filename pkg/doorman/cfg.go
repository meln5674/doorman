package doorman

import (
	corev1 "k8s.io/api/core/v1"
)

// ConfigFile contains the structure parsed from the YAML config file
type ConfigFile struct {
	Kubernetes *KubernetesConfigFile `json:"kubernetes"`
	Templates  []TemplateConfigFile  `json:"templates"`
	Health     *HealthConfigFile     `json:"health"`
	Metrics    *MetricsConfigFile    `json:"metrics"`
}

type HealthConfigFile struct {
	Port int `json:"health"`
}

type MetricsConfigFile struct {
	Port int `json:"health"`
}

// KubernetesConfigFile contains the configuration for reaching one or more Kubernetes master nodes
type KubernetesConfigFile struct {
	KubeconfigPaths []string `json:"kubeconfigPaths"`
	Contexts        []string `json:"contexts"`
}

type NodePoolConfigFile struct {
	Name         string             `json:"name"`
	TCPPorts     []int              `json:"tcpPorts"`
	UDPPorts     []int              `json:"udpPorts"`
	NodeSelector SelectorConfigFile `json:"nodeSelector"`
	AddressType  corev1.NodeAddressType
}

type SelectorConfigFile struct {
	Labels *map[string]string `json:"labels"`
	Fields *map[string]string `json:"fields"`
}

// TemplateConfigFile contains the configuration for templating a file with node information
type TemplateConfigFile struct {
	Template string `json:"template"`
	Path     string `json:"path"`
	Engine   string `json:"engine"`
}
