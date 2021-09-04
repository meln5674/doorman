ARG K8S_VERSION=v1.22.1

FROM kindest/node:${K8S_VERSION}

RUN ln -s /etc/systemd/system/doorman.service /etc/systemd/system/multi-user.target.wants/doorman.service
RUN apt-get update --allow-insecure-repositories && apt-get install -y nginx --allow-unauthenticated

COPY hack/integration-test/doorman.yaml /etc/nginx/doorman.yaml
COPY bin/doorman /usr/local/bin/doorman
COPY deployments/doorman.service /etc/systemd/system/doorman.service
