# Agent Guidelines for project-controller

## Project Overview

Kubernetes controller (kubebuilder v4) for Konflux CI. Three CRDs in `projctl.konflux.dev/v1beta1`:
- **Project** — groups development streams
- **ProjectDevelopmentStreamTemplate** — Go `text/template`-based resource generator
- **ProjectDevelopmentStream (PDS)** — the only reconciled CRD; creates resources via server-side apply

Only PDS has a reconciler. Changes to Project or Template trigger PDS re-reconciliation.

## Build & Test

```bash
make build          # binary → bin/manager
make test           # unit tests (envtest), also runs fmt/vet/generate
make lint           # golangci-lint
make lint-fix       # golangci-lint with --fix
make manifests      # regenerate CRDs + RBAC after changing kubebuilder markers
make generate       # regenerate deepcopy after changing api/v1beta1/ types
```

## Key Files

| Purpose | Path |
|---------|------|
| PDS reconciler | `internal/controller/projectdevelopmentstream_controller.go` |
| Resource type registry | `internal/template/resources.go` — `supportedResourceTypes` table |
| Template execution | `internal/template/execute.go` |
| Owner ref handling | `internal/ownership/` |
| CRD type definitions | `api/v1beta1/` |
| Test fixtures | `config/samples/` (inputs + `*_exp_results.yaml` expected outputs) |

## Adding a New Template-Supported Resource Type

1. Add entry to `supportedResourceTypes` in `internal/template/resources.go`
2. Add `//+kubebuilder:rbac` markers in the same file
3. Import the API in `internal/controller/suite_test.go` for CRD loading in tests
4. Run `make manifests` to regenerate RBAC and CRDs
5. Add test fixtures in `config/samples/`

## Reconciliation Flow

`ProjectDevelopmentStreamReconciler.Reconcile` in the controller file:
1. Sets Project as owner of PDS (re-queues after update)
2. If no template ref, marks Ready=True/NoTemplate and returns
3. Fetches referenced template, calls `template.MkResources()`
4. Server-side applies each resource (field owner: `projctl.konflux.dev`)
5. Sets Ready condition on PDS status

## Skills

Detailed guides live in `skills/` — each subdirectory contains a `SKILL.md` with instructions.

| Skill | Use when |
|-------|----------|
| [add-template-field](skills/add-template-field/SKILL.md) | Parametrizing CR fields in PDST templates, editing `templateAbleFields` / `templateAbleNameFields`, or template substitution test fixtures |
| [local-dev-setup](skills/local-dev-setup/SKILL.md) | Running project-controller on local Konflux (Kind), E2E template verification, `make run` / `make deploy` against `kind-konflux` |

## Rules

- **Template-able field PRs:** Apply **add-template-field** (`skills/add-template-field/SKILL.md`) and follow its workflow — do not summarize the allowlist process from memory.
- Always run `make test` before submitting — it includes fmt, vet, generate, and manifests
- RBAC markers live in two places: `internal/controller/` and `internal/template/resources.go`
- Run `make manifests` after any RBAC marker change
- Tests use Ginkgo/Gomega with envtest, impersonating `system:serviceaccount:system:controller-manager`
- The custom `hyphenize` template function is available in template specs
