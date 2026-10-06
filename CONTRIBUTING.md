# Contributing to Platform Lab CLI

Thank you for your interest in contributing to Platform Lab CLI.

Platform Lab CLI is a training-focused tool used to create controlled incidents for troubleshooting exercises. Contributions should preserve that goal: deterministic, safe, understandable, and reversible failure scenarios.

## Development Requirements

You will need:

- Go
- Docker
- Docker Compose
- Git
- A local copy of the Platform Lab repository

## Clone the Repository

```bash
git clone https://github.com/gopher-opsx/platform-lab-cli.git
cd platform-lab-cli
```

## Install Dependencies

```bash
go mod tidy
```

## Build

```bash
go build -o bin/lab ./cmd/lab
```

On Windows:

```powershell
go build -o bin/lab.exe ./cmd/lab
```

## Test

Run:

```bash
go test ./...
```

## Format

Before submitting changes:

```bash
gofmt -w .
```

## Adding a New Scenario

A new scenario should:

1. Represent a clear troubleshooting lesson.
2. Produce a deterministic failure.
3. Modify only the intended part of the Platform Lab environment.
4. Record enough state to allow safe recovery.
5. Verify that the intended incident actually occurred.
6. Support `lab reset`.
7. Restore the original Platform Lab state after reset.
8. Avoid exposing the root cause when the lesson requires investigation.
9. Avoid collecting or exposing secrets.

## Scenario Design Principles

Prefer scenarios that:

- Reproduce realistic platform failures
- Have one understandable underlying cause
- Produce observable evidence
- Can be safely repeated
- Can be fully reversed

Avoid scenarios that:

- Damage the host system
- Delete important user data
- Require cloud resources
- Depend on unpredictable external systems
- Modify production environments

## Pull Requests

Before opening a pull request:

```bash
gofmt -w .
go test ./...
go build -o bin/lab ./cmd/lab
```

Please keep pull requests focused and explain:

- What problem the change solves
- What scenario or behavior changes
- How the change was tested
- How the environment is restored

## Safety

Platform Lab CLI is intended for local training environments only.

Do not use controlled failure scenarios against production infrastructure or systems containing important data.

## License

By contributing to Platform Lab CLI, you agree that your contributions will be licensed under the Apache License 2.0.