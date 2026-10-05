# Platform Lab CLI

Platform Lab CLI (`lab`) is a controlled incident-injection tool
for the Platform Lab training environment.

It creates deterministic and reversible failure scenarios for
practicing production troubleshooting.

## Design Principle

Lab creates the problem.

Doctor collects evidence.

The engineer investigates, reasons, fixes, and verifies.

## Core Commands

```text
lab verify
lab list
lab show <scenario>
lab start <scenario>
lab status
lab reset
lab challenge
```

`lab challenge` is the Course 1 final practical mode. It activates one
challenge-eligible incident without revealing the selected scenario. `lab
status` keeps the scenario hidden while challenge mode is active, and `lab
reset` restores the baseline without printing scenario-specific cleanup
information.

The challenge pool intentionally contains only scenarios that are considered
safe and stable for an unknown final-course incident. Resource-intensive,
persistence, temporary, or not-yet-validated scenarios are excluded.

## Safety

Lab is designed to:

- validate the Platform Lab environment before changing it
- make only predefined changes
- record every mutation
- restore changes through `lab reset`
- avoid destructive Docker cleanup operations
- never delete Platform Lab's baseline persistent volumes
- hide the selected root-cause scenario during challenge mode

## Development Status

Course 1 Lab CLI scenarios through Lesson 48 and the Lesson 50 challenge mode
are implemented. Lesson 49 is an image-security exercise and does not require a
Lab incident scenario.
