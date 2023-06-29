# ca-injector

A kubernetes MutatingAdmissionWebhook to inject certificate bundles into pods
based on annotations, so that off-the-shelf deployments can be deployed in
clusters with custom certificate authorities, with minimal disruption and
minimal maintenance. No more building images off of upstream base images just to
`ADD yourca.crt /usr/share/ca-certificates/trust-source/anchors/` and `RUN trust
extract-compat || update-ca-certificates` etc.

This webhook does three things:

1. Add to pods as a volume the certificate bundle specified by the value of the
   `microcumul.us/injectssl` annotation. The value should correspond with a
   secret in the same namespace as the pod which has a key `ca.crt` whose value
   is a CA bundle.
1. Add this volume to all containers as a volumemount
1. Add the `SSL_CERT_FILE` environment variable [respected by
   OpenSSL](https://www.openssl.org/docs/man3.1/man3/SSL_CTX_set_default_verify_paths.html)
   and most tls libraries.

Just deploy this in your cluster, create CA bundles as e.g. `foo-crt` secret,
with the key `ca.crt` (`kubectl create secret generic foo-crt
--from-file=ca.crt=my-bundle.crt`), and use the `microcumul.us/injectssl:
foo-crt` annotation on your pod or in your helm chart's appropriate annotations
section. `ca.crt` can be changed by setting `tls.ca.key` in any of the typical
ways (config files at `/etc/ca-injector.yaml`, `$HOME/.config/ca-injector.yaml`,
or `TLS_CA_KEY` env var all work). 

I highly suggest using this with
[replicator](https://github.com/mittwald/kubernetes-replicator) for a consistent
experience across namespaces. Another great tool that may work well for building
the bundle here is [trust-manager](https://github.com/cert-manager/trust-manager).

# Installation

```bash
kubectl apply -f k8s.yaml
kubectl apply -f mwh.yaml
```
