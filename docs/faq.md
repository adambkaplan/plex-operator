# Frequently Asked Questions

## How is this operator different from other Plex operators?

I decided to use a [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) to deploy Plex Media Server, whereas most other Plex operators use a standard [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) and persistent volume claims [(PVCs)](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims).
Deployments are easier to configure at the outset, but default to behavior that is not suited for a stateful application.
Plex Media Server is a stateful application, with its own database and a standard mount for user data.
It is designed to be run as a singleton instance on a home computer.

## Why write an operator? Won't a Helm Chart suffice?

Writing an operator is not for the faint of heart.
Learning [Go](https://golang.org/) can be a significant barrier for most developers.
Furthermore, a well written operator requires deep knowledge of what Kubernetes will (and won't) let you do.
I found myself learning a lot of Kubernetes quirks by writing this operator, especially default values that can be injected via admission webhooks and controller reconciliations.

[Helm](https://helm.sh/) is certainly a great tool for templating and deploying applications.
The New Stack has a great [interview with Darren Shepherd](https://thenewstack.io/kubernetes-when-to-use-and-when-to-avoid-the-operator-pattern/), where he makes the case that for many applications YAML-based tooling (Helm, Kustomize, manifests+GitOps) is the way to go.

In analyzing Plex Media Server, I found these features make it complex to deploy:

1. The `ADVERTISE_IP` environment variable.
   When exposing Plex via a LoadBalancer, the external IP is dynamically provisioned.
   Therefore, _something_ needs to provision the LoadBalancer, wait for the external IP to be provisioned, and then set the environment variable.
   An operator is perfectly suited for this task.
2. Provisioning and configuring persistent storage.
   StatefulSet will only let you set volume claim templates when it is created.
   Modifying the storage can only be done by deleting and re-creating the StatefulSet.
