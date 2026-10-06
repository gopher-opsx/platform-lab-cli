# Platform Lab CLI

Platform Lab CLI is a training-focused command-line tool used by the Platform Lab course to create controlled Docker and Docker Compose incidents for troubleshooting practice.

The CLI is designed for the local Platform Lab environment.

It intentionally introduces failures so students can investigate symptoms, collect evidence, identify root causes, recover the environment, and verify the result.

## Important Safety Notice

Platform Lab CLI is intended for local training environments only.

Do not run Lab CLI scenarios against:

- Production infrastructure
- Shared environments
- Systems containing important data
- Docker hosts that are not part of the supported Platform Lab setup

Some scenarios intentionally stop services, change configuration, apply resource constraints, or recreate containers.

## Requirements

You need:

- Docker
- Docker Compose
- Git
- The Platform Lab student repository

Go is required only if you choose to build the CLI from source.

## Build from Source

Clone the CLI repository:

```bash
git clone https://github.com/gopher-opsx/platform-lab-cli.git
cd platform-lab-cli
```

Download Go dependencies:

```bash
go mod tidy
```

Build the CLI.

Linux or macOS:

```bash
go build -o bin/lab ./cmd/lab
```

Windows:

```powershell
go build -o bin/lab.exe ./cmd/lab
```

## Run Platform Lab CLI

The CLI is designed to be executed from inside the Platform Lab student repository.

Example:

```bash
cd platform-lab
```

If the Lab CLI binary is available in your `PATH`:

```bash
lab verify
```

If you built the CLI separately and have not installed it into your `PATH`, use the appropriate relative or absolute path to the binary.

## Commands

### Verify the Environment

```bash
lab verify
```

Checks that:

- The Platform Lab repository is detected
- Docker is available
- Docker Compose is available
- Required Compose files exist
- Required Platform Lab services are defined

No changes are made.

### List Scenarios

```bash
lab list
```

Shows the available controlled incidents.

### Show a Scenario

```bash
lab show restart-loop
```

Displays information about a scenario before starting it.

### Start an Incident

```bash
lab start restart-loop
```

The CLI:

1. Validates the environment
2. Records the required recovery state
3. Injects the controlled incident
4. Verifies that the intended failure occurred

The student then investigates the platform manually.

### Check Active Incident Status

```bash
lab status
```

Shows whether a Lab scenario or troubleshooting challenge is active.

### Reset the Environment

```bash
lab reset
```

Restores the supported Platform Lab baseline and removes Lab-owned temporary changes.

### Start the Final Troubleshooting Challenge

```bash
lab challenge
```

Challenge mode activates a supported controlled incident without revealing the selected scenario.

The student must begin from the system and the customer symptom, collect evidence, form a hypothesis, identify the root cause, recover the platform, and verify the result.

## Course 1 Scenarios

Platform Lab CLI currently supports controlled incidents for:

- Catalog restart loops
- Running containers with dead applications
- Environment configuration failures
- DNS failures
- Port exposure failures
- Dependency readiness failures
- Lost persistence
- Disk growth
- CPU pressure
- Memory growth
- OOM kills
- Slow dependencies
- Resource limits
- PostgreSQL outages
- Redis outages
- Kafka outages
- Kafka consumer failures
- Cascading incidents

Lesson 49 uses image security scanning and does not require a Lab CLI incident.

Lesson 50 uses `lab challenge`.

## Investigation Philosophy

Platform Lab follows this troubleshooting method:

```text
Symptom
   ↓
Understand the system path
   ↓
Collect evidence
   ↓
Form a hypothesis
   ↓
Test
   ↓
Identify the root cause
   ↓
Recover
   ↓
Verify
```

Production troubleshooting begins with the system and the symptom, not with a random command.

## Project Principles

Platform Lab CLI scenarios should be:

- Deterministic
- Reversible
- Safe for the supported training environment
- Evidence-oriented
- Focused on realistic platform failures

The CLI is not intended to automatically diagnose or fix incidents for the student.

## Contributing

See:

```text
CONTRIBUTING.md
```

## Security

See:

```text
SECURITY.md
```

## Code of Conduct

See:

```text
CODE_OF_CONDUCT.md
```

## License

Platform Lab CLI is licensed under the Apache License 2.0.

See:

```text
LICENSE
```