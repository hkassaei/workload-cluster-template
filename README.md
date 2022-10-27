# edc-demo
This demo shows how to use Crossplane and Flux to automate the provisioning of 
cloud infrastructure resources at scale, and by using GitOps principles.
On high level, the solution relies on a management cluster that can be used to bootstrap
and monitor the status of other workload clusters, where the applications/workloads
are deployed.

As a fundamental principle, the a design choice to not create too much dependency 
from workload clusters to the management cluster beyond the bootstrapping and initial 
setup phase.

Since this is a PoC, there are still a number of steps that need to be performed 
manually. Understanding those steps with the goal of automating as much of them 
as possible will be another benefit of this demo.

Finally, there are several [ways to structure the repositories](https://fluxcd.io/flux/guides/repository-structure/) based on various critria such as access control, organizational structure, etc. This demo adopts a simple and
easy-to-understand aprpoach of using a Monorepo, where infrasture components and applications
are all stored in the same repo.

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
  7. Create a directory for the workload cluster.
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

## Bootstrap Flux in the management cluster
Once you have brough up the management cluster, whether on your local machine or 
in a cloud environment, bootstrap flux on it:

```
GITHUB_USERNAME=<your github username>
GITHUB_PAT=<your GitHub Personal Access Token>

flux bootstrap github \
  --owner=${GITHUB_USERNAME} \
  --repository=edc-demo \
  --branch=main \
  --path=./management-cluster \
  --personal
```
## Create a sops secret for decrypting secret in the cluster
Make sure you have already created your own GPG key following [these instructions](https://fluxcd.io/flux/guides/mozilla-sops/)

Flux Kustomize controller has built-in support for SOPS and given the configuration,
it can decrypt the encrypted credentials and populate the secret that Crossplane
GCP provider needs in order to access GCP APIs. 

Regarding the configuration that Flux needs, look under `/infra-blueprints/deployment-artifacts/secrets` directory.

Flux also needs the encryption key to be able decrypt the credentials stored
in the git repo. To provide it with the key, do the following:

```
gpg --list-secret-keys crossplane-gcp-provider-creds  
# this command prints the key's fingerprint (key fp):
#   sec   rsa4096 2020-09-06 [SC]
#       1F3D1CED2F865F5E59CA564553241F147E7C5FA4

export KEY_FP=1F3D1CED2F865F5E59CA564553241F147E7C5FA4

# which we use in the following command:
gpg --export-secret-keys --armor "${KEY_FP}"  | kubectl create secret generic sops-gpg --namespace=flux-system --from-file=sops.asc=/dev/stdin
```
## Instruct Flux to deploy Crossplane and related artifacts
All the setup has already been taken care of in the repo under
`/infra-blurprints/crossplane` and `/infra-blueprints/deployment-artifacts`. All you 
need to do is to point flux to the prepare kustomizations by copying
a prepared crossplane.yaml from `/utils/sync-crossplane` to the `/management-cluster` directory:
```
cp /utils/sync-crossplane/crossplane.yaml /management-cluster
```

In addition, set up the flux controller in the management-cluster to sync the changes
under the workload clusters (mainly the claims for new workload clusters)
```
cp /utils/sync-workload-clusters/workload-clusters.yaml /management-cluster
``` 
and commit the file to the git repository.

## Bootstrap a workload cluster
With Flux and Crossplane deployed and configured, the management cluster
is ready to bootstrap workload clusters.

To create the first workload-cluster, simply copy a cluster claim template
from `utils/claim-templates` to `/workload-clusters` directory, and modify if needed.

```
cp /utils/claim-templates/workload-cluster-1.yaml /workload-clusters
```
and commit the file to the git repository. 

After around a minute, you should see that a new gke cluster is being created.

### GKE Cluster Creation
For more information for different types of GKE clusters, see:
  * [Regional Clusters](https://cloud.google.com/kubernetes-engine/docs/concepts/regional-clusters)
  * [Zonal Clusters](https://cloud.google.com/kubernetes-engine/docs/concepts/types-of-clusters#zonal_clusters)
  * [Mode of operation (Autopilot/Standard)](https://cloud.google.com/kubernetes-engine/docs/concepts/types-of-clusters#modes)

## Set up the workload cluster
We need to connect the new workload cluster that was created in the previous step
to a git repo. There are various ways to do this. For example, we can create a
new git repository and bootstrap a flux controller with that repo. We then need
to copy over some of the blueprints to that repository for reuse. Alternative,
we can just create a new directory in this same git repo and connect it to the
new cluster. In this demo, we do the latter.

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
And to have flux deploy Crossplane and its related artifacts:
```
cp ./utils/sync-crossplane/crossplane.yaml ./workload-clusters/workload-cluster-1
```
and finally commit the changes to the repo.

## Deploy a sample application with dependency to CloudSQL
Prerequisite: You have created an IAM service account with that is bound to a role
that allows it to create and access CloudSQL databases. An example of how to do this:
```
export PROJECT_ID=$(gcloud config list --format 'value(core.project)')
export CLOUDSQL_SERVICE_ACCOUNT=cloudsql-service-account
gcloud iam service-accounts create $CLOUDSQL_SERVICE_ACCOUNT --project=$PROJECT_ID
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$CLOUDSQL_SERVICE_ACCOUNT@$PROJECT_ID.iam.gserviceaccount.com"  \
    --role="roles/cloudsql.admin"
```
### Enable workload identity on the gke cluster
[Workload identity](https://cloud.google.com/sql/docs/postgres/connect-kubernetes-engine#workload-identity) binds a Kubernetes Service Account (KSA) to a Google Service Account (GSA) enabling any workload (e.g. modeled as a Kubernetes Deployment)
with that KSA to authenticate as the GSA in their interaction with Google Cloud.

To enable workload identity (not needed if using autopilot since in that case wi is enabled by default):
```
gcloud container clusters update workload-cluster-1 --zone=us-central1-a --workload-pool=gcprdpscdpochcppaasdev01-c304.svc.id.goog
```

```
cp ./utils/sync-app/app.yaml ./workload-clusters/workload-cluster-1
```
Once Crossplane gcp provider succeeds in instantiating a CloudSQL instance, it stores 
the connection data in a `db-conn-cloudsql` secret. The data field of the secret
conatins the endpoint (IP address), username, password, and port number.

## Clean up
  *  Delete the claim for CloudSQL to trigger Crossplane to stop CloudSQL instance
     * what's the best way to do this? Remove the claim from kustomization file
       under /app-blueprints/app1? Seems more natural to not touch the blueprints and 
       only remove something from workload-cluster-1 directory.
  *  Once the removal of CloudSQL is finished, remove the workload cluster by deleting
     the workload-cluster-1 claim from the workload-clusters directoy.