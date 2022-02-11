.PHONY: all fmt vet test integration-test clean install-systemd install-docker uninstall-systemd uninstall-docker

TIMESTAMP := $(shell date +%s)

all: fmt vet bin/coverage.html test bin/doorman bin/doorman-arm64 bin/doorman.exe integration-test

clean:
	rm -rf bin/

fmt: $(wildcard **/*.go *.go)
	go fmt ./...

vet: $(wildcard **/*.go *.go)
	go vet ./...

bin/coverage.out: $(wildcard **/*.go *.go)
	mkdir -p bin
	go test -coverprofile=bin/coverage.out

bin/coverage.html: bin/coverage.out
	go tool cover -html=bin/coverage.out -o bin/coverage.html

test: $(wildcard **/*.go *.go) bin/coverage.html

bin/doorman: $(wildcard **/*.go *.go)
	CGO_ENABLED=0 GOOS=linux go build -a -tags '-w -extldflags "-static"' -o bin/doorman main.go

bin/doorman-arm64: $(wildcard **/*.go *.go)
	CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -a -tags '-w -extldflags "-static"' -o bin/doorman-arm64 main.go

bin/doorman.exe: $(wildcard **/*.go *.go)
	CGO_ENABLED=0 GOOS=windows go build -a -tags '-w -extldflags "-static"' -o bin/doorman.exe main.go

integration-test: bin/doorman hack/integration-test/run.sh hack/integration-test/Dockerfile hack/integration-test/cluster-issuer.yaml hack/integration-test/doorman.yaml
	hack/integration-test/run.sh

install-systemd: bin/doorman
	cp bin/doorman /usr/local/bin/doorman
	if [ ! -e /etc/nginx/doorman.yaml ]; then \
		cp docs/examples/default.yaml /etc/nginx/doorman.yaml ; \
		chown www-data /etc/nginx/doorman.yaml ; \
	fi
	cp deployments/doorman.service /etc/systemd/system/doorman.service
	ln -sf /etc/systemd/system/doorman.service /etc/systemd/system/multi-user.target.wants/
	if ! grep '^%www-data ALL=/usr/bin/systemctl restart nginx$$' /etc/sudoers; then \
		echo '%www-data ALL=/usr/bin/systemctl restart nginx$$' >> /etc/sudoers; \
	fi
	systemctl daemon-reload
	systemctl start doorman
install-systemd-arm64: bin/doorman-arm64
	cp bin/doorman-arm64 /usr/local/bin/doorman
	if [ ! -e /etc/nginx/doorman.yaml ]; then \
		cp docs/examples/default.yaml /etc/nginx/doorman.yaml ; \
		chown www-data /etc/nginx/doorman.yaml ; \
	fi
	cp deployments/doorman.service /etc/systemd/system/doorman.service
	ln -sf /etc/systemd/system/doorman.service /etc/systemd/system/multi-user.target.wants/
	if ! grep '^%www-data ALL=/usr/bin/systemctl restart nginx$$' /etc/sudoers; then \
		echo '%www-data ALL=/usr/bin/systemctl restart nginx$$' >> /etc/sudoers; \
	fi
	systemctl daemon-reload
	systemctl start doorman
uninstall-systemd:
	systemctl stop doorman
	systemctl disable doorman
	rm /etc/systemd/system/doorman.service
install-docker: bin/doorman
	docker build --tag=doorman:local-$(TIMESTAMP) .
	if [ ! -e /etc/nginx/doorman.yaml ]; then \
		cp docs/examples/default.yaml /etc/nginx/doorman.yaml ; \
		chown www-data /etc/nginx/doorman.yaml ; \
	fi
	docker run \
		--detach \
		--name=doorman \
		--restart=always \
		--mount src=/etc/nginx/,dst=/etc/nginx/,type=bind \
		--mount src=/var/www/.kube/config,dst=/var/www/.kube/config,type=bind \
		--mount src=/var/run/docker.sock,dst=/var/run/docker.sock,type=bind \
		doorman:local-$(TIMESTAMP)
uninstall-docker:
	docker stop doorman
	docker rm doorman

todo:
	grep --exclude=bin/ -R TODO: .
