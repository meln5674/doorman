package internal

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	corev1api "k8s.io/client-go/kubernetes/typed/core/v1"

	public "github.com/meln5674/doorman/pkg/doorman"
)

type PortMapping struct {
	Source int
	Dest   int
}

func (p *PortMapping) FromConfig(cfg public.PortMapping) {
	p.Source = cfg.Source
	if cfg.Dest == nil {
		p.Dest = cfg.Source
	} else {
		p.Dest = *cfg.Dest
	}
}

type NodePoolDescription struct {
	name        string
	tcpPorts    []PortMapping
	udpPorts    []PortMapping
	selectors   []Selector
	addressType corev1.NodeAddressType
}

func (n *NodePoolDescription) FromConfig(cfg *public.NodePoolConfigFile) error {
	n.name = cfg.Name
	n.tcpPorts = make([]PortMapping, 0, len(cfg.TCPPorts))
	for i, port := range cfg.TCPPorts {
		(&n.tcpPorts[i]).FromConfig(port)
	}
	n.udpPorts = make([]PortMapping, 0, len(cfg.UDPPorts))
	for i, port := range cfg.UDPPorts {
		(&n.udpPorts[i]).FromConfig(port)
	}
	n.selectors = make([]Selector, len(cfg.NodeSelectors))
	for i, selector := range cfg.NodeSelectors {
		if selector.Labels != nil {
			n.selectors[i].labelSelector = metav1.FormatLabelSelector(selector.Labels)
		}
		if selector.Fields != nil {
			n.selectors[i].fieldSelector = public.FieldSelectorsAsString(*selector.Fields)
		}
	}
	n.addressType = cfg.AddressType
	return nil
}

type Selector struct {
	labelSelector string
	fieldSelector string
}

type PoolWatcher struct {
	kubernetesAPIs []corev1api.NodeInterface
	pool           NodePoolDescription
}

func (p *PoolWatcher) Run(ctx context.Context, events chan<- NodeEvent, stop <-chan struct{}) error {
	watchEvents := make(chan watch.Event)
	defer close(watchEvents)
	for _, api := range p.kubernetesAPIs {
		for _, selector := range p.pool.selectors {
			watcher, err := api.Watch(ctx, metav1.ListOptions{LabelSelector: selector.labelSelector, FieldSelector: selector.fieldSelector})
			if err != nil {
				return err
			}
			defer watcher.Stop()
			go func() {
				for event := range watcher.ResultChan() {
					watchEvents <- event
				}
			}()
		}
	}
	running := true
	nodes := make(map[string]*corev1.Node)
	for running {
		select {
		case watchEvent := <-watchEvents:
			node := watchEvent.Object.(*corev1.Node)
			switch watchEvent.Type {
			case watch.Added:
				nodes[node.Name] = node
				for _, address := range node.Status.Addresses {
					if address.Type == p.pool.addressType {
						for _, port := range p.pool.tcpPorts {
							events <- NodeEvent{
								Type: watch.Added,
								Port: Port{
									TCP: &port.Source,
								},
								Address: address.Address,
							}
						}
						for _, port := range p.pool.udpPorts {
							events <- NodeEvent{
								Type: watch.Added,
								Port: Port{
									UDP: &port.Source,
								},
								Address: address.Address,
							}
						}
					}
				}
				// TODO: Handle remaining types
				// Deleted: Send delete event for last known set of addresses for each port
				// Modified: Compare new addresses vs last known addresses, send added/deleted as necessary
				// Error: Pass error
			}
		case <-stop:
			running = false
		}
	}
	return nil
}
