# Either edit below variables to their desired value or provide these in the command line
# Recommended to edit the variables to get reproducible builds
project_name = worker
flavor = full # full,minimal
runtime = podman # podman, docker
editor = cli # cli, vscode
infra = pod # pod, compose (pod is only supported with podman)
container_name = ${project_name}-dev
optimized = true # If set to true, this will look for existing image localhost/project-name:dev and use it for the new image

devcontainer:
ifeq ($(strip $(editor)), cli)
ifeq ($(strip $(infra)), pod)
	$(MAKE) _setup_pod
else
	$(MAKE) _setup_compose
endif
else
	$(MAKE) _setup_vscode
endif

_prebuild:
	-mv .devcontainer/project-name .devcontainer/$(project_name)
	sed -i "s/project-name/$(project_name)/g" .devcontainer/$(project_name)/*
	sed -i "s/project-name/$(project_name)/g" .devcontainer/docker-compose.yaml
	sed -i "s/project-name/$(project_name)/g" .devcontainer/pod.yaml
	sed -i "s/project-name/$(project_name)/g" .devcontainer/devcontainer.json
	sed -i "s/container-name/$(container_name)/g" .devcontainer/devcontainer.json
	sed -i "s/container-name/$(container_name)/g" .devcontainer/pod.yaml
	sed -i "s/container-name/$(container_name)/g" .devcontainer/docker-compose.yaml
	cp .devcontainer/$(project_name)/Dockerfile.template .devcontainer/$(project_name)/Dockerfile

ifeq ($(strip $(flavor)), full)
	cp .devcontainer/$(project_name)/Dockerfile.full.base .devcontainer/$(project_name)/Dockerfile.base
	cp .devcontainer/$(project_name)/Makefile.full .devcontainer/$(project_name)/Makefile
else
	cp .devcontainer/$(project_name)/Dockerfile.minimal.base .devcontainer/$(project_name)/Dockerfile.base
	cp .devcontainer/$(project_name)/Makefile.minimal .devcontainer/$(project_name)/Makefile
endif

baseimage: _prebuild
	$(runtime) build -f .devcontainer/$(project_name)/Dockerfile.base -t localhost/$(project_name):base .devcontainer

_setup_vscode: baseimage
	devcontainer up --docker-path $(runtime) # Ensure that devcontainer cli is installed (sudo npm install -g @devcontainers/cli)

_setup_pod: baseimage
ifeq ($(strip $(runtime)), podman)
	sed -i "s/$(project_name)\///g" .devcontainer/$(project_name)/Dockerfile
	cd .devcontainer && podman kube play --build --replace pod.yaml && cd - # Ensure that the podman cli is installed
else
	echo "Invalid runtime and infra combination"
	exit 1
endif

_setup_compose: baseimage
ifeq ($(strip $(runtime)), podman)
	podman-compose -f .devcontainer/docker-compose.yaml up --build -d # Ensure that podman-compose is installed (python3 -m pip install podman-compose)
else
	docker compose --progress plain -f .devcontainer/docker-compose.yaml up --build -d # Ensure that the compose plugin is installed
endif

.PHONY: list
list:
	@LC_ALL=C $(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/(^|\n)# Files(\n|$$)/,/(^|\n)# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | grep -E -v -e '^[^[:alnum:]]' -e '^$@$$'

envstart:
ifeq ($(strip $(editor)), cli)
ifeq ($(strip $(infra)), pod)
	podman pod start $(project_name) && podman exec -it -w /workspace $(project_name)-$(container_name) bash
else
ifeq ($(strip $(runtime)), podman)
	podman-compose -f .devcontainer/docker-compose.yaml up -d && \
	podman-compose -f .devcontainer/docker-compose.yaml exec -w /workspace development bash
else
	docker compose -f .devcontainer/docker-compose.yaml up -d && \
	docker compose -f .devcontainer/docker-compose.yaml exec -w /workspace development bash
endif
endif
else
	devcontainer up --docker-path $(runtime) && devcontainer open
endif

enventer:
ifeq ($(strip $(editor)), cli)
ifeq ($(strip $(infra)), pod)
	podman exec -it -w /workspace $(project_name)-$(container_name) bash
else
ifeq ($(strip $(runtime)), podman)
	podman-compose -f .devcontainer/docker-compose.yaml exec -w /workspace development bash
else
	docker compose -f .devcontainer/docker-compose.yaml exec -w /workspace development bash
endif
endif
else
	# devcontainer exec --docker-path $(runtime) && devcontainer open
endif

envstop:
ifeq ($(strip $(editor)), cli)
ifeq ($(strip $(infra)), pod)
	podman pod stop $(project_name)
else
ifeq ($(strip $(runtime)), podman)
	podman-compose -f .devcontainer/docker-compose.yaml stop
else
	docker compose -f .devcontainer/docker-compose.yaml stop
endif
endif
else
	# devcontainer up --docker-path $(runtime) && devcontainer open
endif

envdestroy:
ifeq ($(strip $(editor)), cli)
ifeq ($(strip $(infra)), pod)
	cd .devcontainer && podman kube play --down pod.yaml && cd -
else
ifeq ($(strip $(runtime)), podman)
	podman-compose -f .devcontainer/docker-compose.yaml down
else
	docker compose -f .devcontainer/docker-compose.yaml down
endif
endif
else
	# devcontainer up --docker-path $(runtime) && devcontainer open
endif
