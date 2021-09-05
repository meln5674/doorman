package internal

import (
	"context"
	"fmt"
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
	n.tcpPorts = make([]PortMapping, len(cfg.TCPPorts))
	for i, port := range cfg.TCPPorts {
		(&n.tcpPorts[i]).FromConfig(port)
	}
	n.udpPorts = make([]PortMapping, len(cfg.UDPPorts))
	for i, port := range cfg.UDPPorts {
		(&n.udpPorts[i]).FromConfig(port)
	}
	n.selectors = make([]Selector, len(cfg.NodeSelectors))
	for i, selector := range cfg.NodeSelectors {
		if selector.Labels != nil {
			selectorI, err := metav1.LabelSelectorAsSelector(selector.Labels)
			if err != nil {
				return err
			}
			n.selectors[i].labelSelector = selectorI.String()
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

func (p *PoolWatcher) processNewNode(node *corev1.Node, events chan<- NodeEvent) {
	fmt.Printf("New Node: %s, %s, %v, %v\n", node.Name, p.pool.addressType, node.Status.Addresses, p.pool.tcpPorts)
	for _, address := range node.Status.Addresses {
		if address.Type == p.pool.addressType {
			for _, port := range p.pool.tcpPorts {
				fmt.Printf("New TCP Port: %d->%s:%d\n", port.Source, address.Address, port.Dest)
				events <- NodeEvent{
					Type: watch.Added,
					Port: Port{
						TCP: &port.Source,
					},
					Address: address.Address,
				}
			}
			for _, port := range p.pool.udpPorts {
				fmt.Printf("New UDP Port: %d->%s:%d\n", port.Source, address.Address, port.Dest)
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

}

func (p *PoolWatcher) Run(ctx context.Context, events chan<- NodeEvent, stop <-chan struct{}) error {
	fmt.Printf("Starting watches for %#v\b", *p)
	watchEvents := make(chan watch.Event)
	defer close(watchEvents)
	nodes := make(map[string]*corev1.Node)
	initialList := make([][]corev1.Node, 0, len(p.pool.selectors))
	for _, api := range p.kubernetesAPIs {
		for i, selector := range p.pool.selectors {
			fmt.Printf("Getting initial list of nodes for pool %s: --selector=%s --fieldSelector=%s\n", p.pool.name, selector.labelSelector, selector.fieldSelector)
			options := metav1.ListOptions{LabelSelector: selector.labelSelector, FieldSelector: selector.fieldSelector}
			if i >= len(initialList) {
				nodes, err := api.List(ctx, options)
				if err != nil {
					fmt.Printf("Listing nodes failed: %v\n", err)
					continue // Maybe another call will succeed?
				}
				initialList = append(initialList, nodes.Items)
			}
			watcher, err := api.Watch(ctx, options)
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

	if len(initialList) != len(p.pool.selectors) {
		return fmt.Errorf("Did not get an initial list for all selectors")
	}

	initialProcessedNodeNames := make(map[string]struct{})
	for _, nodes := range initialList {
		for _, node := range nodes {
			if _, ok := initialProcessedNodeNames[node.Name]; ok {
				continue
			}
			initialProcessedNodeNames[node.Name] = struct{}{}
			p.processNewNode(&node, events)
		}
	}

	running := true

	for running {
		select {
		case watchEvent := <-watchEvents:
			node := watchEvent.Object.(*corev1.Node)
			switch watchEvent.Type {
			case watch.Added:
				nodes[node.Name] = node
				p.processNewNode(node, events)
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
