## 0.1.34 / 2021-03-14

* [ENHANCEMENT] fixup steamcmd update logic

## 0.1.33 / 2021-03-14

* [ENHANCEMENT] allow core file size up to 2000
* [BUGFIX] use `docker run` for `steamcmd.sh` update process to circumvent the permissions issues if "not same uid" gameservers were updated

## 0.1.32 / 2021-02-11

* [ENHANCEMENT] cleanup go mod files
* [ENHANCEMENT] update base image to `galexrt/gameserver:v20210211-204709-521`

## 0.1.31 / 2021-01-22

* [ENHANCEMENT] enable cgo in builds

## 0.1.30 / 2021-01-22

* [BUGFIX] enable cgo in build

## 0.1.29 / 2021-01-18

* [ENHANCEMENT] add SYS_PTRACE capability to containers for gdb

## 0.1.28 / 2021-01-14

* [ENHANCEMENT] update base image to use galexrt/gameserver:v20210113-183742-977

## 0.1.27 / 2021-01-12

* [BUGFIX] fix dockerfile binary paths

## 0.1.25 / 2020-10-23

* [BUGFIX] added missing changelog for v0.1.24

## 0.1.24 / 2020-10-23

* [ENHANCEMENT] allow additional env vars to be set for gameservers
* [ENHANCEMENT] allow /etc/localtime to be set differently than /etc/timezone file

## 0.1.23 / 2020-10-23

* [BUGFIX] use cgo for builds

## 0.1.22 / 2020-10-23

* [BUGFIX] fix mono installation in docker till we have per gameserver type images

## 0.1.21 / 2020-10-23

* [BUGFIX] fix nil pointer when certain config options are not given (e.g., non srcds games)

## 0.1.20 / 2020-10-22

* [BUGFIX] fix docker base image (updated it to buster as well)

## 0.1.19 / 2020-10-22

* [ENHANCEMENT] fixed the command name in the "no servers given" error message

## 0.1.18 / 2020-10-06

* [BUGFIX] fix promu build config

## 0.1.17 / 2020-10-06

* [BUGFIX] fix type mismatch in srcds_runner

## 0.1.16 / 2020-10-06

* [BUGFIX] reworked config reconcile logic in srcds_runner

## 0.1.15 / 2020-09-13

* [ENHANCEMENT] remove hardcoded `srcds_run` flags

## 0.1.14 / 2020-08-16

* [BUGFIX] fix changelog

## 0.1.13 / 2020-08-16

* [BUGFIX] fix config reconcile bug

## 0.1.12 / 2020-07-27

* [BUGFIX] terminate srcds_runner when process has exited

## 0.1.11 / 2020-06-07

* [BUGFIX] ci: update makefile

## 0.1.10 / 2020-06-07

* [BUGFIX] ci: update makefile

## 0.1.9 / 2020-06-07

* [BUGFIX] *: fix `log.Error*` calls

## 0.1.8 / 2020-06-07

* [ENHANCEMENT] use GitHub actions + promu

## 0.0.1 / 2020-06-07

* [ENHANCEMENT] initial files
