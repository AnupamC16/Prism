# Prism Kubernetes Manifests

Apply the bundle with:

```bash
kubectl apply -f deploy/k8s/
```

Before production use, replace values in `secret.example.yaml` with real secrets or create a `prism-secrets` Secret through your secret manager.
