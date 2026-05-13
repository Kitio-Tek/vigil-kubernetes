# Athos Kubernetes Helm chart museum

This branch is published as a Helm chart repository at:

    helm repo add athos https://kitio-tek.github.io/athos-kubernetes
    helm repo update
    helm search repo athos

The branch is rewritten by `chart-releaser-action` on every push to
`main` that touches `charts/**`. Do not commit changes here by hand.
