IMAGE_NAME := asteria798/krillinai
CONTAINER_NAME := krillin
DOCKER := /Applications/Docker.app/Contents/Resources/bin/docker

.PHONY: build run stop deploy logs

build:
	$(DOCKER) build -f Dockerfile.local -t $(IMAGE_NAME) .

stop:
	-$(DOCKER) stop $(CONTAINER_NAME)
	-$(DOCKER) rm $(CONTAINER_NAME)

run:
	$(DOCKER) run -d \
		--name $(CONTAINER_NAME) \
		-p 8888:8888 \
		-v $(PWD)/config/config.toml:/app/config/config.toml \
		-v $(PWD)/tasks:/app/tasks \
		-v $(PWD)/models:/app/models \
		-v $(PWD)/bin:/app/bin \
		-v $(PWD)/cookies.txt:/app/cookies.txt \
		-v $(PWD)/static:/app/static \
		$(IMAGE_NAME)

deploy: build stop run

logs:
	$(DOCKER) logs -f $(CONTAINER_NAME)
