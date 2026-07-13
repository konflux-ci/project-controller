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
| One-time cross-controller signal | `liveStateConditionalFields` | Conditionally included based on live state |

**Pitfall:** `spec.image.name` on ImageRepository is `templateAbleFields`, not name
fields — it may contain `/`.

### Field category details

#### `liveStateConditionalFields` semantics

Fields in `liveStateConditionalFields` are included in the SSA desired state
when a resource is **created**, but on **updates** the reconciler inspects the
live resource first: the field is only included in the desired state if it is
still present and non-empty in the live resource. If the field is absent or
empty in the live resource, it is removed from the desired state before the
SSA patch is applied.

This prevents the reconciler from re-applying a field that an external
controller has already processed and removed. Without this category, the
reconciler's SSA patch would re-add the field on every reconcile, undoing
the external controller's work.

A field listed in `liveStateConditionalFields` must **also** appear in
`templateAbleFields` (or `templateAbleNameFields`) so that template
substitution is applied to it. The `liveStateConditionalFields` list only
controls inclusion in the desired state — it does not trigger template
processing on its own.

#### Decision guide — `liveStateConditionalFields` vs `createOnlyFields`

| Criterion | `createOnlyFields` | `liveStateConditionalFields` |
|-----------|-------------------|------------------------------|
| **When included in SSA patch** | Only on resource creation | On creation always; on update only if present and non-empty in live resource |
| **On subsequent reconciles** | Stripped from desired state unconditionally | Stripped only if absent or empty in live resource |
| **Use when** | Field is fire-and-forget — set once, never managed again by this controller | Field must persist across reconciles until an external controller processes and removes it |
| **Risk if wrong category** | External controller never sees the field (removed too early by SSA) | Field re-appears after external controller removes it (SSA re-adds it) |

Use **`createOnlyFields`** when the annotation triggers a one-time action and
the reconciler should never re-apply it — even if the external controller has
not yet processed it. Example: `build.appstudio.openshift.io/request` on
Component tells build-service to submit an initial build. Once the resource
exists, the reconciler strips this field from all future SSA patches.

Use **`liveStateConditionalFields`** when the annotation must survive
multiple reconcile cycles until the external controller processes and removes
it. The reconciler keeps including the field as long as the live resource
still has it, and stops only after the external controller clears it.
Example: `image-controller.appstudio.redhat.com/update-component-image` on
ImageRepository tells image-controller to set `spec.image.name` on the
associated Component. Image-controller removes the annotation once
provisioning completes; only then does the reconciler stop including it.

#### Cross-controller annotation lifecycle

Two annotations currently use these field categories. Changes to their
category affect cross-controller contracts — do not reclassify without
coordinating with the owning controller's team.

| Annotation | Resource | Category | External controller | Lifecycle |
|-----------|----------|----------|--------------------|----|
| `image-controller.appstudio.redhat.com/update-component-image` | ImageRepository | `liveStateConditionalFields` | image-controller | Applied on creation. Image-controller reads it, sets `spec.image.name` on the Component, then removes the annotation. Reconciler keeps including it until image-controller removes it. |
| `build.appstudio.openshift.io/request` | Component | `createOnlyFields` | build-service | Applied on creation only. Build-service reads it to trigger an initial build pipeline. Reconciler never re-applies it on subsequent reconciles. |

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
- [ ] 2. Pick category (name vs general vs untouchable vs createOnly vs liveStateConditional)
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
