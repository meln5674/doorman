package internal

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	corev1api "k8s.io/client-go/kubernetes/typed/core/v1"
)

type NodePoolDescription struct {
	name        string
	tcpPorts    []int
	udpPorts    []int
	selectors   []Selector
	addressType corev1.NodeAddressType
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
									TCP: &port,
								},
								Address: address.Address,
							}
						}
						for _, port := range p.pool.udpPorts {
							events <- NodeEvent{
								Type: watch.Added,
								Port: Port{
									UDP: &port,
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
