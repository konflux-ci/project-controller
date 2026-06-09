---
name: local-dev-setup
description: >-
  Use when deploying or iterating on konflux-ci/project-controller against a
  local Konflux Kind cluster, when verifying template reconciliation end-to-end,
  when make run or make deploy against kind-konflux is needed, or when
  troubleshooting project-controller on default-tenant after a local Konflux
  install.
---

# Local Development Setup

Deploy a **local build of project-controller** on top of an existing Konflux
Kind cluster. Konflux does **not** ship project-controller — install Konflux
first, then layer this controller on top.

**Konflux install:** follow [Konflux Operator local deployment](https://konflux-ci.dev/konflux-ci/docs/installation/install-local/).
Minimal path (no GitHub integration): `SKIP_SECRETS=true ./scripts/deploy-local.sh`
in [konflux-ci/konflux-ci](https://github.com/konflux-ci/konflux-ci).

## Prerequisites

- Running Kind cluster `konflux` with Konflux ready
- `kubectl` context: `kind-konflux`
- `docker` or `podman` (for in-cluster mode)
- project-controller repo checkout

```bash
kubectl config use-context kind-konflux
kubectl get ns default-tenant   # Konflux shared tenant namespace
```

## Modes

| Mode | When | Iteration |
|------|------|-----------|
| **Host run** (default) | Active controller development | Rebuild binary; restart `make run` |
| **In-cluster** | Match production deploy, test container image | Rebuild image; reload; rollout restart |

If the user doesn't specify, use **host run**.

## Checklist

```
- [ ] 1. Konflux running on kind-konflux (see link above)
- [ ] 2. Install project-controller CRDs (make install)
- [ ] 3. Run controller (make run OR make deploy)
- [ ] 4. Apply test CRs in default-tenant
- [ ] 5. Verify reconciliation
```

## Host run (fast iteration)

Install CRDs once (or after CRD/RBAC changes):

```bash
make install
```

Run the manager against the Kind cluster — long-running; background it:

```bash
make run
```

Uses `~/.kube/config` and current context. Leader election is off by default
(deployment sets `--leader-elect=false`).

After code changes: stop `make run`, then restart it. No image build required.

## In-cluster deploy

Build, load into Kind, install CRDs, deploy:

```bash
make docker-build IMG=project-controller:dev
kind load docker-image project-controller:dev --name konflux
make install
make deploy IMG=project-controller:dev
```

Controller runs in namespace `project-controller-system`.

**Iteration loop** after code changes:

```bash
make docker-build IMG=project-controller:dev
kind load docker-image project-controller:dev --name konflux
kubectl rollout restart deployment/project-controller-controller-manager \
  -n project-controller-system
kubectl rollout status deployment/project-controller-controller-manager \
  -n project-controller-system
```

Alternative: push to Konflux local registry (`localhost:5001`) if using that
pull policy — see [reference.md](reference.md).

## Verify project-controller

Apply sample CRs to the Konflux tenant namespace:

```bash
NS=default-tenant
kubectl apply -f config/samples/projctl_v1beta1_project.yaml -n "$NS"
kubectl apply -f config/samples/projctl_v1beta1_projectdevelopmentstreamtemplate.yaml -n "$NS"
kubectl apply -f config/samples/projctl_v1beta1_projectdevelopmentstream_w_template_vars.yaml -n "$NS"
```

Check reconciliation:

```bash
kubectl get projectdevelopmentstream -n "$NS"
kubectl describe projectdevelopmentstream projectdevelopmentstream-sample-w-template-vars -n "$NS"
kubectl get application,component -n "$NS"
```

PDS should reach `Ready=True` / `ResourcesApplied`. On template errors, events
show `TemplateGenerationFailed`.

## Teardown

```bash
make undeploy    # remove in-cluster deployment
make uninstall   # remove CRDs (destructive — deletes all projctl CRs)
```

Stop host `make run` with Ctrl+C. Konflux cluster teardown is documented in
Konflux install docs — not covered here.

## Common mistakes

| Mistake | Symptom | Fix |
|---------|---------|-----|
| Wrong kubectl context | CRDs apply to wrong cluster | `kubectl config use-context kind-konflux` |
| Skip `make install` | projctl CRDs missing | Run `make install` before run/deploy |
| Old image after in-cluster edit | Behavior unchanged | Rebuild, `kind load`, rollout restart |
| Samples in wrong namespace | Controller never reconciles tenant CRs | Use `-n default-tenant` |
| Konflux not ready | Application CRD missing | Wait for Konflux; check `default-tenant` |

## Related

- Template field E2E checks: [add-template-field](../add-template-field/SKILL.md)
- Troubleshooting, registry push, sample verification: [reference.md](reference.md)
