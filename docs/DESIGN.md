# Overview

Doorman is a single, statically-linked binary, which reads a single YAML configuration file. From this file, Doorman obtains a list of one or more paths to Kubernetes client configuration files, and a list of one or more contexts within those files. Doorman also reads from that file a list of node "pools", each of which have one or more TCP and/or UDP ports, and a label selector. Doorman watches the Kubernetes master nodes for updates to the list of nodes which meet each pools' criteria, and if a change is detected, it generates one or more files from a set of templates, and then executes a set of system commands. Additionally, it performs these actions upon startup, after getting the initial state of each pool.

The "canonical" usage of Doorman is this:
* N master nodes, tagged as such, listening on the same port (e.g. 6443) for the API server
* M worker nodes, tagged as such, listening on some number of ports (e.g. 80, 443) for HTTP(s) ingress.
* An nginx server, which, when the list of nodes changes, will have its configuration file updated, and the server restarted. This server will act as a TCP load balancer for these ports.

Doorman is intended to be able to cover common usage cases with only a few minutes of configuration, but exensible enough to cover almost any reasonable use case.

Templates are instantiated using the golang standard library text/template.

If not configured otherwise, Doorman will:
* Perform TCP (Layer 4) load balancing of port 6443 for any node labeled as a master according to common Kubernetes distributions
* Perform TCP (Layer 4) load balancing of ports 80 and 443 for any node labeled as a worker according to common Kubernetes distributions
* Modify the file /etc/nginx/nginx.conf with a pre-made template which performs the above upon any change to those two pools, and then, in order of priority, stopping at the first successful command, "systemctl restart nginx", "docker restart nginx", and then "service nginx restart", "kill -s SIGHUP $(cat /etc/nginx/logs/nginx.pid)". By doing it, it attempts each reasonable way to restart the nginx server.

# Implementation

Being a golang program, Doorman makes heavy use of goroutines and channels. The main goroutine waits for an event on one or more channels, such an event indicates that a node has joined or left a node pool, or that the Doorman configuration file should be reloaded. This main goroutine also maintains a list of each node pool. Each node pool has its own goroutine, where it receives events from a Kubernetes client watch channel, and maintains the internal list of nodes in that pool. When the list of nodes has changed, it sends its output channel the new complete list. When a watch event that should trigger an update occurs, it it received by the gouroutine for the corresponding pool. That pool's goroutine determines and saves the new list of nodes in the pool, and sends that list to its output channel. The main goroutine receives this new list, replaces the corresponding node pool, and then instantiates each configuration file template using the complete list of pools, plus any appropriate metadata, and saves the results to the matching file paths. It then executes each of the configured system commands.
