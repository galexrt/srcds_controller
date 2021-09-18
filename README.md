# srcds_controller

## Components

### sc

Client tool to manage (start, stop, remove, update, etc. Gameserver containers).

### srcds_cmdrelay

Relay component to allow a Gameserver to send commands by HTTP GET / POST request to other servers if needed.

### srcds_controller

Controller component which can check on the Gameservers and if necessary restart them.

### srcds_runner

Component inside the container running the Gameserver process. Reading from stdin as the "normal" Gameserver process.
