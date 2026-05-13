# Charts

The Helm chart in `charts/athos-kubernetes/` is published alongside each
release and uploaded as a `.tgz` asset on the GitHub Release page. The
rolling chart museum is hosted on the `gh-pages` branch and is available at:

```bash
helm repo add athos https://kitio-tek.github.io/athos-kubernetes
helm repo update
helm search repo athos
```

Chart releases are tagged `helm-v<version>` and marked as pre-releases on
GitHub so that the operator's `v0.x.y` releases remain the user-facing
Latest entry.

| Chart version | Operator version | Min Kubernetes | Notes |
|---|---|---|---|
| 0.9.0 | 0.9.0 | 1.25 | Chainsaw e2e suite, CodeQL v4, dep CVE clears |
| 0.8.1 | 0.8.0 | 1.25 | First chart-museum publish via gh-pages |
| 0.8.0 | 0.8.0 | 1.25 | Security hardening, sqlescape coverage |
| 0.7.0 | 0.7.0 | 1.25 | govulncheck wired in CI, PDB verbs widened |
| 0.6.0 | 0.6.0 | 1.25 | Backup robustness fixes |
| 0.5.0 | 0.5.0 | 1.25 | User-management drift detection |
| 0.4.0 | 0.4.0 | 1.25 | Replica scaling refinements |
| 0.3.0 | 0.3.0 | 1.25 | PITR scaffolding |
| 0.2.0 | 0.2.0 | 1.25 | Metrics endpoint |
| 0.1.0 | 0.1.0 | 1.25 | Initial preview |

## Installation

```bash
helm install athos-kubernetes charts/athos-kubernetes/ \
  --namespace athos-system \
  --create-namespace
```

## Values reference

See `charts/athos-kubernetes/values.yaml` for the full list of configurable
parameters. The most commonly tuned keys are:

| Key | Default | Description |
|---|---|---|
| `image.repository` | `ghcr.io/kitio-tek/athos-kubernetes` | Manager image repository |
| `image.tag` | matches the chart's `appVersion` | Manager image tag |
| `image.pullPolicy` | `IfNotPresent` | Image pull policy |
| `replicaCount` | `1` | Number of manager replicas (set 2+ to enable leader election failover) |
| `metrics.enabled` | `true` | Enable the `:8443` metrics endpoint |
| `webhook.enabled` | `false` | Enable the validating/mutating webhook server |
| `networkPolicy.enabled` | `false` | Restrict ingress to the manager Pod |
| `resources` | sane defaults | CPU/memory requests and limits |

## Upgrading

The chart is forward-compatible across minor versions while the operator is
in 0.x preview. To upgrade, bump the chart version and re-run `helm upgrade`:

```bash
helm upgrade athos-kubernetes charts/athos-kubernetes/ \
  --namespace athos-system
```

CRD changes are not auto-applied by `helm upgrade`. Apply the new CRDs first:

```bash
kubectl apply -f config/crd/bases/
```
