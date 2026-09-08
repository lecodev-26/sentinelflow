terraform {
  required_version = ">= 1.0"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.9"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5"
    }
  }
}

provider "kubernetes" {
  config_path = var.kubeconfig_path
}

provider "helm" {
  kubernetes {
    config_path = var.kubeconfig_path
  }
}

# Variables
variable "kubeconfig_path" {
  description = "Path to kubeconfig file"
  default     = "~/.kube/config"
}

variable "namespace" {
  description = "Kubernetes namespace"
  default     = "sentinelflow"
}

variable "openai_api_key" {
  description = "OpenAI API Key"
  sensitive   = true
  default     = ""
}

variable "anthropic_api_key" {
  description = "Anthropic API Key"
  sensitive   = true
  default     = ""
}

# Namespace
resource "kubernetes_namespace" "sentinelflow" {
  metadata {
    name = var.namespace
  }
}

# Helm Release
resource "helm_release" "sentinelflow" {
  name       = "sentinelflow"
  namespace  = kubernetes_namespace.sentinelflow.metadata[0].name
  chart      = "../helm/sentinelflow"
  wait       = true
  timeout    = 300

  values = [
    <<-EOT
    secrets:
      openai:
        apiKey: "${var.openai_api_key}"
      anthropic:
        apiKey: "${var.anthropic_api_key}"
    EOT
  ]

  set {
    name  = "image.repository"
    value = var.image_repository
  }

  set {
    name  = "image.tag"
    value = var.image_tag
  }

  set {
    name  = "replicaCount"
    value = var.replica_count
  }
}

variable "image_repository" {
  description = "Docker image repository"
  default     = "sentinelflow"
}

variable "image_tag" {
  description = "Docker image tag"
  default     = "latest"
}

variable "replica_count" {
  description = "Number of replicas"
  default     = 3
}

output "namespace" {
  value = kubernetes_namespace.sentinelflow.metadata[0].name
}

output "release_name" {
  value = helm_release.sentinelflow.name
}
