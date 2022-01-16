## States

- `OFF`

  The heater is off. We are not watching for activity; nothing
  will activate the heater.

- `PREWARM`

  The heater will run for *N* minutes (or until the set temperature is
  reached), after which it will enter the `ACTIVE` state.

- `IDLE`

  The heater is off, but we are watching for activity. If
  motion is detected, we will transition to `TRACKING`.

- `TRACKING`

  We detected motion. We must detect motion at least once every two minutes for
  *N* minutes in order to confirm presence. If insufficient motion is detected,
  transition to `IDLE`; otherwise, after *N* minutes, transition to `ACTIVE`.

- `ACTIVE`

  The heater is on. We are watching for activity, and if there is no motion
  detected for *N* minutes (by default 1.5 hours) we will transition back to
  `IDLE` and turn off the heater.

- `LOCKED`

  The heater is on. We are not watching for active; the heater will remain
  on until we are explicitly switched into another state.
