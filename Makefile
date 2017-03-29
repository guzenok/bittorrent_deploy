CONTAINERS_COUNT=004
CONSUL_VERSION=0.7.5

build: configure test_cluster/ansible.key
	docker build -t ansible_managed_host -f ./test_cluster/Dockerfile.managed ./test_cluster
	docker build -t ansible_control_host -f ./test_cluster/Dockerfile.control ./test_cluster


build_containers: build
	cd test_containers/bin && CGO_ENABLED=0 go build -a -ldflags '-s' -installsuffix cgo ../../deploy_service/ && cd ../..
	cd test_containers/bin && wget https://releases.hashicorp.com/consul/${CONSUL_VERSION}/consul_${CONSUL_VERSION}_linux_amd64.zip -O consul.zip && unzip consul.zip && cd ../..
	cp test_cluster/ansible/ansible.cfg test_cluster/ansible/inventory.txt test_containers/ansible/
	sed -e 's/ansible_/deploy_/g' -e 's/\.\/storage/\.\.\/test_cluster\/storage/g' -e '/\.\/bin/d' test_cluster/docker-compose.yml > test_containers/docker-compose.yml
	docker build -t deploy_managed_host -f ./test_containers/Dockerfile.managed ./test_containers
	docker build -t deploy_control_host -f ./test_containers/Dockerfile.control ./test_containers


compile: lint
	cd test_cluster/bin && CGO_ENABLED=0 go build -a -ldflags '-s' -installsuffix cgo ../../deploy_service/ && cd ../..


lint:
	gometalinter --disable-all --enable=misspell --enable=errcheck --enable=goconst --enable=gosimple --enable=deadcode --enable=aligncheck --enable=unconvert --enable=gas --deadline=100s ./deploy_service


test_cluster/ansible.key:
	ssh-keygen -q -f test_cluster/ansible.key -t rsa -b4096 -C "ansible@*" -N ""


configure:
	cp test_cluster/ansible/inventory.txt.header		   test_cluster/ansible/inventory.txt;\
	cp test_cluster/docker-compose.yml.header		   test_cluster/docker-compose.yml;\
	find test_cluster/storage/ -mindepth 1 -type d -exec rm -fr "{}" \; 2>/dev/null ;\
	mkdir -p test_cluster/storage/host_000;\
	for i in `seq -w 001 ${CONTAINERS_COUNT}` ;\
	do \
	  echo ansible_managed_$$i >> test_cluster/ansible/inventory.txt;\
	  echo "    ansible_managed_$$i:"			>> test_cluster/docker-compose.yml;\
	  echo "        image: ansible_managed_host"		>> test_cluster/docker-compose.yml;\
	  echo "        volumes:"				>> test_cluster/docker-compose.yml;\
	  echo "            - ./bin:/usr/local/bin:ro"		>> test_cluster/docker-compose.yml;\
	  echo "            - ./storage/host_$$i:/var/deploy"	>> test_cluster/docker-compose.yml;\
	  echo "        networks:"				>> test_cluster/docker-compose.yml;\
	  echo "            - code-network"			>> test_cluster/docker-compose.yml;\
	  echo ""						>> test_cluster/docker-compose.yml;\
	  mkdir -p test_cluster/storage/host_$$i;\
	done


up: compile
	cd ./test_cluster && docker-compose -p host up -d && cd ..
	docker exec -ti host_ansible_control_000_1 bash -c "/root/ansible/install.sh"
	docker exec -ti host_ansible_control_000_1 bash -c "/root/ansible/start.sh"

down:
	cd ./test_cluster && docker-compose -p host down && cd ..


start: compile
	cd ./test_cluster && docker-compose -p host up -d && cd ..
	docker exec -ti host_ansible_control_000_1 bash -c "/root/ansible/start.sh"

stop:
	cd ./test_cluster && docker-compose -p host stop && cd ..


restart: compile
	docker exec -ti host_ansible_control_000_1 bash -c "/root/ansible/start.sh restart"


clean: down
	find . -type f -name ".gitignore" | while read CONF ;\
	do \
	  cat $$CONF | while read FULLMASK ;\
	  do \
	    SUBDIR=`dirname "$$FULLMASK"` ;\
	    MASK=`basename "$$FULLMASK"` ;\
	    find `dirname $$CONF`/$$SUBDIR -type f -name "$$MASK" ! -name ".gitignore" -mindepth 1 -exec rm -f {} \; ;\
	  done ;\
	done
