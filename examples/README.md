# Examples

Ready-to-apply manifests for common deployment shapes. Each file is a
self-contained sample and is exercised by the KUTTL e2e suite where indicated.

| File | Purpose |
|---|---|
| [cluster-basic.yaml](cluster-basic.yaml) | Smallest viable PostgresCluster (1 instance, 10Gi). |
| [cluster-ha.yaml](cluster-ha.yaml) | Highly available cluster with 3 instances and topology spread. |
| [cluster-tls.yaml](cluster-tls.yaml) | TLS-enabled cluster with cert-manager-issued certificates. |
| [backup-basebackup.yaml](backup-basebackup.yaml) | On-demand physical backup via `pg_basebackup`. |
| [backup-pgdump.yaml](backup-pgdump.yaml) | Logical backup via `pg_dump`. |
| [user-app.yaml](user-app.yaml) | Application user with database-level grants. |
| [user-readonly.yaml](user-readonly.yaml) | Read-only role for analytics workloads. |

## Apply order

For a turnkey kind walkthrough:

```bash
kubectl apply -f examples/cluster-basic.yaml
kubectl wait --for=condition=Ready pgc/example-cluster --timeout=300s

kubectl apply -f examples/user-app.yaml
kubectl apply -f examples/backup-basebackup.yaml
```
