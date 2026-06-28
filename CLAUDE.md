# tekton-custom-task

`github.com/KubeRocketCI/tekton-custom-task` — Go kubebuilder operator that implements Tekton's [Custom Task](https://tekton.dev/docs/pipelines/customruns/) extension point to provide a human-approval gate in CI/CD pipelines. It defines the `ApprovalTask` CRD (`edp.epam.com/v1alpha1`) and a controller that watches Tekton `CustomRun` objects, pausing a `PipelineRun` until a human approves or rejects it.

## Build & Test

```bash
make build          # compile → dist/manager-<arch>
make test           # envtest-based integration tests (downloads K8s binaries via setup-envtest)
make lint           # golangci-lint (config: .golangci.yml)
make lint-fix       # golangci-lint with auto-fix
make fmt            # go fmt ./...
make vet            # go vet ./...
```

Run a single test package:

```bash
KUBEBUILDER_ASSETS="$(./bin/setup-envtest use --bin-dir ./bin -p path)" \
  go test ./internal/controller/... -v -run "Should create ApprovalTask"
```

First-time setup (downloads all tool binaries into `./bin/`):

```bash
make setup-envtest
```

## Architecture

### Key Directories

```
cmd/main.go                        — manager entry point; registers schemes and the CustomRun controller
api/v1alpha1/
  approvaltask_types.go            — ApprovalTask spec/status; action enum: Pending|Approved|Rejected|Canceled
  groupversion_info.go             — GroupVersion = edp.epam.com/v1alpha1
  zz_generated.deepcopy.go        — generated; do not edit
internal/controller/
  customrun_controller.go          — sole controller; reconciles tekton.dev/v1beta1 CustomRun objects
  customrun_controller_test.go     — Ginkgo/Gomega integration tests using envtest
  suite_test.go                    — envtest bootstrap; loads CRDs from config/crd/bases + hack/crd/
config/crd/bases/                  — generated CRD manifests (Kustomize source)
deploy-templates/                  — Helm chart; crds/ subdirectory mirrors config/crd/bases/
hack/crd/customrun.yaml            — Tekton CustomRun CRD bundled for local envtest (not installed in cluster)
pkg/testutils/                     — helper to locate setup-envtest binaries
```

### How the Custom Task Mechanism Works

Tekton's Custom Task feature uses `CustomRun` (group `tekton.dev`, version `v1beta1`) as a generic execution unit. When a `PipelineRun` reaches a task step that references a custom task via `customRef.apiVersion: edp.epam.com/v1alpha1` and `customRef.kind: ApprovalTask`, Tekton creates a `CustomRun` and keeps the `PipelineRun` blocked until that `CustomRun` reaches a terminal condition.

The controller (`ReconcileCustomRun`) only acts on `CustomRun` objects whose `customRef` or `customSpec` points to `edp.epam.com/v1alpha1 / ApprovalTask`. All other `CustomRun` objects are ignored immediately.

### Reconcile Flow

1. **New run**: sets `CustomRun.status` to Running; creates a child `ApprovalTask` named `<customrun-name>-approval` with owner reference to the `CustomRun`. The `description` param from the `CustomRun` spec is forwarded to `ApprovalTask.spec.description`.
2. **Pending poll** (requeues every 5 s): reads the child `ApprovalTask` and checks `spec.action`.
3. **Approved** (`spec.action: Approved`): marks `CustomRun` Succeeded; writes results `approved=true`, `approvedBy`, `comment`.
4. **Rejected** (`spec.action: Rejected`): marks `CustomRun` Failed/Rejected; writes results `approved=false`, `approvedBy`, `comment`.
5. **Cancelled or timed out**: sets `ApprovalTask.spec.action = Canceled`, then marks `CustomRun` Failed/Canceled; result `approved=false`.

Human approval is performed by patching `ApprovalTask.spec.action` to `Approved` or `Rejected` (and optionally setting `spec.approve.approvedBy` / `spec.approve.comment`). The portal UI or a CLI tool is expected to do this — the operator just polls.

### Code Generation

After modifying types in `api/v1alpha1/`:

```bash
make generate    # regenerates zz_generated.deepcopy.go
make manifests   # regenerates CRDs → config/crd/bases/ AND deploy-templates/crds/; also regenerates docs/api.md
```

`make manifests` writes CRDs to **both** `config/crd/bases/` (Kustomize source) and `deploy-templates/crds/` (Helm chart). Always run it after any API type change — the Helm chart CRD must stay in sync.

After editing `deploy-templates/`:

```bash
make helm-docs   # regenerates deploy-templates/README.md from README.md.gotmpl
```

CI validates both with `make validate-docs`.

### Testing Notes

- Tests use Ginkgo v2 + Gomega and run against a real envtest API server (no mocks for the K8s API).
- `hack/crd/customrun.yaml` supplies the Tekton `CustomRun` CRD to envtest — this is a stripped-down copy, not the full upstream CRD. Do not replace it with the full upstream CRD without testing.
- `setup-envtest` downloads K8s API server binaries matching the controller-runtime version from `go.mod` at first run.
