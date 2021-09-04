package internal

import (
	"context"
	"k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"os"

	"k8s.io/client-go/kubernetes"
	k8sconfig "k8s.io/client-go/tools/clientcmd"
	//k8sconfigapi "k8s.io/client-go/tools/clientcmd/api"

	public "github.com/meln5674/doorman/pkg/doorman"
)

// Doorman is the data parsed from a ConfigFile
type Doorman struct {
	kubernetesAPIs []corev1.NodeInterface
	nodePools      []NodePoolDescription
	templates      []Templater
	actions        []Action
	health         *HealthEndpoint
	metrics        *MetricsEndpoint
}

type HealthEndpoint struct {
}

type MetricsEndpoint struct {
}

func (d *Doorman) FromConfig(cfg *public.ConfigFile) error {
	if cfg.Kubernetes != nil {
		loadingRules := k8sconfig.NewDefaultClientConfigLoadingRules()
		loadingRules.Precedence = cfg.Kubernetes.KubeconfigPaths
		allKubeConfigs := k8sconfig.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &k8sconfig.ConfigOverrides{})
		allConfigs, err := allKubeConfigs.RawConfig()
		if err != nil {
			return err
		}
		var contextFilter map[string]struct{}
		if len(cfg.Kubernetes.Contexts) == 0 {
			d.kubernetesAPIs = make([]corev1.NodeInterface, 0, len(allConfigs.Contexts))
			contextFilter = make(map[string]struct{}, len(allConfigs.Contexts))
			for contextName, _ := range allConfigs.Contexts {
				contextFilter[contextName] = struct{}{}
			}
		} else {
			d.kubernetesAPIs = make([]corev1.NodeInterface, 0, len(cfg.Kubernetes.Contexts))
			contextFilter = make(map[string]struct{}, 0)
		}
		for contextName, context := range allConfigs.Contexts {
			if _, ok := contextFilter[contextName]; !ok && len(cfg.Kubernetes.Contexts) != 0 {
				continue
			}
			delete(contextFilter, contextName)
			kubeConfig := k8sconfig.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &k8sconfig.ConfigOverrides{Context: *context})
			config, err := kubeConfig.ClientConfig()
			if err != nil {
				return err
			}
			client, err := kubernetes.NewForConfig(config)
			if err != nil {
				return err
			}
			d.kubernetesAPIs = append(d.kubernetesAPIs, client.CoreV1().Nodes())
		}
		// TODO: validate no contexts are present multiple times
		// TODO: populate d.kubernetesAPIs from contexts within all loaded kubeconfigs
	} else {
		config, err := k8sconfig.BuildConfigFromFlags("", os.Getenv(k8sconfig.RecommendedConfigPathEnvVar))
		if err != nil {
			return err
		}

		d.kubernetesAPIs = make([]corev1.NodeInterface, 1)
		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}
		d.kubernetesAPIs[0] = client.CoreV1().Nodes()
	}
	for _ /*, template :*/ = range cfg.Templates {
		// TODO: populate d.templates
	}
	// TODO: Create default action which tries to restart nginx
	if cfg.Health != nil {
		// TODO: set up http health endpoint handler
	}
	if cfg.Metrics != nil {
		// TODO: set up prometheus metrics
	}

	return nil
}

type portPool map[int]map[string]struct{}

func (p portPool) init(port int) {
	p[port] = make(map[string]struct{})
}

func (p portPool) add(port int, address string) (added bool) {
	_, ok := p[port][address]
	added = !ok
	p[port][address] = struct{}{}
	return
}

func (p portPool) remove(port int, address string) (removed bool) {
	_, ok := p[port][address]
	removed = ok
	delete(p[port], address)
	return
}

func (p portPool) render() []PortVars {
	ports := make([]PortVars, len(p))
	for port, addressSet := range p {
		addressList := make([]string, 0, len(addressSet))
		for address, _ := range addressSet {
			addressList = append(addressList, address)
		}
		ports = append(ports, PortVars{Port: port, Addresses: addressList})
	}
	return ports
}

type PortVars struct {
	Port      int      `json:"port"`
	Addresses []string `json:"addresses"`
}

type TemplateVars struct {
	TCPPorts []PortVars `json:"tcp"`
	UDPPorts []PortVars `json:"udp"`
}

func (d *Doorman) Run(ctx context.Context, stop <-chan struct{}) error {
	tcpPools := make(portPool)
	udpPools := make(portPool)
	events := make(chan NodeEvent)
	for _, pool := range d.nodePools {
		go func() {
			err := (&PoolWatcher{
				kubernetesAPIs: d.kubernetesAPIs,
				pool:           pool,
			}).Run(ctx, events, stop)
			if err != nil {
				// TODO: Handle error
			}
		}()
		for _, port := range pool.tcpPorts {
			tcpPools.init(port)
		}
		for _, port := range pool.udpPorts {
			udpPools.init(port)
		}
	}
	// TODO: Serve health endpoint
	// TODO: Serve metrics
	// TODO: Define and populate metrics

	for event := range events {
		var pools portPool
		var port int
		if event.Port.TCP != nil {
			pools = tcpPools
			port = *event.Port.TCP
		} else {
			pools = udpPools
			port = *event.Port.UDP
		}
		updated := false
		switch event.Type {
		case watch.Added:
			updated = pools.add(port, event.Address)
		case watch.Deleted:
			updated = pools.remove(port, event.Address)
			// TODO: Handle remaining events
			// Error: ???
		}
		if !updated {
			continue
		}
		// TODO: Implement some sort of throttling so that only one re-template
		// happens per "chunk" of activity

		templateVars := TemplateVars{
			TCPPorts: tcpPools.render(),
			UDPPorts: udpPools.render(),
		}
		for _, templater := range d.templates {
			err := templater.Template(templateVars)
			if err != nil {
				// TODO: Handle error
			}
		}
		for _, action := range d.actions {
			err := action.Do()
			if err != nil {
				// TODO: Handle error
			}
		}
	}
	return nil
}

type Port struct {
	TCP *int
	UDP *int
}

// Templater intantiates a template using variables
type Templater interface {
	Template(in interface{}) error
}

type Action interface {
	Do() error
}
