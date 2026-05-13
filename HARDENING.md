# Hardening Guide

The default Helm chart is opinionated for safety. Production deployments may
want to go further:

- Set `--metrics-secure=true` and provide TLS material via volume mounts.
- Apply `networkPolicy.enabled=true` and limit ingress to your Prometheus ns.
- Run `helm install ... -f charts/athos-kubernetes/values-strict.yaml` for
  the recommended defaults.
- Restrict the default ClusterRole to the workloads Athos actually governs
  using a Role + RoleBinding under each watched namespace.
- Pin the manager image by digest, not tag.
- Configure pod-level `priorityClassName` for the operator so that an
  overloaded node still schedules the controller.
- Pair the operator deployment with a PodDisruptionBudget so voluntary
  evictions cannot remove the only replica during a drain.

## Threat model

Athos is a control-plane component. It runs with the privilege required to
create StatefulSets, Services, Secrets, ConfigMaps and PodDisruptionBudgets
in the namespaces it watches, plus the cluster-scoped permissions needed
for leader election and CRD validation. The threat model assumes:

- An attacker who can submit arbitrary `PostgresCluster`, `PostgresBackup`
  or `PostgresUser` resources can already manage workloads in the namespace.
  Athos validates inputs but does not attempt to constrain the namespace
  administrator.
- The operator itself does not run user-supplied SQL. The only SQL paths
  that quote user input route through `internal/sqlescape` and are unit-tested
  with adversarial fixtures.
- The operator never logs or surfaces password literals. `PASSWORD` clauses
  are redacted before `psql` stderr is propagated up.
