terraform {
  required_version = ">= 1.6"
  required_providers { kubernetes = { source = "hashicorp/kubernetes", version = ">= 2.30" } }
}
variable "namespace" { type = string, default = "sentinelflow" }
variable "image" { type = string, default = "sentinelflow:5.0.0" }
provider "kubernetes" {}
resource "kubernetes_namespace" "sentinelflow" { metadata { name = var.namespace } }
resource "kubernetes_deployment" "gateway" {
  metadata { name = "sentinelflow-gateway"; namespace = kubernetes_namespace.sentinelflow.metadata[0].name }
  spec { replicas = 2 selector { match_labels = { app = "sentinelflow-gateway" } }
    template { metadata { labels = { app = "sentinelflow-gateway" } }
      spec { container { name = "gateway"; image = var.image; port { container_port = 8080 } } }
    }
  }
}
resource "kubernetes_service" "gateway" {
  metadata { name = "sentinelflow-gateway"; namespace = kubernetes_namespace.sentinelflow.metadata[0].name }
  spec { selector = { app = "sentinelflow-gateway" } port { port = 8080; target_port = 8080 } }
}
