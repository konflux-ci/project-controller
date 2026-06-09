# Add Template Field — Reference

## Architecture

Two layers:

1. **Variables** — PDST `spec.variables`; values from PDS `spec.template.values`
   or variable `defaultValue` (later defaults may reference earlier vars)
2. **Template-able fields** — allowlist per kind in `supportedResourceTypes`

Reconciliation entry: `template.MkResources()` in `internal/template/resources.go`.
Name fields are processed first, validated, then general fields.

## Supported resource types

| Kind | API |
|------|-----|
| Application | `appstudio.redhat.com/v1alpha1` |
| Component | `appstudio.redhat.com/v1alpha1` |
| ImageRepository | `appstudio.redhat.com/v1alpha1` |
| IntegrationTestScenario | `appstudio.redhat.com/v1beta2` |
| ReleasePlan | `appstudio.redhat.com/v1alpha1` |

**Known gap:** ITS `spec.params` **names** are not templateable (TODO in
`resources.go`); only `value` fields are supported.

## Key files

| Purpose | Path |
|---------|------|
| Field allowlist | `internal/template/resources.go` |
| Path traversal / `[]` | `internal/template/unstructured.go` |
| Template execution | `internal/template/execute.go` |
| Integration tests | `internal/controller/projectdevelopmentstream_controller_test.go` |
| Fixtures | `config/samples/` |

Controller tests reconcile twice (owner ref, then resources) and compare to
`*_exp_results.yaml` via `checkExpectedFile()`.

## Example 1: Simple Component field (PR #880)

**Commit:** `f80ec02` — [PR #880](https://github.com/konflux-ci/project-controller/pull/880)

Add `spec.containerImage` so build-service gets resolved image paths, not
`{{.versionName}}` literals in `.tekton` PipelineRuns.

**Files:**

- `internal/template/resources.go` — `{"spec", "containerImage"}`
- `config/samples/projctl_v1beta1_projectdevelopmentstreamtemplate.yaml`
- `config/samples/projctl_v1beta1_pds_w_tmp_vars_exp_results.yaml`

No controller test table change — existing `"Application and Component resources"`
entry covers it.

## Example 2: Annotations + variables (PR #239)

**Commit:** `635a6db` — [PR #239](https://github.com/konflux-ci/project-controller/pull/239)

- `git-provider` / `git-provider-url` in Component `templateAbleFields`
- New PDST variables with defaults
- Updated `_exp_results.yaml`

## Example 3: Nested array — ITS params (PR #220)

**Commit:** `2de30a5`

- Path: `{"spec", "params", "[]", "value"}`
- Fixture triple: `pdst_w_intgtstscnario`, `pds_w_intgtstscnario`,
  `pds_w_intgtstscnario_exp_results`

## Example 4: String slice name field (RHTAPWATCH-1193)

**Commit:** `42b2ace`

- Path: `{"spec", "build-nudges-ref", "[]"}` in **name** fields
- Required prior work on `unstructured.go` + unit tests (first slice-of-strings
  name field)

## Example 5: ReleasePlan tenant pipeline

**Commit:** `dd88e4f`

```go
{"spec", "tenantPipeline", "params", "[]", "value"},
{"spec", "tenantPipeline", "pipelineRef", "params", "[]", "value"},
```

Fixtures: `pdst_w_relpln` / `pds_w_relpln_exp_results`.

## Example 6: Name vs general — ImageRepository

**Commit:** `a04da5f4` — `spec.image.name` moved to `templateAbleFields` (contains `/`).

## PR message pattern

```
fix: add spec.<field> to <Kind> templateAbleFields (<TICKET>)

<Why templating is needed — what breaks without it>
```

## git log helpers

```bash
git log --oneline -- internal/template/resources.go
git log --oneline --grep="templateAble"
git show <commit> --stat
```

## Local E2E verification

Optional level-3 testing — see [local-dev-setup](../local-dev-setup/SKILL.md).

## Grep helpers

```bash
rg 'templateAbleFields|templateAbleNameFields' internal/template/resources.go
rg 'Entry\(' internal/controller/projectdevelopmentstream_controller_test.go
rg '\{\{\.' config/samples/
```
