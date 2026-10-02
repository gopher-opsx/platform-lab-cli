# Platform Lab CLI

Platform Lab CLI (`lab`) is a controlled incident-injection tool
for the Platform Lab training environment.

It creates deterministic and reversible failure scenarios for
practicing production troubleshooting.

## Design Principle

Lab creates the problem.

Doctor collects evidence.

The engineer investigates, reasons, fixes, and verifies.

## Safety

Lab is designed to:

- validate the Platform Lab environment before changing it
- make only predefined changes
- record every mutation
- restore changes through `lab reset`
- avoid destructive Docker cleanup operations
- never delete Platform Lab's baseline persistent volumes

## Status

Under development.