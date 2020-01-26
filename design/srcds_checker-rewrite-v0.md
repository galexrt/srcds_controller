# srcds_checker Rewrite v1

## Goals

* Config and Server State / Status independent
  * Config is reloaded every X minutes (or watched)
  * As long as a Server is found in the config(s), the state will be kept.
