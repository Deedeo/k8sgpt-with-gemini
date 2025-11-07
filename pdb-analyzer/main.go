package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	v1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Handler implements the analyzer interface
type Handler struct {
	rpc.CustomAnalyzerServiceServer
}

// Analyzer struct holds the handler
type Analyzer struct {
	Handler *Handler
}

// NewHandler creates a new analyzer handler
func NewHandler() *Handler {
	return &Handler{}
}

// Helper function to split a workload string like "Deployment 'namespace/name'" into namespace and workload parts
func splitNamespaceWorkload(input string) []string {
	// Extract the 'namespace/name' part from the string
	parts := strings.Split(input, "'")
	if len(parts) < 2 {
		return []string{}
	}
	
	// Split namespace and workload name
	namespaceParts := strings.Split(parts[1], "/")
	if len(namespaceParts) != 2 {
		return []string{}
	}
	
	// Return namespace, workload type, and name separately
	workloadType := strings.Split(input, " ")[0] // Get "Deployment" or "StatefulSet"
	return []string{namespaceParts[0], workloadType, namespaceParts[1]}
}

// Run is the implementation of the analyzer interface
func (a *Handler) Run(ctx context.Context, req *v1.RunRequest) (*v1.RunResponse, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	missingPDBs := []string{}

	// Get all namespaces
	namespaceList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, ns := range namespaceList.Items {
		namespace := ns.Name

		// Get deployments
		deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue // skip namespace if error occurs
		}

		// Get statefulsets
		statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		// Get pdbs
		pdbs, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		// Index PDBs by selector
		pdbMap := make(map[string]struct{})
		for _, pdb := range pdbs.Items {
			selector := pdb.Spec.Selector.String()
			pdbMap[selector] = struct{}{}
		}

		// Check deployments
		for _, deploy := range deployments.Items {
			selector := metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: deploy.Spec.Selector.MatchLabels})
			if _, exists := pdbMap[selector]; !exists {
				missingPDBs = append(missingPDBs, fmt.Sprintf("Deployment '%s/%s'", namespace, deploy.Name))
			}
		}

		// Check statefulsets
		for _, sts := range statefulsets.Items {
			selector := metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: sts.Spec.Selector.MatchLabels})
			if _, exists := pdbMap[selector]; !exists {
				missingPDBs = append(missingPDBs, fmt.Sprintf("StatefulSet '%s/%s'", namespace, sts.Name))
			}
		}
	}

	if len(missingPDBs) == 0 {
		return &v1.RunResponse{
			Result: &v1.Result{
				Name:    "pdb-analyzer",
				Details: "All Deployments and StatefulSets across all namespaces have matching PDBs.",
			},
		}, nil
	}
	
	// Group workloads by namespace for better organization
	namespaceMap := make(map[string][]string)
	for _, workload := range missingPDBs {
		parts := splitNamespaceWorkload(workload)
		if len(parts) == 3 {
			namespace := parts[0]
			kind := parts[1]
			name := parts[2]
			
			// Store workload info as "Kind Name" format
			namespaceMap[namespace] = append(namespaceMap[namespace], kind + " " + name)
		}
	}
	
	// Format the output in a human-readable way
	var formattedOutput strings.Builder
	formattedOutput.WriteString("Missing PodDisruptionBudgets detected for the following workloads:\n\n")
	
	// Get sorted list of namespaces for consistent output
	var namespaceNames []string
	for ns := range namespaceMap {
		namespaceNames = append(namespaceNames, ns)
	}
	sort.Strings(namespaceNames)
	
	// Build the human-readable output
	for _, ns := range namespaceNames {
		formattedOutput.WriteString(fmt.Sprintf("Namespace: %s\n", ns))
		
		// Add each workload to the human-readable output
		for _, workload := range namespaceMap[ns] {
			formattedOutput.WriteString(fmt.Sprintf("  - %s\n", workload))
		}
		formattedOutput.WriteString("\n")
	}
	
	// Add the recommendation
	formattedOutput.WriteString("\n=== RECOMMENDATION ===\n")
	formattedOutput.WriteString("Create PodDisruptionBudgets for these workloads to ensure high availability during voluntary disruptions.\n")
	formattedOutput.WriteString("\n=== HOW TO FIX ===\n")
	formattedOutput.WriteString("For each workload, create a PDB that matches the workload's selector.\n")
	formattedOutput.WriteString("Example for Deployment 'app' in namespace 'default':\n\n")
	formattedOutput.WriteString("```yaml\n")
	formattedOutput.WriteString("apiVersion: policy/v1\n")
	formattedOutput.WriteString("kind: PodDisruptionBudget\n")
	formattedOutput.WriteString("metadata:\n")
	formattedOutput.WriteString("  name: app-pdb\n")
	formattedOutput.WriteString("  namespace: default\n")
	formattedOutput.WriteString("spec:\n")
	formattedOutput.WriteString("  minAvailable: 1  # or use maxUnavailable\n")
	formattedOutput.WriteString("  selector:\n")
	formattedOutput.WriteString("    matchLabels:\n")
	formattedOutput.WriteString("      app: app-name  # must match your workload's selector\n")
	formattedOutput.WriteString("```\n\n")
	
	// Also include a summary of missing PDBs by namespace
	formattedOutput.WriteString("=== SUMMARY ===\n")
	formattedOutput.WriteString("Missing PodDisruptionBudgets by namespace:\n")
	for _, ns := range namespaceNames {
		formattedOutput.WriteString(fmt.Sprintf("  - %s: %d workloads\n", ns, len(namespaceMap[ns])))
	}

	return &v1.RunResponse{
		Result: &v1.Result{
			Name:    "pdb-analyzer",
			Details: "Missing PodDisruptionBudgets detected for some workloads.",
			Error: []*v1.ErrorDetail{{
				Text: formattedOutput.String(),
			}},
		},
	}, nil
}

func main() {
	var err error
	address := fmt.Sprintf(":%s", "8085")
	lis, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	
	// Initialize our analyzer
	aa := Analyzer{
		Handler: NewHandler(),
	}

	// Register the analyzer service
	rpc.RegisterCustomAnalyzerServiceServer(grpcServer, aa.Handler)
	
	fmt.Println("Starting PDB Analyzer server on port 8085!")
	if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
