## Hardware controls

LEDs:

- Off/On/Locked

  - Off: warmoffice is `OFF`
  - Red: warmoffice is `ON`
  - Orange: warmoffice is `LOCKED`

Buttons:

- Off/On/Locked

  Three position switch that controls major operating mode:

  - Off: warmoffice is not controlling the heater
  - On: warmoffice is controlling the heater. This will initially transition
    to `IDLE` mode.
  - Locked: warmoffice will enter `LOCKED` mode. The heater will maintain the
    set temperature indefinitely.

## External control

<!-- https://gist.github.com/teknoraver/5ffacb8757330715bcbcc90e6d46ac74 -->

The server listen on a Unix socket (by default `/run/warmoffice.sock`) for HTTP
connections and responds to the following commands:

- `GET /state`

  Show current state

- `POST /state`

  Request body:

  ```
  {
    "state": "<state>"
  }
  ```

  Enter state `<state>`, which can be one of:

  - `OFF`
  - `IDLE`
  - `PREWARM`
  - `LOCKED`

## States

- `OFF`

  The heater is off. We are not watching for activity; nothing
  will activate the heater.

- `PREWARM`

  The heater will run for up to an hour (or until the set temperature is
  reached), after which it will enter the `IDLE` state.

- `IDLE`

  The heater is off, but we are watching for activity. If
  motion is detected, we will transition to `TRACKING`.

- `TRACKING`

  We detected motion. We must detect motion at least once every two minutes for
  10 minutes in order to confirm presence. If insufficient motion is detected,
  transition to `IDLE`; otherwise, after 10 minutes, transition to `ACTIVE`.

- `ACTIVE`

  The heater is on. We are watching for activity, and if there is no motion
  detected for 1.5 hours we will transition back to `IDLE` and turn off the
  heater.

- `LOCKED`

  The heater is on. We are no watching for active; the heater will remain
  on until we are explicitly switched into another state.
