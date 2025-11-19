# Contributing to This Project

Thank you for your interest in contributing! Please follow these guidelines to help us review your changes quickly and effectively.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Style](#code-style)
- [Testing](#testing)
- [Commit Messages](#commit-messages)
- [Pull Requests](#pull-requests)
- [Reporting Issues](#reporting-issues)
- [Contact](#contact)

## Getting Started

1. **Fork the repository** and clone your fork.
2. **Install Go** (version specified in `go.mod`) and [Node.js](https://nodejs.org/) (if working with JS code).
3. Run `go mod tidy` to install Go dependencies.
4. Run `npm install` in relevant JS directories.

## Development Workflow

- Make your changes in a feature branch.
- Keep your branch up to date with the `main` branch.
- Run `make test` before submitting a pull request.

## Code Style

- **Go:** Use `gofmt` and `goimports` for formatting. Follow idiomatic Go practices.
- **JavaScript:** Use the project's `.eslintrc` or standard JS style.

## Naming Tools

- **Tools Naming Convention:** Name tools using the format `<service>-<action>`, e.g., `apps-list` or `spaces-key-create`. Use lowercase and hyphens to separate words.
- **Tools Argument Naming:** Name tool arguments using UpperCamelCase (e.g., `AppID`, `PerPage`, `Request`). This matches the convention used in Go structs and tool definitions.

## Testing

- **Go:** Run `go test ./...` to execute all tests.
- **JavaScript:** Run `npm test` in the relevant directory.
- Add or update tests for any new features or bug fixes.

## Commit Messages

- Use clear, descriptive commit messages.
- Reference issues or PRs when relevant (e.g., `Fixes #123`).

## Pull Requests

- Ensure your branch is rebased on the latest `main`.
- Provide a clear description of your changes.
- Link related issues.
- Ensure all tests pass and code is linted.

## Reporting Issues

- Search existing issues before opening a new one.
- Provide as much detail as possible: steps to reproduce, expected behavior, logs, etc.

## Contact

For questions, open an issue or contact the maintainers via GitHub.

---

Thank you for contributing!