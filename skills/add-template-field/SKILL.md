---
name: add-template-field
description: >-
  Use when konflux-ci/project-controller PRs add template-able fields in
  internal/template/resources.go, when users request parametrization of CR
  fields in ProjectDevelopmentStreamTemplate, when {{.var}} literals appear in
  generated Application/Component/ImageRepository/IntegrationTestScenario/
  ReleasePlan resources, or when extending config/samples fixtures for template
  substitution tests.
---

# Add Template Field

Users parametrize Konflux resources via **ProjectDevelopmentStreamTemplate**
(PDST) variables and Go `text/template` syntax (`{{.varName}}`). The controller
only substitutes templates in fields listed in `supportedResourceTypes`
(`internal/template/resources.go`). Unlisted fields are ignored — `{{...}}`
literals leak into created CRs.

**Not this skill:** adding a new supported resource type (RBAC markers,
`suite_test.go`, `make manifests` — see `AGENTS.md`).

## Triage

| Request | Action |
|---------|--------|
| Parametrize existing field on supported kind | → **Workflow** below |
| New resource kind in templates | → `AGENTS.md` resource-type section |
| Template variable missing / default errors | Check PDST variables + PDS values, not allowlist |
| Literal `{{.foo}}` in created CR | Field likely missing from allowlist |

## Field category

| Field value is… | Add to… | Validated after substitution |
|-----------------|---------|------------------------------|
| K8s name (metadata.name, spec.application, DNS-subdomain labels) | `templateAbleNameFields` | `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` |
| Free-form string (URLs, image refs, descriptions, paths with `/`) | `templateAbleFields` | None |
| Must never be controller-managed | `untouchableFields` | Stripped before apply |
| Set only on create | `createOnlyFields` | Stripped on update |

**Pitfall:** `spec.image.name` on ImageRepository is `templateAbleFields`, not name
fields — it may contain `/`.

## Path syntax

| `"[]"` position | Meaning | Example |
|-----------------|---------|---------|
| At end | String slice; template each element | `{"spec", "build-nudges-ref", "[]"}` |
| In middle | Array of objects; recurse | `{"spec", "params", "[]", "value"}` |

New path shapes may need `internal/template/unstructured.go` changes — see
[reference.md](reference.md).

## Workflow

Copy and track:

```
- [ ] 1. Confirm field path and resource kind
- [ ] 2. Pick category (name vs general vs untouchable vs createOnly)
- [ ] 3. Add path to supportedResourceTypes in resources.go
- [ ] 4. Update config/samples fixtures
- [ ] 5. Run make test (required)
- [ ] 6. (Optional) E2E on local Konflux — see local-dev-setup skill
```

### Registry change

```go
templateAbleFields: [][]string{
    {"spec", "containerImage"},
},
```

Field-only additions need **no** CRD, `make generate`, or `make manifests`
(unless RBAC markers change).

### Tests

Three levels — do level 1 for every PR; level 2 when fixtures need a new
scenario; level 3 optional before merge if behavior is hard to cover in envtest.

**Level 1 — required (in-repo CI):**

```bash
make test    # fmt, vet, generate, manifests, envtest controller tests
make lint    # recommended
```

Extend fixtures so controller tests assert the new field:

| Scope | Files |
|-------|-------|
| Minimal (most Component/Application fields) | `projctl_v1beta1_projectdevelopmentstreamtemplate.yaml` + `projctl_v1beta1_pds_w_tmp_vars_exp_results.yaml` |
| Dedicated scenario (other kinds / distinct behavior) | New PDST/PDS/`_exp_results.yaml` triple + `Entry(...)` in `projectdevelopmentstream_controller_test.go` |

**Level 2 — unit tests (only when needed):** new `[]` path semantics or template
edge cases — see [reference.md](reference.md).

**Level 3 — optional E2E (local Konflux):** when envtest is insufficient or you
need to confirm against real Application/Component CRDs. Apply
**local-dev-setup** ([skills/local-dev-setup/SKILL.md](../local-dev-setup/SKILL.md)):
install Konflux per [Konflux Operator docs](https://konflux-ci.dev/konflux-ci/docs/installation/install-local/),
deploy your project-controller build, apply samples to `default-tenant`, verify
the new field is substituted (not literal `{{.var}}`). Sample apply and checks:
[local-dev-setup/reference.md](../local-dev-setup/reference.md).

## Common mistakes

| Mistake | Symptom | Fix |
|---------|---------|-----|
| Field not in allowlist | Literal `{{.foo}}` in CR | Add path to `resources.go` |
| Name field with `/` | `TemplateGenerationFailed` | Use `templateAbleFields` or `hyphenize` |
| Wrong `[]` placement | Field silently not templated | Match unstructured.go traversal |
| PDST updated, not `_exp_results.yaml` | Controller test fails | Update expected fixture |

## Rationalizations (do not accept)

| Excuse | Reality |
|--------|---------|
| "Template syntax in PDST is enough" | Allowlist in Go is required |
| "Unit test passes, fixtures optional" | Controller reconcile tests are the integration gate |
| "Name field because it's metadata" | Annotations and image paths are usually general fields |

## Related

- Local Konflux + project-controller E2E: [local-dev-setup](../local-dev-setup/SKILL.md)
- Past PR patterns, grep helpers: [reference.md](reference.md)
- Resource type registry: `internal/template/resources.go`
- Custom template func: `hyphenize` (K8s-safe names from semver strings)
