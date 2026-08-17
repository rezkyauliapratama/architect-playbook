# Architect Playbook

A personal solution-architecture playbook: reusable IaC blueprints and reference patterns for cloud-native systems (GCP-first, AWS secondary).

## Structure

```
cloud/
├── gcp/terraform/
│   ├── compute_engine/basic_vm/    # minimal GCE VM with API enablement
│   ├── gke/basic_public/           # public GKE cluster
│   └── network/                    # VPC, NAT, basic & GKE-ready networks
kubernetes/                         # k8s manifests & workload examples
platforms/                          # platform tooling references
services/                           # service patterns
```

## Terraform modules

- `compute_engine/basic_vm` — GCE VM + required APIs + optional shutdown script
- `gke/basic_public` — public GKE cluster with node pool
- `network/basic` / `basic_for_gke` / `basic_nat` — VPC, subnets, Cloud NAT

## Usage

```bash
cd cloud/gcp/terraform/<module>
terraform init && terraform plan && terraform apply
```

## Status

Active reference — continuously updated with production lessons from GCP/AWS work.
