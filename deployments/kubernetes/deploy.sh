#!/bin/bash

# Colores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}🚀 Desplegando SentinelFlow en Kubernetes...${NC}"

# Construir imagen Docker
echo -e "${YELLOW}📦 Construyendo imagen...${NC}"
docker build -f Dockerfile.kubernetes -t sentinelflow:latest .

# Aplicar ConfigMap y Secrets
echo -e "${YELLOW}📋 Aplicando ConfigMap...${NC}"
kubectl apply -f deployments/kubernetes/configmap.yaml

# Aplicar Secrets
echo -e "${YELLOW}🔐 Aplicando Secrets...${NC}"
kubectl apply -f deployments/kubernetes/secrets.yaml

# Aplicar Deployment y Service
echo -e "${YELLOW}📦 Desplegando Deployment...${NC}"
kubectl apply -f deployments/kubernetes/deployment.yaml
kubectl apply -f deployments/kubernetes/service.yaml

# Aplicar HPA
echo -e "${YELLOW}📊 Configurando autoscaling...${NC}"
kubectl apply -f deployments/kubernetes/hpa.yaml

# Esperar que esté listo
echo -e "${YELLOW}⏳ Esperando que los pods estén listos...${NC}"
kubectl rollout status deployment/sentinelflow

# Ver estado
echo -e "${GREEN}✅ ¡Despliegue completado!${NC}"
echo ""
echo -e "📋 Estado del deployment:"
kubectl get pods -l app=sentinelflow
echo ""
echo -e "📋 Servicios:"
kubectl get svc sentinelflow-service
echo ""
echo -e "📋 HPA:"
kubectl get hpa sentinelflow-hpa
