# Deploy Plex

To deploy Plex on your cluster, create an instance of the `PlexMediaServer` custom resource in the desired namespace.

```bash
$ kubectl apply -f - <<EOF
kind: PlexMediaServer
apiVersion: plex.adambkaplan.com/v1alpha1
metadata:
  name: plex
spec:
  claimToken: <claim-token>
  version: latest
EOF
```

## Deployment Spec Options

The following options can be added to the `spec` YAML object:

| Config | Description | Default |
| ------ | ----------- | ------- |
| `claimToken` | Claim token for your Plex Media Server. Visit [https://www.plex.tv/claim](https://www.plex.tv/claim) to obtain a token | `""` |
| `version` | Version of Plex to deploy | `latest` |
| `storage.config` | Configure persistent storage for Plex's internal database | Ephemeral storage |
| `storage.data` | Configure persistent storage for external media | Ephemeral storage |
| `storage.transcode` | Configure persistent storage for Plex's transcoded media files | Ephemeral storage |
| `storage.[*].accessMode` | Access mode needed for the desired storage | Cluster default |
| `storage.[*].capacity`| Desired storage capacity for the persistent storage | None |
| `storage.[*].storageClassName` | Storage class used to select a persistent storage provisioner | Cluster default |
| `storage.[*].selector` | Label selector used to find persistent storage | None |
| `networking.externalServiceType` | Service type to expose Plex outside of the Kubernetes cluster. Can be empty, `NodePort`, or `LoadBalancer` | Empty - no external access |
| `networking.enableDiscovery` | Enable GDM discovery outside of the cluster. This lets Plex be discovered by other devices on the network. | `false` |
| `networking.enableDLNA` | Enable DLNA access | `false` |
| `networking.enableRoku` | Enable communication with Roku devices on the network | `false` |

## Real world example

The following is an example deployment that has the following attributes:

1. The `config` database persisted to a block storage device, provided by the cluster's default storage provisioner.
2. The `data` directory persisted to an NFS share, managed by a custom `nfs` storage class.
   The cluster already has a PersistentVolume labeled `media: plex` with the desired media uploaded.
3. Plex's web and Roku ports exposed ouside of the cluster via a load balancer.

```yaml
apiVersion: plex.adambkaplan.com/v1alpha1
kind: PlexMediaServer
metadata:
  name: plex
spec:
  networking:
    externalServiceType: LoadBalancer
    enableRoku: true
  storage:
    config:
      accessMode: ReadWriteOnce
      capacity: 10Gi
    data:
      accessMode: ReadWriteMany
      storageClassName: nfs
      capacity: 100Gi
      selector:
        matchLabels:
          media: plex
```
