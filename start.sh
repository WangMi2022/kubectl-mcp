docker build -t kubectl-mcp:latest .

docker rm -f kubectl-mcp
docker run -d --name=kubectl-mcp -v ./config.yaml:/app/config/config.yaml -v ./kubeconfig.yaml:/app/config/kubeconfig.yaml -p 8000:8080 kubectl-mcp:latest 
