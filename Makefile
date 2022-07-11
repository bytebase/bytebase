PACKAGE_NAME          := github.com/bytebase/bytebase
GOLANG_CROSS_VERSION  ?= v1.17.6

.PHONY: release-dry-run
release-dry-run:
	docker run \
	--rm \
	--privileged \
	-e CGO_ENABLED=1 \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v `pwd`:/go/src/$(PACKAGE_NAME) \
	-w /go/src/$(PACKAGE_NAME) \
	goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
	--rm-dist --skip-validate --skip-publish
