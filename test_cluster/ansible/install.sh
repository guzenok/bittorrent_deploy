#!/bin/bash

cd `dirname $0`

# config ssh
grep "ansible_" inventory.txt | while read host
do
  ssh-keyscan $host >> ~/.ssh/known_hosts
done

# run playbook
ansible-playbook install.yml
