# k8s-config-drift - UNDER DEVELOPMENT

k8s client that uses client-go (https://github.com/kubernetes/client-go) to talk to the kubernetes api and find config drift across environments.

## Config Drift Program for Kubernetes

The intent of this CLI is to be a config-drift evaluation tool for SREs and Operations engineers.

Currently it has a single command, replicaDrift, which takes one argument (name of a deployment).

`replicaDrift` will find all instances of the deployment in your cluster (even across namespaces), and report back on the number of replicas it is set to, within each namespace.

```
config-drift replicaDrift --deployment <deployment-name>
```

### Build

```
go build ./cmd/k8s-config-drift
```

### Run

```
./k8s-config-drift <command> -etc...

# Or, place k8s-config-drift in your /usr/local/bin
k8s-config-drift <command> -etc...
```


