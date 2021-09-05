package internal

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"os"
	"path"

	"k8s.io/client-go/kubernetes"
	k8sconfig "k8s.io/client-go/tools/clientcmd"
	//k8sconfigapi "k8s.io/client-go/tools/clientcmd/api"

	public "github.com/meln5674/doorman/pkg/doorman"
)

// TODO: Make some of this public so that new template factories and actions can
// be added without modifying this directory

var TemplateFactories map[string]TemplateFactory = make(map[string]TemplateFactory)

type TemplateFactory interface {
	Parse(template string, path string) (Templater, error)
}

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
	} else {
		kubeconfigPath := os.Getenv(k8sconfig.RecommendedConfigPathEnvVar)
		if kubeconfigPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			kubeconfigPath = path.Join(home, k8sconfig.RecommendedHomeDir, k8sconfig.RecommendedFileName)
		}
		config, err := k8sconfig.BuildConfigFromFlags("", kubeconfigPath)
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

	d.nodePools = make([]NodePoolDescription, len(cfg.NodePools))
	for i, pool := range cfg.NodePools {
		d.nodePools[i].FromConfig(&pool)
	}

	d.templates = make([]Templater, 0, len(cfg.Templates))
	for _, template := range cfg.Templates {
		factory, ok := TemplateFactories[template.Engine]
		if !ok {
			return fmt.Errorf("Unrecognized template engine: %s", template.Engine)
		}
		tpl, err := factory.Parse(template.Template, template.Path)
		if err != nil {
			return err
		}
		d.templates = append(d.templates, tpl)
	}
	d.actions = make([]Action, 1)
	d.actions[0] = &BlindNginxRestartAction{}
	if cfg.Health != nil {
		// TODO: set up http health endpoint handler
	}
	if cfg.Metrics != nil {
		// TODO: set up prometheus metrics
	}

	return nil
}

type portPool struct {
	addresses map[string]struct{}
	destPort  int
}

type portPools map[int]portPool

func (p portPools) init(port PortMapping) {
	p[port.Source] = portPool{addresses: make(map[string]struct{}), destPort: port.Dest}
}

func (p portPools) add(port int, address string) (added bool) {
	_, ok := p[port].addresses[address]
	added = !ok
	p[port].addresses[address] = struct{}{}
	return
}

func (p portPools) remove(port int, address string) (removed bool) {
	_, ok := p[port].addresses[address]
	removed = ok
	delete(p[port].addresses, address)
	return
}

func (p portPools) render() []PortVars {
	ports := make([]PortVars, len(p))
	for port, pool := range p {
		addressList := make([]string, 0, len(pool.addresses))
		for address, _ := range pool.addresses {
			addressList = append(addressList, address)
		}
		ports = append(ports, PortVars{SourcePort: port, DestPort: pool.destPort, Addresses: addressList})
	}
	return ports
}

type PortVars struct {
	SourcePort int      `json:"srcPort"`
	DestPort   int      `json:"destPort"`
	Addresses  []string `json:"addresses"`
}

type TemplateVars struct {
	TCPPorts []PortVars `json:"tcp"`
	UDPPorts []PortVars `json:"udp"`
}

func (d *Doorman) Run(ctx context.Context, stop <-chan struct{}) error {
	tcpPools := make(portPools)
	udpPools := make(portPools)
	events := make(chan NodeEvent)
	fmt.Println("Starting node pool watchers")
	for _, pool := range d.nodePools {
		fmt.Printf("Starting watches for pool %s\n", pool.name)
		go func(pool NodePoolDescription) {
			err := (&PoolWatcher{
				kubernetesAPIs: d.kubernetesAPIs,
				pool:           pool,
			}).Run(ctx, events, stop)
			if err != nil {
				fmt.Printf("Watcher failed: %v\n", err)
			}
		}(pool)
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

	fmt.Println("Listening for events from watchers...")

	for event := range events {
		fmt.Printf("Got event %#v\n", event)
		var pools portPools
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
			fmt.Print("Event did not change state, not regenerating templates")
			continue
		}
		// TODO: Implement some sort of throttling so that only one re-template
		// happens per "chunk" of activity
		fmt.Println("Regenerating templates")
		templateVars := TemplateVars{
			TCPPorts: tcpPools.render(),
			UDPPorts: udpPools.render(),
		}
		for _, templater := range d.templates {
			err := templater.Template(templateVars)
			if err != nil {
				fmt.Printf("Templating failed: %v\n", err)
			}
		}
		fmt.Println("Performing post-template actions")
		for _, action := range d.actions {
			err := action.Do(ctx)
			if err != nil {
				fmt.Printf("Failed post-template action: %v\n", err)
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
	Do(context.Context) error
}
