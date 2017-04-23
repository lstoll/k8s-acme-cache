[![Build Status](https://travis-ci.org/micahhausler/k8s-acme-cache.svg)](https://travis-ci.org/micahhausler/k8s-acme-cache)
[![https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](http://godoc.org/github.com/micahhausler/k8s-acme-cache/)

# k8s-acme-cache

An ACME [autocert](https://godoc.org/golang.org/x/crypto/acme/autocert#Cache)
cache that stores keys as Kubernetes secrets.

## Required RBAC permissions

```yaml
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: Role
metadata:
  name: <role-name>
rules:

- apiGroups:
  - ""
  resources:
  - secret
  resourceNames: 
  - <secret-name>
  verbs:
  - get
  - create
  - update
```

You'll also need a `RoleBinding` to bind the above role to the `ServiceAccount` 
the application is assigned.

If the secret you want to use is in a different namespace than the application,
use a `ClusterRole`, and a `ClusterRoleBinding` 

## License
MIT License. See [License](/LICENSE) for full text
