#!/bin/bash

# Сколько контейнеров-хостов создать (3 цифры, ведущие нули)
CONTAINERS_COUNT=099


# Генерация конфигов и директорий
make CONTAINERS_COUNT=${CONTAINERS_COUNT} build


# Генерация тестового файла для раздачи
FILE_NAME=`tempfile`
dd if=/dev/urandom of=${FILE_NAME} bs=1M count=160
mv ${FILE_NAME} ./test_cluster/storage/host_000/
FILE_NAME=`basename ${FILE_NAME}`


# Запуск контейнеров
make up

echo `date '+%H:%M:%S'` 0
while true
do
    CNT=`find ./test_cluster/storage/ -name "${FILE_NAME}" | wc -l`
    if [ ${CNT} -ge ${CONTAINERS_COUNT} ]
    then
      echo `date '+%H:%M:%S'` ${CNT}
      break
    fi
done

make clean