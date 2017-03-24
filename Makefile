build: ansible.key.pub compile
	cp ansible.key* test_cluster/
	docker build -t ansible_managed_host -f ./test_cluster/Dockerfile.managed ./test_cluster
	docker build -t ansible_control_host -f ./test_cluster/Dockerfile.control ./test_cluster

compile:
	cd test_cluster/bin && CGO_ENABLED=0 go build -a -ldflags '-s' -installsuffix cgo ../../deploy_service/ && cd ../..

ansible.key.pub:
	ssh-keygen -q -f ansible.key -t rsa -b4096 -C "ansible@*" -N ""

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
	rm -f test_cluster/ansible.key* test_cluster/bin/* deploy_service/deploy_service