#!/bin/bash

/bin/consul.run start
/bin/deploy_srv.run start

/usr/sbin/sshd -D

