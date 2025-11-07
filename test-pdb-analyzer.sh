#!/bin/bash

# Script to test PDB analyzer deployment

# Check if the analyzer service is running
echo "Checking if PDB analyzer service is running..."
kubectl get pods -n k8sgpt-system -l app=pdb-analyzer

# Test the analyzer with grpcurl
echo "Testing PDB analyzer with grpcurl..."
echo "Make sure you have grpcurl installed: https://github.com/fullstorydev/grpcurl"
echo ""
echo "Port-forward the service to localhost:"
echo "kubectl port-forward service/pdb-analyzer -n k8sgpt-system 8085:8085"
echo ""
echo "Then in another terminal, run:"
echo "grpcurl --plaintext localhost:8085 schema.v1.AnalyzerService/Run"
echo ""

# List K8sGPT resources
echo "Checking K8sGPT resources..."
kubectl get k8sgpts -n k8sgpt-operator-system

# Get scan results
echo "Retrieving scan results (if available)..."
echo "kubectl get results -n k8sgpt-operator-system"
echo ""
echo "To view detailed results for a specific scan:"
echo "kubectl get results <result-name> -n k8sgpt-operator-system -o json"