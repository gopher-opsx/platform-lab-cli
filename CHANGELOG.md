# Changelog

All notable changes to Platform Lab CLI will be documented in this file.

## v1.0.0

Initial public release for Platform Lab Course 1.

### Added

- Environment verification with `lab verify`
- Scenario discovery with `lab list`
- Scenario inspection with `lab show`
- Controlled incident injection with `lab start`
- Active incident status with `lab status`
- Safe recovery with `lab reset`
- Hidden troubleshooting challenge mode with `lab challenge`

### Course 1 Scenarios

- Lesson 31 — Catalog Restart Loop
- Lesson 32 — Container Running but Application Dead
- Lesson 33 — Catalog Environment Configuration Failure
- Lesson 34 — Catalog DNS Service Discovery Failure
- Lesson 35 — Web BFF Port Exposure Failure
- Lesson 36 — PostgreSQL Dependency Not Ready
- Lesson 37 — PostgreSQL Lost Persistence
- Lesson 38 — Disk Usage Keeps Growing
- Lesson 39 — Container CPU Goes High
- Lesson 40 — Container Memory Keeps Growing
- Lesson 41 — Container Gets OOM Killed
- Lesson 42 — Application Slow While CPU Looks Fine
- Lesson 43 — Resource Limits Under Load
- Lesson 44 — PostgreSQL Goes Down
- Lesson 45 — Redis Goes Down
- Lesson 46 — Kafka Goes Down
- Lesson 47 — Kafka Consumer Stops
- Lesson 48 — Cascading / Partial Major Incident

### Course Support

- Lesson 49 uses external image security scanning and does not require a Lab CLI incident.
- Lesson 50 uses `lab challenge` to activate a hidden scenario from the supported challenge pool.

### Safety

- Scenarios are intended for the local Platform Lab training environment.
- Recovery state is recorded before controlled incidents are injected.
- `lab reset` restores the supported baseline state.
- Challenge mode hides the selected scenario during investigation.