# Oracle Cloud Infrastructure - Free Tier
# Provisions a single ARM instance (4 OCPU, 24GB RAM) for k3s
#
# Prerequisites:
#   1. Create an Oracle Cloud account (free tier)
#   2. Generate an API key in your OCI console
#   3. Set the variables below or use terraform.tfvars
#
# Usage:
#   terraform init
#   terraform plan
#   terraform apply

terraform {
  required_providers {
    oci = {
      source  = "oracle/oci"
      version = "~> 5.0"
    }
  }
}

provider "oci" {
  tenancy_ocid     = var.tenancy_ocid
  user_ocid        = var.user_ocid
  fingerprint      = var.fingerprint
  private_key_path = var.private_key_path
  region           = var.region
}

# Variables
variable "tenancy_ocid" {
  description = "OCI Tenancy OCID"
  type        = string
}

variable "user_ocid" {
  description = "OCI User OCID"
  type        = string
}

variable "fingerprint" {
  description = "OCI API Key Fingerprint"
  type        = string
}

variable "private_key_path" {
  description = "Path to OCI API private key"
  type        = string
}

variable "region" {
  description = "OCI Region"
  type        = string
  default     = "us-ashburn-1"
}

variable "compartment_ocid" {
  description = "OCI Compartment OCID (use tenancy OCID for root compartment)"
  type        = string
}

variable "ssh_public_key" {
  description = "SSH public key for instance access"
  type        = string
}

# Data sources
data "oci_identity_availability_domains" "ads" {
  compartment_id = var.tenancy_ocid
}

data "oci_core_images" "ubuntu" {
  compartment_id           = var.compartment_ocid
  operating_system         = "Canonical Ubuntu"
  operating_system_version = "22.04"
  shape                    = "VM.Standard.A1.Flex"
  sort_by                  = "TIMECREATED"
  sort_order               = "DESC"
}

# Networking
resource "oci_core_vcn" "echofs" {
  compartment_id = var.compartment_ocid
  cidr_blocks    = ["10.0.0.0/16"]
  display_name   = "echofs-vcn"
  dns_label      = "echofs"
}

resource "oci_core_subnet" "public" {
  compartment_id    = var.compartment_ocid
  vcn_id            = oci_core_vcn.echofs.id
  cidr_block        = "10.0.1.0/24"
  display_name      = "echofs-public"
  dns_label         = "public"
  security_list_ids = [oci_core_security_list.echofs.id]
  route_table_id    = oci_core_route_table.echofs.id
}

resource "oci_core_internet_gateway" "echofs" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.echofs.id
  display_name   = "echofs-igw"
}

resource "oci_core_route_table" "echofs" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.echofs.id
  display_name   = "echofs-rt"

  route_rules {
    destination       = "0.0.0.0/0"
    network_entity_id = oci_core_internet_gateway.echofs.id
  }
}

resource "oci_core_security_list" "echofs" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.echofs.id
  display_name   = "echofs-sl"

  # Allow SSH
  ingress_security_rules {
    protocol = "6" # TCP
    source   = "0.0.0.0/0"
    tcp_options {
      min = 22
      max = 22
    }
  }

  # Allow HTTP
  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 80
      max = 80
    }
  }

  # Allow HTTPS
  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 443
      max = 443
    }
  }

  # Allow K8s API (for remote kubectl)
  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 6443
      max = 6443
    }
  }

  # Allow all egress
  egress_security_rules {
    protocol    = "all"
    destination = "0.0.0.0/0"
  }
}

# Compute Instance - ARM (Free Tier: 4 OCPU, 24GB RAM)
resource "oci_core_instance" "echofs" {
  compartment_id      = var.compartment_ocid
  availability_domain = data.oci_identity_availability_domains.ads.availability_domains[0].name
  display_name        = "echofs-k3s"
  shape               = "VM.Standard.A1.Flex"

  shape_config {
    ocpus         = 4
    memory_in_gbs = 24
  }

  source_details {
    source_type             = "image"
    source_id               = data.oci_core_images.ubuntu.images[0].id
    boot_volume_size_in_gbs = 100
  }

  create_vnic_details {
    subnet_id        = oci_core_subnet.public.id
    assign_public_ip = true
  }

  metadata = {
    ssh_authorized_keys = var.ssh_public_key
    user_data = base64encode(<<-EOF
      #!/bin/bash
      set -euo pipefail

      # Update system
      apt-get update && apt-get upgrade -y

      # Install k3s (lightweight Kubernetes)
      curl -sfL https://get.k3s.io | sh -s - \
        --disable traefik \
        --write-kubeconfig-mode 644 \
        --tls-san $(curl -s ifconfig.me)

      # Wait for k3s to be ready
      sleep 30
      kubectl wait --for=condition=Ready nodes --all --timeout=120s

      # Install Caddy ingress controller
      kubectl apply -f https://raw.githubusercontent.com/caddyserver/ingress/master/deploy/caddy-ingress-controller.yaml

      echo "k3s installation complete"
    EOF
    )
  }
}

# Block Volume for persistent storage (50GB free)
resource "oci_core_volume" "data" {
  compartment_id      = var.compartment_ocid
  availability_domain = data.oci_identity_availability_domains.ads.availability_domains[0].name
  display_name        = "echofs-data"
  size_in_gbs         = 50
}

resource "oci_core_volume_attachment" "data" {
  attachment_type = "paravirtualized"
  instance_id     = oci_core_instance.echofs.id
  volume_id       = oci_core_volume.data.id
}

# Outputs
output "instance_public_ip" {
  value = oci_core_instance.echofs.public_ip
}

output "ssh_command" {
  value = "ssh ubuntu@${oci_core_instance.echofs.public_ip}"
}

output "kubeconfig_command" {
  value = "scp ubuntu@${oci_core_instance.echofs.public_ip}:/etc/rancher/k3s/k3s.yaml ~/.kube/echofs.yaml && sed -i '' 's/127.0.0.1/${oci_core_instance.echofs.public_ip}/g' ~/.kube/echofs.yaml"
}
