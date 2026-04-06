# Contributing to Gofasta

Thank you for your interest in contributing to Gofasta! This document explains how to get started.

## Ways to Contribute

- **Report bugs** — Open an issue describing what happened, what you expected, and steps to reproduce.
- **Suggest features** — Open an issue describing the use case and proposed solution.
- **Submit code** — Fork the repo, make changes, and open a pull request.
- **Improve docs** — Fix typos, clarify explanations, or add examples.

## Development Setup

1. **Fork and clone** the repository:

   ```bash
   git clone https://github.com/YOUR_USERNAME/gofasta.git
   cd gofasta
   ```

2. **Verify Go version** (1.25 or later required):

   ```bash
   go version
   ```

3. **Run tests** to make sure everything works:

   ```bash
   go test ./...
   ```

4. **Build** to verify compilation:

   ```bash
   go build ./...
   ```

## Making Changes

1. **Create a branch** from `main`:

   ```bash
   git checkout -b your-feature-name
   ```

2. **Make your changes.** Follow the conventions you see in existing code:
   - Each package under `pkg/` is self-contained.
   - Exported functions and types have doc comments.
   - No package imports another package's internal details.

3. **Write tests** for new functionality. Place test files next to the code they test (`foo_test.go` alongside `foo.go`).

4. **Run tests and linting** before committing:

   ```bash
   go test ./...
   go vet ./...
   ```

5. **Commit** with a clear message describing *what* and *why*:

   ```
   pkg/cache: add TTL support to memory cache

   The in-memory cache now respects expiration times passed via Set().
   Previously all entries lived forever until explicitly deleted.
   ```

6. **Push and open a pull request** against `main`.

## Pull Request Guidelines

- Keep PRs focused — one logical change per PR.
- Include tests for new behavior.
- Update doc comments if you change a public API.
- Ensure all tests pass before requesting review.
- If your PR addresses an issue, reference it: `Fixes #123`.

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`).
- Use meaningful names. Avoid abbreviations unless they're widely understood (`cfg`, `ctx`, `db`, `err`).
- Keep functions short and focused on one task.
- Prefer returning errors over panicking.

## Reporting Security Issues

If you discover a security vulnerability, **do not open a public issue.** See [SECURITY.md](SECURITY.md) for instructions.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
