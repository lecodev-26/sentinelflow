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

variable "kubeconfig_path" {
  description = "Path to kubeconfig file"
  default     = "~/.kube/config"
}

variable "namespace" {
  description = "Kubernetes namespace"
  default     = "sentinelflow"
}
