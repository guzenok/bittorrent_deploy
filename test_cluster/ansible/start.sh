#!/bin/bash

cd `dirname $0`

TAG=$1

test -z $TAG && TAG=start

# run playbook
ansible-playbook -e "flag=$TAG" start.yml
