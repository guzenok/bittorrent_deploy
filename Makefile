build: test_cluster/ansible.key compile
	sudo chown -R $$USER ./test_cluster/storage/*
	docker build -t ansible_managed_host -f ./test_cluster/Dockerfile.managed ./test_cluster
	docker build -t ansible_control_host -f ./test_cluster/Dockerfile.control ./test_cluster

compile:
	cd test_cluster/bin && CGO_ENABLED=0 go build -a -ldflags '-s' -installsuffix cgo ../../deploy_service/ && cd ../..

lint:
	gometalinter --disable-all --enable=misspell --enable=errcheck --enable=goconst --enable=gosimple --enable=deadcode --enable=aligncheck --enable=unconvert --enable=gas --deadline=100s ./deploy_service

test_cluster/ansible.key:
	ssh-keygen -q -f test_cluster/ansible.key -t rsa -b4096 -C "ansible@*" -N ""

start: compile
	cd ./test_cluster && docker-compose -p host up -d && cd ..
	docker exec -ti host_ansible_control_00_1 bash -c "/root/ansible/install.sh"
	docker exec -ti host_ansible_control_00_1 bash -c "/root/ansible/start.sh"

restart: compile
	docker exec -ti host_ansible_control_00_1 bash -c "/root/ansible/start.sh restart"

stop:
	cd ./test_cluster && docker-compose -p host stop && cd ..

down:
	cd ./test_cluster && docker-compose -p host down && cd ..

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
