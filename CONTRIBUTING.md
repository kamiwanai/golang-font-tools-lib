# Contributing to fonttools

## Contributor License Agreement (CLA)

By submitting a pull request or any contribution to this project, you agree to the following:

1. **Grant of rights.** You grant the copyright holder (kamiwanai) a perpetual,
   worldwide, non-exclusive, royalty-free, irrevocable license to use, modify,
   sublicense, and relicense your contribution, in whole or in part, under any
   license terms — including but not limited to open-source, proprietary, and
   commercial licenses.

2. **Original work.** You represent that your contribution is your original work,
   that you have the right to grant the above license, and that no third party
   has any claim to your contribution that would conflict with these terms.

3. **No obligation.** You are not obligated to contribute. This agreement only
   applies when you do.

4. **Irrevocability.** Once submitted, this grant cannot be revoked.

This CLA exists so that the project can offer both AGPL-3.0 and commercial
licenses without requiring every contributor to sign a separate agreement.

If you do not agree to these terms, do not submit contributions.

---

## How to contribute

1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/my-feature`).
3. Write code with tests. Run `go test ./...` and `go vet ./...`.
4. Ensure `gofmt -l .` produces no output.
5. Open a pull request against `main`.

## Code style

- `gofmt` formatting (non-negotiable).
- Exported functions and types must have doc comments.
- Keep package dependencies minimal — `golang.org/x/image` is the only external dep.
- Tests for every new decoder or parser path.
- Fuzz tests for binary parsers where feasible.

## Reporting issues

Open a GitHub issue. Include:
- Font file (or a link to one) that triggers the problem.
- Expected vs actual output.
- Stack trace if there is a panic.
