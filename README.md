# K8sGPT with Gemini on GKE

This guide provides step-by-step instructions to deploy K8sGPT on a Google Kubernetes Engine (GKE) cluster and configure it to use Google's Gemini model for analysis. It also covers deploying a custom analyzer to check for missing PodDisruptionBudgets (PDBs).

## Prerequisites

- Google Cloud SDK (`gcloud`) installed and authenticated: `gcloud auth login`
- `kubectl` installed.
- `helm` installed (for the K8sGPT Operator).
- A Google Cloud project with the GKE API enabled.

---

## Step 1: Clone This Repository

Clone the repository to get all the necessary configuration files.

```bash
git clone https://github.com/Deedeo/k8sgpt-with-gemini.git
cd k8sgpt-with-gemini
```

---

## Step 2: Provision a GKE Cluster

Create a GKE cluster. This command uses cost-effective Spot VMs, which are suitable for development and testing environments.

```bash
# Pick a single zone within the region, e.g., us-central1-a
gcloud container clusters create devfest-ib --zone us-central1-a --num-nodes 3 --spot
```

After the cluster is created, configure `kubectl` to connect to it:

```bash
gcloud container clusters get-credentials devfest-location --region us-central1
```

---

## Step 3: Install the K8sGPT Operator

The K8sGPT Operator manages the K8sGPT resources within your cluster. We'll use Helm to install it.

1.  **Add the K8sGPT Helm repository:**
    ```bash
    helm repo add k8sgpt https://charts.k8sgpt.ai/
    helm repo update
    ```

2.  **Create a namespace for the operator:**
    ```bash
    kubectl create namespace k8sgpt-operator-system
    ```

3.  **Install the operator:**
    ```bash
    helm install release k8sgpt/k8sgpt-operator -n k8sgpt-operator-system
    ```

---

## Step 4: Configure Gemini API Key

K8sGPT needs a Google AI Studio API key to communicate with the Gemini model.

1.  **Generate an API Key:**
    Go to [Google AI Studio](https://aistudio.google.com/app/apikey) and create a new API key.

2.  **Create a Kubernetes Secret:**
    Use the generated API key to create a secret in your cluster. This secret will be securely accessed by K8sGPT.

    ```bash
    kubectl create secret generic k8sgpt-secret -n k8sgpt-operator-system \
      --from-literal=api-key=YOUR_API_KEY
    ```
    > **Note:** Replace `YOUR_API_KEY` with the actual key you generated.

---

## Step 5: Deploy K8sGPT with Gemini Configuration

Now, deploy the K8sGPT custom resource that configures it to use Gemini.

```bash
# The k8sgpt-gemini.yaml file is pre-configured for this setup
kubectl apply -f k8sgpt-gemini.yaml
```

This configuration tells K8sGPT to:
- Use the `gemini-2.0-flash` model.
- Use `google` as the AI backend.
- Find the API key in the `k8sgpt-secret` you created.

---

## Step 6: Analyze the Cluster and View Results

The K8sGPT operator will automatically start analyzing your cluster.

1.  **Check for the K8sGPT resource:**
    ```bash
    kubectl get k8sgpt -n k8sgpt-operator-system
    ```
    You should see the `gemini` resource listed.

2.  **View Analysis Results:**
    Results are stored as `Result` custom resources.
    ```bash
    # Wait a few minutes for the first analysis to complete
    kubectl get results -n k8sgpt-operator-system
    ```

3.  **View Details of a Specific Result:**
    To get an AI-powered explanation for an issue, view the details of a specific result object.
    ```bash
    # Replace <result-name> with a name from the list above
    kubectl get result <result-name> -n k8sgpt-operator-system -o yaml
    ```
    Look for the `spec.ai.details` field in the output for the explanation from Gemini.

### Alternative: Manual Analysis with K8sGPT CLI

You can also trigger analysis manually using the `k8sgpt` CLI. This is useful for on-demand checks.

1.  **Install the K8sGPT CLI:**
    Follow the official instructions to install the CLI for your operating system: [K8sGPT CLI Installation](https://docs.k8sgpt.ai/getting-started/installation/#cli)

2.  **Configure Authentication:**
    Use the `auth` command to add the Google backend, specify the model, and provide your API key.
    ```bash
    k8sgpt auth add --backend google --model gemini-2.0-flash
    ```
    When prompted, enter your Google AI Studio API key.

3.  **Run the analysis:**
    Now you can run the analysis command, specifying the backend. The model you configured during authentication will be used automatically.
    ```bash
    k8sgpt analyze --backend google
    ```

    This will stream the results directly to your terminal.

4.  **Get Explanations for Failures:**
    To get AI-powered explanations for any issues found, run the analysis with the `--explain` flag.
    ```bash
    k8sgpt analyze --backend google --explain
    ```

---

## (Optional) Step 7: Deploy the Custom PDB Analyzer

This repository also includes a custom analyzer that checks for missing PodDisruptionBudgets (PDBs).

1.  **Build and Push the Analyzer Image:**
    Navigate to the `pdb-analyzer` directory, build the Docker image, and push it to a container registry (like Docker Hub or GCR).
    ```bash
    cd pdb-analyzer
    docker build -t YOUR_REGISTRY/pdb-analyzer:latest .
    docker push YOUR_REGISTRY/pdb-analyzer:latest
    cd ..
    ```
    > **Note:** Replace `YOUR_REGISTRY` with your container registry's path.

2.  **Update the Deployment Manifest:**
    Open `pdb-analyzer-deployment.yaml` and change the `image` field to point to the image you just pushed.

3.  **Deploy the Custom Analyzer:**
    ```bash
    kubectl apply -f pdb-analyzer-deployment.yaml
    ```

4.  **Configure K8sGPT to Use the Custom Analyzer:**
    Apply the `k8sgpt-config.yaml` file, which tells K8sGPT about the custom analyzer service.
    ```bash
    kubectl apply -f k8sgpt-config.yaml
    ```

5.  **View Custom Analyzer Results:**
    The results from the PDB analyzer will now also appear in the `results` list.
    ```bash
    kubectl get results -n k8sgpt-operator-system | grep pdb
    ```

---

## Cleanup

To remove the resources created in this guide, run the following commands:

```bash
# Delete K8sGPT configurations
kubectl delete -f k8sgpt-gemini.yaml
kubectl delete -f k8sgpt-config.yaml

# Delete the custom analyzer
kubectl delete -f pdb-analyzer-deployment.yaml

# Uninstall the K8sGPT operator
helm uninstall release -n k8sgpt-operator-system
kubectl delete namespace k8sgpt-operator-system

# Delete the GKE cluster
gcloud container clusters delete devfest-location --region us-central1-a
```
