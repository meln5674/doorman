package internal

import (
	"k8s.io/apimachinery/pkg/watch"
)

type NodeEvent struct {
	Type    watch.EventType
	Port    Port
	Address string
}
