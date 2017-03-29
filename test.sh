#!/bin/bash

# Сколько контейнеров-хостов создать (3 цифры, ведущие нули)
CONTAINERS_COUNT=009


# Генерация конфигов и директорий
make clean
make CONTAINERS_COUNT=${CONTAINERS_COUNT} build_containers


# Генерация тестового файла для раздачи
FILE_NAME=`tempfile`
dd if=/dev/urandom of=${FILE_NAME} bs=1M count=160
mv ${FILE_NAME} ./test_cluster/storage/host_000/
FILE_NAME=`basename ${FILE_NAME}`


# Запуск контейнеров
cd test_containers
docker-compose -p host up -d
cd ..

echo TIME FILES_COUNT
echo `date '+%H:%M:%S'` 0
while true
do
    CNT=`find ./test_cluster/storage/ -name "${FILE_NAME}" | wc -l`
    echo `date '+%H:%M:%S'` ${CNT}
    if [ ${CNT} -ge ${CONTAINERS_COUNT} ]
    then
      break
    else
      sleep 1
    fi
done

cd test_containers
docker-compose -p host down
cd ..