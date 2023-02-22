NAME = nginx-hosts-reload
GOBUILD = go build

.PHONY: build
build:
	mkdir -p bin
	$(GOBUILD) -o bin/$(NAME)

build_amd64:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o bin/$(NAME)-amd64

.PHONY: test
test: vet unit integration

.PHONY: vet
vet:
	go vet

.PHONY: unit
unit:
	go test

.PHONY: integration
integration: image
	go clean -testcache && go test ./ci

.PHONY: image
image: build_amd64
	cd ci/docker
	docker build -t nginx-hosts-reload -f ci/docker/Dockerfile .

docker_start: docker_stop image
	docker run -d --name nginx-hosts-reload nginx-hosts-reload
	docker cp bin/nginx-hosts-reload-amd64 nginx-hosts-reload:/usr/local/bin/nginx-hosts-reload 
	docker exec nginx-hosts-reload "/bin/bash" "-c" 'nohup /usr/local/bin/nginx-hosts-reload > /tmp/nginx-hosts-reload.log 2>&1 &'

docker_stop:
	docker stop nginx-hosts-reload > /dev/null 2>&1 || true
	docker rm nginx-hosts-reload> /dev/null 2>&1 || true

docker_exec:
	docker exec -it nginx-hosts-reload /bin/bash

.PHONY: bosh_release
bosh_release:
	$(eval export VERSION ?= 0.0.$(shell date +"%s"))
	$(eval export REGION ?= ${AWS_DEFAULT_REGION})
	$(eval export BUCKET ?= gds-paas-build-releases)
	$(eval export TARBALL_DIR ?= bosh-release-tarballs)
	$(eval export TARBALL_NAME = paas-nginx-hosts-reload-${VERSION}.tgz)
	$(eval export TARBALL_PATH = ${TARBALL_DIR}/${TARBALL_NAME})

	@[ -d "${TARBALL_DIR}" ] || mkdir "${TARBALL_DIR}"
	@[ -d "release/src/github.com/alphagov/paas-nginx-hosts-reload" ] || mkdir -p "release/src/github.com/alphagov/paas-nginx-hosts-reload"

	@rm -rf release/src/github.com/alphagov/paas-nginx-hosts-reload/*

	# rsync doesn't exist in the container
	# which is used in CI for building the
	# bosh release. Creating and extracting
	# a tar archive is a simple enough replacement.
	git ls-files \
	| grep -v "release/" \
	| tar cf nginx-hosts-reload.tz -T -

	tar xf nginx-hosts-reload.tz -C release/src/github.com/alphagov/paas-nginx-hosts-reload
	rm nginx-hosts-reload.tz

	bosh create-release \
		--name "paas-nginx-hosts-reload" \
		--version "${VERSION}" \
		--tarball "${TARBALL_PATH}" \
		--dir release \
		--force

	ls -al ${TARBALL_DIR}

	@# Can't use heredoc in Make target
	@echo "releases:"
	@echo "  - name: paas-nginx-hosts-reload"
	@echo "    version: ${VERSION}"
	@echo "    url: https://s3-${REGION}.amazonaws.com/$${BUCKET}/paas-nginx-hosts-reload-${VERSION}.tgz"
	@echo "    sha1: $$(openssl sha1 "${TARBALL_PATH}" | cut -d' ' -f 2)"

.PHONY: gh_actions
gh_actions:
	act

.PHONY: clean
clean:
	rm -rf bin
	rm -rf bosh-release-tarballs
	rm -rf release/.dev_builds
	rm -rf release/dev_releases
