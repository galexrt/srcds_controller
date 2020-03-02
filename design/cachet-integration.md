# Cachet Integration

**Status**: Abandoned.

## Questions

* When should an incident message / update be created on Cachet?
  * **Thoughts**:
    * In case of a Gameserver having exited / "dieing" (Container `die` event)
    * Manual Server Stop and Start? By default on?
  * Answer(s):
    * When a container has died and is / will be restarted an incident message seems appropriate.
    * Having a flag for the `sc stop` and `sc restart` sub commands to post an incident message might be an option.
