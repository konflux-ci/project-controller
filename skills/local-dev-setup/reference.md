# Local Development Setup — Reference

## Assumptions

| Item | Value |
|------|-------|
| Kind cluster name | `konflux` (default from `deploy-local.sh`) |
| kubectl context | `kind-konflux` |
| Konflux UI | https://localhost:9443 |
| Demo login | `user1@konflux.dev` / `password` |
| Tenant namespace | `default-tenant` |
| Controller namespace | `project-controller-system` |
| Deployment name | `project-controller-controller-manager` |

Konflux install: [Konflux Operator docs](https://konflux-ci.dev/konflux-ci/docs/installation/install-local/).

## In-cluster deploy via local registry

Konflux Kind exposes a registry at `localhost:5001`. Use when `imagePullPolicy`
must pull from registry instead of `kind load`:

```bash
make docker-build IMG=localhost:5001/project-controller:dev
docker push localhost:5001/project-controller:dev
make install
make deploy IMG=localhost:5001/project-controller:dev
```

## Host run details

`make run` executes `go run ./cmd/main.go` after `manifests generate fmt vet`.
It binds health probes on `:8081` by default.

Only one manager should run per cluster (host **or** in-cluster deployment).
Undeploy in-cluster controller before using `make run`:

```bash
make undeploy
make run
```

## Verify template substitution

After applying samples (see SKILL.md), confirm fields are **resolved**, not
literal `{{.varName}}`:

```bash
NS=default-tenant
kubectl get component -n "$NS" -o yaml | rg 'containerImage|git-provider'
```

For a specific new field from [add-template-field](../add-template-field/SKILL.md):

```bash
kubectl get component -n "$NS" -o yaml | rg '<your-field>'
```

Trigger re-reconcile by annotating the PDS or editing the PDST.

## Troubleshooting

| Symptom | Check |
|---------|-------|
| `no matches for kind "Project"` | `make install` not run |
| PDS `TemplateGenerationFailed` | `kubectl describe pds -n $NS`; fix template/allowlist |
| Applications not created | Controller logs; RBAC on appstudio CRs |
| Literal `{{.foo}}` in Component | Field missing from `templateAbleFields` — see add-template-field skill |
| `connection refused` on make run | Wrong kubeconfig / cluster down |

Controller logs (in-cluster):

```bash
kubectl logs -n project-controller-system \
  deployment/project-controller-controller-manager -f
```

CRD and deployment status:

```bash
kubectl get crd | rg projctl
kubectl get deploy -n project-controller-system
```

## Makefile targets

| Target | Purpose |
|--------|---------|
| `make install` | Install projctl CRDs |
| `make uninstall` | Remove CRDs |
| `make deploy IMG=...` | Deploy controller to cluster |
| `make undeploy` | Remove controller deployment |
| `make docker-build IMG=...` | Build manager container image |
| `make run` | Run manager on host against current kube context |
| `make build` | Build `bin/manager` only |
