#!/bin/bash -xe

cluster_name=doorman-integration-test
image_repo=doorman/integration-test
test_dir=hack/integration-test
container_home=/var/www
container_kube_dir="${container_home}/.kube"
container_kubeconfig="${container_kube_dir}/config"

cert_manager_chart_version=v1.5.3 
nginx_ingress_controller_chart_version=v4.0.1
nginx_chart_version=v9.5.3

function in-container {
	docker exec "${cluster_name}-control-plane" "$@"
}

function cp-container {
	docker cp "$1" "${cluster_name}-control-plane:$2"
}

function cluster-exists {
	kind get clusters | grep -Eq "^${cluster_name}\$"
}

function release-exists {
	helm list | grep -Eq "^$1\$"
}

export KUBECONFIG=bin/integration-test.kubeconfig

start_cluster=

if [ -n "${DOORMAN_INTEGRATION_TEST_REUSE}" ]; then
	if cluster-exists; then
		echo 'Reusing existing test cluster'
		cp-container bin/doorman /usr/local/bin/doorman
		cp-container hack/integration-test/doorman.yaml /etc/nginx/doorman.yaml
		cp-container hack/integration-test/doorman.service /etc/systemd/system/doorman.service
		in-container systemctl daemon-reload
		in-container systemctl restart doorman
	else
		echo 'No existing test cluster to re-use'
		start_cluster=1
	fi	
fi

if [ -z "${DOORMAN_INTEGRATION_TEST_REUSE}" ] || [ -n "${start_cluster}" ]; then
	echo 'Building test image...'
	build_timestamp=$(date +%s)
	integration_test_image="${image_repo}:${build_timestamp}" 
	docker build \
		--quiet \
		--file="${test_dir}/Dockerfile" \
		--tag="${integration_test_image}" \
		.
	
	if cluster-exists; then
		echo 'Deleting old test cluster...'
		kind delete cluster --name="${cluster_name}"
	fi

	echo 'Creating test cluster...'
	kind create cluster \
		--name="${cluster_name}" \
		--image="${integration_test_image}" \
		--kubeconfig="${KUBECONFIG}"
	if [ -z "${DOORMAN_INTEGRATION_TEST_DEBUG}" ]; then
		trap 'kind delete cluster --name=${cluster_name}' EXIT
	fi
fi

echo 'Creating kubeconfig...'
in-container mkdir -p "${container_kube_dir}"
in-container chown www-data "${container_kube_dir}"
in-container chown www-data "/etc/nginx/nginx.conf"
cp-container "${KUBECONFIG}" "${container_kubeconfig}"
in-container chown www-data "${container_kubeconfig}"
in-container sed -Ei 's|server: https://127.0.0.1:[[:digit:]]+|server: https://127.0.0.1:6443|' "${container_kubeconfig}" 

echo 'Ensuring doormain is running...'
in-container systemctl restart doorman || doorman_status=$?
if [ "${doorman_status}" != '' ]; then
	in-container journalctl -u doorman
	exit "${doorman_status}"
fi


sleep 30
in-container systemctl restart doorman || doorman_status=$?
if [ "${doorman_status}" != '' ]; then
	in-container journalctl -u doorman
	exit "${doorman_status}"
fi

echo 'Installing test apps...'
helm repo add doorman-integration-test-jetstack https://charts.jetstack.io
helm repo add doorman-integration-test-ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo add doorman-integration-test-bitnami https://charts.bitnami.com/bitnami
helm repo update
helm upgrade cert-manager doorman-integration-test-jetstack/cert-manager \
	--install \
	--version "${cert_manager_chart_version}" \
	--set installCRDs=true \
	--debug \
	--wait
kubectl apply -f "${test_dir}/cluster-issuer.yaml"
helm upgrade --install ingress-nginx doorman-integration-test-ingress-nginx/ingress-nginx \
	--install \
	--version "${nginx_ingress_controller_chart_version}" \
	--set controller.kind=DaemonSet \
	--set controller.hostPort.enabled=true \
	--set controller.hostPort.ports.http=8080 \
	--set controller.hostPort.ports.https=8443 \
	--set controller.service.type=ClusterIP \
	--debug \
	--wait
sleep 30 # ingress-nginx can become unhealth after it become healthy sometimes, best to let it settle
helm upgrade --install nginx doorman-integration-test-bitnami/nginx \
	--install \
	--version "${nginx_chart_version}" \
	--set ingress.enabled=true \
	--set ingress.hostname=doorman.integration.test \
	--set ingress.tls=true \
	--set ingress.annotations.'cert-manager\.io/cluster-issuer'=selfsigned-cluster-issuer \
	--set ingress.annotations.'kubernetes\.io/ingress\.class'=nginx \
	--set service.type=ClusterIP \
	--debug \
	--wait

echo 'Testing control plane load balancing...'
in-container curl -vk https://localhost:7443

echo 'Testing http ingress load balancing...'
in-container curl -v http://localhost:80 -H 'Host:doorman.integration.test'

echo 'Testing https ingress load balancing...'
in-container curl -vk https://localhost:8443 -H 'Host:doorman.integration.test'
