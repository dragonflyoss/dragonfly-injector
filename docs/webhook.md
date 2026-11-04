# Webhook configuration and failure modes

The injector registers a single mutating admission webhook for Pods at
`/mutate--v1-pod` (name `mpod-v1.d7y.io`). It reacts to `CREATE` and `UPDATE` on
`v1/pods` and is registered with `failurePolicy: Ignore` and
`sideEffects: None`. The shipped `dist/install.yaml` additionally skips the
`kube-system`, `cert-manager` and `dragonfly-system` namespaces through a
`namespaceSelector`.

## Configuration surface

| Knob | Where it comes from | Default | Notes |
| --- | --- | --- | --- |
| Injector config file | ConfigMap `dragonfly-injector-config`, key `injector.yaml` | see below | Mounted at `/etc/dragonfly/injector.yaml`; the directory is hard-coded in `injector.ConfigMapPath` and patched in `config/default/manager_webhook_patch.yaml`. |
| Config reload interval | `injector.ConfigReloadInterval` | `30s` | The file is read once at startup and re-read on every tick. |
| Webhook registration switch | env `ENABLE_WEBHOOKS` | enabled | Any value other than `false` keeps the webhook registered. |
| Webhook TLS certificate | flags `--webhook-cert-path`, `--webhook-cert-name` (`tls.crt`), `--webhook-cert-key` (`tls.key`) | self-signed | In production the certificate is issued by cert-manager and injected through `cert-manager.io/inject-ca-from`. |
| Metrics / probe endpoints | flags `--metrics-bind-address`, `--health-probe-bind-address`, `--metrics-secure` | `0`, `:8081`, `true` | Not related to injection, but they share the manager process and therefore its failure modes. |
| RBAC | `config/rbac/role.yaml` | `get;list;watch` on `namespaces;pods` | The webhook reads the Pod namespace to evaluate the namespace label. |

Config file schema (all fields optional):

```yaml
initContainerImage:
  registry: docker.io          # used as <registry>/<repository>:<tag>@<digest>
  repository: dragonflyoss/client
  tag: latest
  digest: ''
  pullPolicy: IfNotPresent     # ImagePullPolicy of the init container
  pullSecrets: []              # copied into the Pod's spec.imagePullSecrets
```

Injection is requested with the Pod annotation `dragonfly.io/inject: "true"` or
the namespace label `dragonfly.io/inject: "true"`. The Pod annotation wins: when
it is present, only its value is considered and the namespace label is ignored.
Further Pod annotations:

- `dragonfly.io/skip-unix-sock-inject: "true"` – inject the binaries only, do not
  mount the dfdaemon unix socket;
- `dragonfly.io/init-container-image: <reference>` – override the init container
  image for that Pod.

## Failure modes that are tolerated

- **Config file missing, unreadable or invalid YAML** – `injector.LoadConfig`
  logs the error and falls back to `injector.DefaultConfig()` (image
  `docker.io/dragonflyoss/client:v1.3.0`, `IfNotPresent`, no pull secrets).
  Injection keeps working with those defaults, and a later fix of the ConfigMap
  is picked up by the periodic reload.
- **Config reload fails** – the previously loaded configuration stays in place;
  loading never disables injection.
- **Namespace lookup fails** (namespace deleted, or RBAC denies `get` on
  namespaces) – the error is logged and the Pod is admitted without injection.
  `Default` still returns `nil`, so the webhook does not deny the Pod.
- **Malformed AdmissionReview** (`request` is `null`, `object` is missing or has
  an empty `raw`, the object is not a `Pod`, or the body is not JSON) – decoding
  fails with an explicit error and the webhook answers with a `400`. Combined
  with `failurePolicy: Ignore`, the API server then admits the Pod unmodified
  instead of rejecting it.
- **Webhook server unavailable** (crash, bad certificate, network policy) – the
  API server gives up on the webhook because of `failurePolicy: Ignore` and the
  Pod is created without injection. This is a silent degradation: no error shows
  up on the Pod, only in the API server audit log and the injector logs.

## Failure modes that are not tolerated

- **dfdaemon socket missing on the node.** The injected hostPath volume uses
  `type: Socket` for `/var/run/dragonfly/dfdaemon.sock`. If dfdaemon is not
  running on that node, kubelet cannot perform the hostPath type check and the
  Pod stays in `ContainerCreating`. Run dfdaemon on the node, or set
  `dragonfly.io/skip-unix-sock-inject: "true"`.
- **An empty or partial but syntactically valid `injector.yaml`.** Defaults are
  only used when the file cannot be read or parsed, not when a field is simply
  absent. For example `initContainerImage: {}` produces the image reference
  `/`, and the Pod ends up in `Init:InvalidImageName`. Always ship a complete
  image block.
- **An invalid `dragonfly.io/init-container-image` annotation.** The value is
  used verbatim and is not validated, so a typo leads to
  `Init:InvalidImageName` / `ImagePullBackOff` on impacted Pods.
- **Missing or incomplete pull secrets.** The configured `pullSecrets` are only
  references; if the secret does not exist in the Pod namespace, the init
  container cannot pull the image.

Injection is never required for a Pod to be admitted: when it cannot be decided
or performed, the Pod is admitted unchanged. The exception is the Pod spec
itself becoming invalid because of injected fields, as described above.

## Verifying a deployment

```sh
# The webhook must be present and configured with failurePolicy: Ignore.
kubectl get mutatingwebhookconfiguration dragonfly-injector-mutating-webhook-configuration -o yaml

# The current config as seen by the injector.
kubectl -n dragonfly-system get configmap dragonfly-injector-config -o yaml

# The injector logs record why a Pod was or was not injected.
kubectl -n dragonfly-system logs deploy/dragonfly-injector-controller-manager | grep -E "skip inject|Defaulting for Pod|pod namespace labels"
```
