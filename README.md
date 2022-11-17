# Crossplane + Flux + App using CloudSQL  
This repository is part (the workload cluster part) of a demo that shows how to use
Crossplane and Flux to automate the provisioning of cloud infrastructure resources at scale,
and by using GitOps principles.

On high level, the solution relies on a 'management cluster' that can be used to bootstrap
and monitor the status of other 'workload clusters', where the applications/workloads
are deployed.

By design, there was a goal to  not create too much dependency from workload clusters 
to the management cluster beyond the bootstrapping and initial setup phase. As a result, 
the management cluster runs an instance of Crossplane that can spin up workload clusters (GKE clusters)
by leveraging the defined Kubernetese Composite resource and compositions.

EAch workload cluster runs its own instance of Crossplane that can provision other
GCP managed services needed by the application, in this case CloudSQL for PostgreSQL.

Since this is a PoC, there are still a number of steps that need to be performed 
manually. Understanding those steps with the goal of automating as much of them 
as possible will be another benefit of this demo.

## Overview
The following steps describe how to run this demo. There are also some prerequisites
to be able run the demo.

### Prerequisites
  * A Kubernetes cluster for running the management cluster. This can be either
  a local cluster such as `Minikube`, `Kind`, or `k3d` or a managed Kubernetes 
  offering such as `GKE`, `EKS`, or `AKS`.
  * Kubernetes CLI, [kubectl](https://kubernetes.io/docs/tasks/tools/)
  * Flux CLI, [flux](https://fluxcd.io/flux/cmd/)
  * GPG and Mozilla SOPS, [sops](https://fluxcd.io/flux/guides/mozilla-sops/)
  * Access to a managed Kubernetes service. In this demo we use GKE, so GCP access and [gcloud](https://cloud.google.com/sdk/gcloud)
  * A Git repository provider. In this demo, we use github. Make sure you create your
    [Personal Access Token (PAT)](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html) before starting the demo.

### Steps
  0. Install all the prerequisites tools on your machine.
  1. Follow [these instructions](https://fluxcd.io/flux/guides/mozilla-sops/) to encrypt your GCP credentials before storing them in the git repo.
  2. Bootstrap Flux in the management cluster.
  3. Create a secret to provide Flux with the key for decrypting GCP credentials in the cluster.
  4. Instruct Flux to deploy Crossplane, Crossplane GCP provider, and all the
     needed configurations and XRDs in the management cluster.
  5. Submit a claim for a gke cluster to the management cluster.
  6. Wait for the workload (GKE) cluster to be in ready state.
        * Run `gcloud container clusters get-credentials CLUSTER_NAME --region=REGION_NAME`
          to fetch the kubeconfig for the GKE cluster.
  7. Create a deployment repository for the workload cluster.
  8. Bootsrap Flux on the workload cluster.
        * make sure to first switch kubeconfig context to the gke cluster.
  9. Create a secret for sops to provide Flux with the key for decrypting GCP credentials in the cluster.
        * Basically, the same step as 3 but for the workload cluster.
  10. Instruct Flux to deploy Crossplane, Crossplane GCP provider, and all the
      needed configurations and XRDs in the management cluster.
        * Basically, the same step as 4 but for the workload cluster.
  11. Instruct Flux to deploy the application to the workload cluster.
      * Note that the application has a dependency to CloudSQL, which triggers 
        the instantiation of a CloudSQL instance.

## Set up the management cluster
Make sure to follow the instructions in the management cluster repository to set up the management cluster
before following the steps below.

## Set up the workload cluster
We need to connect the new workload cluster that was created in the previous step
to a git repo.

Make sure you create a directory that matches the name of the workload cluster
that you set in the cluster claim. This makes it easier to track clusters and repos.

Next, repeat the same steps as in the management cluster to bootstrap flux,
but first get the kubeconfig for the newly created gke cluster:

`gcloud container clusters get-credentials CLUSTER_NAME --region=REGION_NAME`

  * !!! Note: Before running the commands below, make sure you switch kubectl context
to point to the workload cluster: `kubectl config use-context <THE GKE CLUSTER NAME>`

```
mkdir -p ./workload-clusters/workload-cluster-1
flux bootstrap github \
  --owner=${GITHUB_USERNAME} \
  --repository=edc-demo \
  --branch=main \
  --path=./workload-clusters/workload-cluster-1 \
  --personal
```
Create a sops secret and point flux to the kustomization need to deploy crossplane:

```
gpg --list-secret-keys crossplane-gcp-provider-creds  
# this command prints the key's fingerprint (key fp):
#   sec   rsa4096 2020-09-06 [SC]
#       1F3D1CED2F865F5E59CA564553241F147E7C5FA4

export KEY_FP=1F3D1CED2F865F5E59CA564553241F147E7C5FA4

# which we use in the following command:
gpg --export-secret-keys --armor "${KEY_FP}"  | kubectl create secret generic sops-gpg --namespace=flux-system --from-file=sops.asc=/dev/stdin
```

## Deploy a sample application with dependency to CloudSQL
Copy the files listed in ./application/blueprint/kustomization.yaml along with the kustomization.yaml 
file itself to ./application/deployment directory and push the changes to the remote git repository.

TODO:  store the application container image in a public repository

### Enable workload identity on the gke cluster
The Crossplane Kubernetes composition that implements the creation of the gke cluster configures 
workload identity, so no manual step is needed here.
