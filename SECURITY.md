# Security Policy

## Purpose

Platform Lab CLI is a training and lab automation tool designed for the Platform Lab learning environment.

The `lab` command intentionally introduces controlled failures into local Docker and Docker Compose environments so students can practice troubleshooting and recovery.

## Supported Use

Platform Lab CLI is intended for:

- Local development environments
- Training environments
- Platform Lab course exercises
- Disposable Docker and Docker Compose environments

It is not intended for use against production systems.

## Important Safety Notice

Some Lab CLI commands deliberately:

- Stop services
- Change container configuration
- Inject resource constraints
- Modify networking or dependency configuration
- Generate temporary workload or resource pressure
- Recreate containers as part of controlled scenarios

Always use the CLI only with the supported Platform Lab environment.

Do not run Lab CLI scenarios against production infrastructure, shared environments, or systems containing important data.

## Reporting a Security Issue

If you discover a security vulnerability in Platform Lab CLI, please do not publish sensitive details in a public GitHub issue.

Instead, use GitHub's private security reporting or Security Advisory feature for the repository.

When reporting a vulnerability:

- Describe the issue clearly
- Include steps to reproduce it
- Include the affected version
- Do not include passwords, API keys, tokens, credentials, or other secrets
- Do not include private or sensitive user data

## Secrets and Sensitive Information

Platform Lab CLI should not intentionally collect, display, or store credentials or secrets.

Contributions that introduce logging or collection of sensitive values should include appropriate redaction or filtering.

## Supported Versions

Security fixes are provided for the latest released version of Platform Lab CLI.