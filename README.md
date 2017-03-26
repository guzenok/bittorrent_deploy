# bittorrent_deploy

## Запуск:

  ```make && make up```

## Результат:

  Запущено CONTAINERS_COUNT докер-контейнеров, каждый символизирует узел кластера. На каждом запущен consul в роли SD и deploy_service. Рабочии директории каждого deploy_service в test_cluster/storage/host_XXX/.
  Разместив файл в одной, через время получим его во всех.
  Только в первом контейнере установлен ansible для управления остальными, и проброшен порт с UI на http://localhost:8500/.

## Цели make подробнее:

* ### build:
  1.  генерирует ключи для доступа в контейнеры по ssh;
  2.  собирает докер-образы хостов;
  3.  формирует конфиги и директории для docker-composer'а на CONTAINERS_COUNT хостов;
  4.  компилирует deploy_service;

* ### up:

  запуск докер-контейнеров, установка на них consul и с помощью ansible и старт;

* ### down:

  остановка и удаление докер-контейнеров;

* ### start:

  запуск ранее остановленных контейнеров;

* ### stop:

  остановка докер-контейнеров;

* ### restart:

  перекомпиляция deploy_service и перезапуск его в запущенных контейнерах;
