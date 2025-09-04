# dragonfly-p2p-webhook

A Kubernetes Mutating Admission Webhook for automatic P2P capability injection in Dragonfly. This project simplifies Kubernetes Pod configuration by automating the injection of Dragonfly's P2P proxy settings, dfdaemon socket mounts, and CLI tools through annotation-based policies.

## Description

1. **Annotation(label)-based Injection Scope**:
   The webhook supports injecting P2P configurations based on annotations(labels) at both the namespace and pod levels. By adding a specific label to a namespace, all pods within that namespace will have the P2P capabilities automatically injected. Additionally, pods can be annotated to enable or customize the injection. The priority of annotations is as follows: `pod-level annotations` > `namespace-level labels` > `webhook default config`.

   - Namespace injection:

     ```yaml
     apiVersion: v1
     kind: Namespace
     metadata:
       labels:
         dragonflyoss-injection: enabled
       name: test-namespace
     ```

   - Pod injection:

     ```yaml
     apiVersion: v1
     kind: Pod
     metadata:
       name: test-pod
       namespace: test-namespace
       annotations:
         dragonfly.io/inject: "true"
     spec:
       containers:
         - image: test-pod-image
           name: test-container
     ```

2. **P2P Proxy Environment Variable Injection**:
   To enable application traffic within the Pod to pass through the Dragonfly P2P network proxy, the Webhook will inject environment variables such as `DRAGONFLY_INJECT_PROXY` into the application container of the target Pod. The proxy address will be dynamically constructed, where the node name or IP can be obtained via the Downward API (`spec.nodeName` or `status.hostIP`), and the proxy port is retrieved from the Webhook configuration or Helm Chart, forming a proxy address in the form of `http://$(NODE_NAME_OR_IP):$(DRAGONFLY_PROXY_PORT)`. A sample yaml is as follows:

   ```yaml
   apiVersion: v1
   kind: Pod
   metadata:
     name: test-pod
     annotations:
       dragonfly.io/inject: "true" # webhook listens for this annotation
   spec:
     containers:
       - name: test-pod-cotainer
         image: test-pod-image:latest
         env:
           - name: NODE_NAME # Obtain the scheduled node name via Downward API
             valueFrom:
               fieldRef:
                 fieldPath: spec.nodeName
           - name: DRAGONFLY_PROXY_PORT # Port value obtained from Helm Chart
             value: "8001" # Assume the Helm Chart sets the port to 8001
           - name: DRAGONFLY_INJECT_PROXY # Concatenated proxy address
             value: "http://$(NODE_NAME):$(DRAGONFLY_PROXY_PORT)"
   ```

3. **dfdaemon Socket Volume Mounting**:
   `dfget` or other clients need to communicate with the dfdaemon daemon on the node via a Unix Domain Socket. The Webhook will automatically add a hostPath Volume to the Pod to expose the Socket file based on the configuration (default is `/var/run/dfdaemon.sock`) and add the corresponding VolumeMount in the target container to ensure the client can access the Socket. A sample yaml is as follows:

   ```yaml
   apiVersion: v1
   kind: Pod
   metadata:
     name: test-app-with-dfdaemon-socket
     annotations:
       dragonfly.io/inject: "true" # Annotation to trigger the Webhook
   spec:
     containers:
       - name: test-app-container
         image: test-app-image:latest
         volumeMounts:
           - name: dfdaemon-socket
             mountPath: /var/run/dfdaemon.sock # Path to dfdaemon socket inside the container
     volumes:
       - name: dfdaemon-socket
         hostPath:
           path: /var/run/dfdaemon.sock # Actual path to dfdaemon socket on the node
           type: Socket
   ```

4. **Cli Tool Injection**:
   Considering that many base container images do not include the cli tool (such as `dfget`), and manual installation is inconvenient, this project will solve this problem using an Init Container. The Webhook will automatically add an initContainer to the target Pod. This initContainer is a custom lightweight image available in both amd64 and arm64 architectures, each containing the corresponding architecture's cli tool. The Webhook will also copy the cli tool from this initContainer to a shared volume. Subsequently, the Webhook add the `DRAGONFLY_TOOLS_PATH` environment variable of the application container to add the shared volume directory where cli is located, allowing the application container to execute cli commands directly from the command line without additional user installation or specifying the full path.

   The InitContainer uses Docker's manifest list to achieve the function of automatically importing the corresponding architecture initContainer, and its build commands are as follows:

   ```bash
   # Create manifest
   docker manifest create dragonflyoss/cli-tools:latest \
   dragonflyoss/cli-tools-amd64-linux:latest \
   dragonflyoss/cli-tools-arm64-linux:latest

   docker manifest annotate dragonflyoss/cli-tools-amd64-linux:latest --arch amd64 --os linux
   docker manifest annotate dragonflyoss/cli-tools-arm64-linux:latest --arch arm64 --os linux
   docker manifest push dragonflyoss/cli-tools:latest
   ```

   Sample yaml for the injected pod:

   ```yaml
   apiVersion: v1
   kind: Pod
   metadata:
     name: test-app-with-cli-tools-image
     annotations:
       dragonfly.io/inject: "true" # Annotation to trigger the Webhook
       # The image and version fields only need to be added if you want to specify non-default values.
       dragonfly.io/cli-tools-image: "dragonflyoss/cli-tools:v0.0.1"
   spec:
    containers:
    - command:
      - sh
      - -c
      - sleep 3600
      env:
      - name: NODE_NAME
        valueFrom:
          fieldRef:
            apiVersion: v1
            fieldPath: spec.nodeName
      - name: DRAGONFLY_PROXY_PORT
        value: "4001"
      - name: DRAGONFLY_INJECT_PROXY
        value: http://$(NODE_NAME):$(DRAGONFLY_PROXY_PORT)
      - name: DRAGONFLY_TOOLS_PATH
        value: /dragonfly-tools-mount
      image: busybox:latest
    initContainers:
    - command:
      - cp
      - -rf
      - /dragonfly-tools/.
      - /dragonfly-tools-mount/
      image: dragonflyoss/cli-tools:latest
      imagePullPolicy: IfNotPresent
      name: d7y-cli-tools
      volumeMounts:
      - mountPath: /dragonfly-tools-mount
        name: d7y-cli-tools-volume
       containers:
         - name: test-app-container
           image: test-app-image:latest
           env:
             - name: DRAGONFLY_TOOLS_PATH
               value: "/dragonfly-tools-mount"
           volumeMounts:
             - name: dragonfly-tools-volume
               mountPath: /dragonfly-tools-mount
     volumes:
       - name: dragonfly-tools-volume
         emptyDir: {}
   ```

## Getting Started

### Prerequisites

- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- cert-manager for automatic certificate issuance

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/dragonfly-p2p-webhook:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/dragonfly-p2p-webhook:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
> privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

> **NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/dragonfly-p2p-webhook:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/dragonfly-p2p-webhook/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
kubebuilder edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under 'dist/chart', and users
   can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.

## Contributing

// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
