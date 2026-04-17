# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial project scaffold.
- Core CLI structure: `triage {pod, deployment, namespace, cluster, report, rules, config, version, completion}`.
- Rule engine with built-in first-party rule set.
- Output renderers: `text`, `json`, `markdown`.
- Configuration via `~/.config/triage/config.yaml` and `TRIAGE_*` env vars.
- kubectl plugin support (`kubectl-triage` symlink).
