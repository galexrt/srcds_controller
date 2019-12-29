# srcds_controller

[![CircleCI branch](https://img.shields.io/circleci/project/github/RedSparr0w/node-csgo-parser/master.svg)]() [![Docker Repository on Quay](https://quay.io/repository/galexrt/srcds_controller/status "Docker Repository on Quay")](https://quay.io/repository/galexrt/srcds_controller) [![Go Report Card](https://goreportcard.com/badge/github.com/galexrt/srcds_controller)](https://goreportcard.com/report/github.com/galexrt/srcds_controller) [![license](https://img.shields.io/github/license/mashape/apistatus.svg)]()

## Components

### sc

Client tool to manage (start, stop, remove, update, etc. Gameserver containers).

### srcds_cmdrelay

Relay component to allow a Gameserver to send commands by HTTP GET / POST request to other servers if needed.

### srcds_controller

Controller component which can check on the Gameservers and if necessary restart them.

### srcds_runner

Component inside the container running the Gameserver process. Reading from stdin as the "normal" Gameserver process.
