# Contributing to MicroShift 2.0

Thank you for your interest in contributing to MicroShift 2.0. This guide covers everything you need to get started.

## Prerequisites

- Go 1.21 or later
- `golangci-lint` (for linting)
- A Linux system (Fedora recommended for full integration testing)
- `git`

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork:

```bash
git clone https://github.com/<your-username>/microshift-2.0.git
cd microshift-2.0
```

3. Create a branch for your work:

```bash
git checkout -b my-feature
```

4. Make your changes, then build and test:

```bash
make build
make test
make lint
```

5. Commit and push your branch, then open a pull request.

## Building and Testing

```bash
make build      # Build the microshift binary to bin/microshift
make test       # Run unit tests
make vet        # Run go vet
make lint       # Run golangci-lint
make fmt        # Format code with gofmt
make e2e        # Run end-to-end smoke tests (requires running system)
```

All tests must pass before a pull request can be merged. CI runs `build`, `test`, `vet`, and `lint` automatically on every pull request.

## Code Style

- Follow standard Go conventions. Run `make fmt` before committing.
- Keep functions focused and small.
- Prefer clear names over comments. Only add comments when the "why" is non-obvious.
- Use `error` returns rather than panics.
- Handle errors explicitly -- don't ignore them with `_`.

## Project Layout

```
cmd/microshift/        CLI entry point
pkg/config/            Configuration loading, defaults, validation
pkg/certs/             TLS certificate generation
pkg/kubeconfig/        Kubeconfig file generation
pkg/services/          Systemd service management, component config generation
pkg/bootstrap/         Cluster bootstrap (manifest application)
pkg/healthcheck/       Component health monitoring
pkg/daemon/            Main daemon orchestration loop
pkg/version/           Version information (set via ldflags)
assets/                Embedded Kubernetes manifests
packaging/             RPM spec, systemd units, default config
docs/                  Documentation
```

See the [Development Guide](docs/development.md) for a detailed walkthrough of how the daemon works.

## Writing Tests

- Place tests in `_test.go` files in the same package as the code being tested.
- Use `t.TempDir()` for temporary directories -- cleanup is automatic.
- Use table-driven tests for functions with multiple input/output cases.
- Use `t.Helper()` in test helper functions so failures report the caller's line.
- Tests should not require running infrastructure (systemd, API server). Mock external dependencies or test the logic in isolation.

Example:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"basic", "foo", "bar"},
        {"empty", "", ""},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := MyFunction(tt.input)
            if got != tt.expected {
                t.Errorf("MyFunction(%q) = %q, want %q", tt.input, got, tt.expected)
            }
        })
    }
}
```

## Commit Messages

- Use imperative mood: "Add feature" not "Added feature"
- Keep the first line under 72 characters
- Add a blank line then a longer description if needed
- Reference issue numbers when applicable: "Fixes #42"

## Pull Request Process

1. Describe what your PR does and why.
2. Reference any related issues.
3. Ensure all CI checks pass (build, test, vet, lint).
4. Keep PRs focused -- one logical change per PR.
5. Be responsive to review feedback.

## Reporting Issues

Open an issue on GitHub with:

- A clear title describing the problem
- Steps to reproduce
- Expected vs actual behavior
- System information (Fedora version, MicroShift version, `kubectl version`)
- Relevant logs (`journalctl -u microshift.service`)

## Areas Where Help is Wanted

- Testing on different hardware (ARM64, resource-constrained devices)
- Documentation improvements
- Storage driver testing and feedback
- OVN-Kubernetes CNI integration
- Bootc container image builds
- CentOS Stream 10 support

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
