# !!! This project has not reached MVP, do not use in a production environment !!!

# Doorman

Doorman makes it fast and easy to load balance Kubernetes clusters in any cloud or on-premise environment.

See [docs/DESIGN.md] for an explanation of Doorman.

## Building

Prerequisites:
* Go
* Make
* Docker (Integration tests only)
* Kind (Integration tests only)
* Kubectl (Integration tests only)
* Helm (Integration tests only)

```bash
make clean all
# No tests
make clean bin/doorman # amd64
make clean bin/doorman-arm64 # arm64 (Raspberry Pi)
make clean bin/doorman # Windows (Seek help)
# Integration Tests only
make integration-test
```

## Installation

### Quick Start

These methods are not comprehensive, and make broad assumptions.

```bash
# Both methods assume:
# * Your nginx configuration file is at /etc/nginx/nginx.conf on the host
# * A kubeconfig with a context for each of your k8s masters are at /var/www/.kube/config

# Systemd, assumes you have a systemd service called "nginx". Requires root/sudo
make install-systemd
# Docker, assumes you have an nginx container named "nginx" with /etc/nginx mounted. Require root/sudo/docker socket access
make install-docker
```

### Custom

* Create/copy one or more kubeconfig files containing contexts capable of reaching each k8s master you wish to read from. (While each master will have the same information, having multiple provides redundancy in case of individual master failure of maintenance)
* Create your doorman.yaml file (See docs/example/default.yaml for an example and documentation on supported fields).
    * Specify the path(s) to each of your kubeconfig(s), and optionally a subset of the context(s) you wish to use.
    * Specify the selectors for your node pools and which ports to forward for each
    * Modify the default nginx configuration template file, and set the correct path to write the instantiated template to.
* Set the doorman binary to run at server startup, and to restart on failure
* Ensure that the doorman process has permissions to restart your nginx server

## Uninstallation

Only use these if you used the installation methods in "Quick Start"

```bash
# Systemd, requires root/sudo
make uninstall-systemd
# Docker, requires root/sudo/docker socket access
make uninstall-docker
```
