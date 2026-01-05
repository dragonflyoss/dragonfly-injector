# Install

## Deploy

1. Prerequisites

   - go version v1.24.0+
   - docker version 17.03+.
   - kubectl version v1.11.3+.
   - Access to a Kubernetes v1.11.3+ cluster.
   - cert-manager for automatic certificate issuance
   - dragonfly installed and running in the cluster

2. Clone the repository:

    ```sh
    git clone https://github.com/dragonflyoss/dragonfly-injector.git
    ```

3. Build the dragonfly-injector docker image

    ```sh
    make docker-build
    # Use arg `IMG` to specify the image name and tag, default is `d7y.io/dragonfly-injector:latest`.
    # Example: `make docker-build IMG=example.com/dragonfly-p2p-webhook:v0.0.1 `
    ```

    > If you use kind to deploy the cluster, you need to load the image into the cluster.
    > Example: `kind load docker-image d7y.io/dragonfly-injector:latest`

4. Configure Webhook
    You can modify the webhook configuration by editing the configuration file at `config/webhook/config-map.yaml`.
5. Deploy the dragonfly-injector

    ```sh
    make deploy IMG=d7y.io/dragonfly-injector:latest
    ```

6. Verify the deployment

    ```sh
    kubectl -n dragonfly-injector-system get deployment dragonfly-injector-controller-manager
    ```

7. Test the deployment

    Use the following pod to test the deployment:

    ```yaml
    apiVersion: v1
    kind: Pod
    metadata:
      name: test-pod
      annotations:
        dragonfly.io/inject: "true"
    spec:
      containers:
        - name: busybox-container
          image: debian:stable-slim
          imagePullPolicy: IfNotPresent
          command: ["/bin/sh", "-c", "echo 'Hello from BusyBox!'; sleep 3600"]
          resources:
            limits:
              memory: "128Mi"
              cpu: "100m"
            requests:
              memory: "64Mi"
              cpu: "50m"
    ```

8. Verify the test pod

    Check the logs of the test pod

    ```sh
    kubectl -n default get pod -o yaml
    ```

    You should see the following output in yaml:

    ```yaml
    initContainers:
      - command:
          - cp
          - -rf
          - /dragonfly-tools/.
          - /dragonfly-tools-mount/
        image: dragonflyoss/toolkits:latest
        imagePullPolicy: IfNotPresent
        name: d7y-cli-tools
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
        volumeMounts:
          - mountPath: /dragonfly-tools-mount
            name: d7y-cli-tools-volume
    ```

## Undeploy

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

    ```sh
    make build-installer IMG=<some-registry>/dragonfly-injector:tag
    ```

    **NOTE:** The makefile target mentioned above generates an 'install.yaml'
    file in the dist directory. This file contains all the resources built
    with Kustomize, which are necessary to install this project without its
    dependencies.

2. Using the installer

    Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
    the project, i.e.:

    ```sh
    kubectl apply -f dist/install.yaml
    ```
