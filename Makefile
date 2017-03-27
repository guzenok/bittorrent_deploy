CONTAINERS_COUNT=004

build: configure test_cluster/ansible.key
	sudo chown -R $$USER ./test_cluster/storage/*
	docker build -t ansible_managed_host -f ./test_cluster/Dockerfile.managed ./test_cluster
	docker build -t ansible_control_host -f ./test_cluster/Dockerfile.control ./test_cluster


compile: lint
	cd test_cluster/bin && CGO_ENABLED=0 go build -a -ldflags '-s' -installsuffix cgo ../../deploy_service/ && cd ../..


lint:
	gometalinter --disable-all --enable=misspell --enable=errcheck --enable=goconst --enable=gosimple --enable=deadcode --enable=aligncheck --enable=unconvert --enable=gas --deadline=100s ./deploy_service


test_cluster/ansible.key:
	ssh-keygen -q -f test_cluster/ansible.key -t rsa -b4096 -C "ansible@*" -N ""


configure:
	cp test_cluster/ansible/inventory.txt.header		   test_cluster/ansible/inventory.txt;\
	cp test_cluster/docker-compose.yml.header		   test_cluster/docker-compose.yml;\
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
