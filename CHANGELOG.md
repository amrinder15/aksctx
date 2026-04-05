# Changelog

## 1.1.0 - 2026-04-04

### Added in 1.1.0

- Add `aksctx diff` for interactive AKS-to-AKS comparison using subscription-first selection for the left and right clusters.
- Compare AKS identity, platform, access, network, add-on, and node-pool fields in a single report.

## 1.0.0 - 2026-04-04

### Added

- Support local kubeconfig context switching with `aksctx switch` and `aksctx s` without calling Azure APIs.
- Support `-d` and `--discovery` on `switch`, including the short form `aksctx s -d`, to discover AKS clusters from Azure on demand.
- Build and publish Windows binaries through GoReleaser for both `amd64` and `arm64`.

### Changed

- Simplify the local switch picker to show only context names.
- Simplify namespace selection so only the current default namespace is annotated.

## 0.2.2 - 2026-04-03

### Added

- Prompt for a default Kubernetes namespace after switching to an AKS cluster.
- Support `aksctx -v` and `aksctx --version` to print the CLI version.

### Changed

- Inject the release version into binaries during GoReleaser builds.
