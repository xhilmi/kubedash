# Docker Build & Development Guide

This guide provides commands for building, testing, and developing Kubedash locally.

## 📋 Prerequisites

- Docker installed and running
- Go 1.24+ for backend development
- Node.js 20+ and pnpm for frontend development

---

## 🔧 Environment Variables

Create a `.env` file or export these variables before running commands:

```bash
# Application Configuration
export APPS_PORT=8080

# Docker Configuration
export PLATFORM=linux/amd64        # or linux/arm64 for ARM
export DOCKER_NETW=kubedash_default
export DOCKER_PATH=Dockerfile
export DOCKER_REPO=kubedash        # Your Docker repository name
export DOCKER_USER=<your-dockerhub-username>  # ⚠️ CHANGE THIS
export DOCKER_VERS=2.4.6           # Current version
export DOCKER_VERS_OLD=2.4.5       # Previous version (for cleanup)
export DOCKER_LAST=latest
export FOLDER=.

# Example with actual values:
# export DOCKER_USER=mycompany
# This will create: mycompany/kubedash:2.4.6
```

**⚠️ Important**: Replace `<your-dockerhub-username>` with your actual Docker Hub username or registry prefix.

---

## 🐳 Docker Commands

### Build Docker Image

Build for your platform (auto-detects):
```bash
docker build -f $DOCKER_PATH $FOLDER --no-cache \
    -t $DOCKER_USER/$DOCKER_REPO:$DOCKER_VERS \
    -t $DOCKER_USER/$DOCKER_REPO:$DOCKER_LAST
```

Build for specific platform (e.g., ARM64):
```bash
docker build --platform $PLATFORM \
    -f $DOCKER_PATH $FOLDER --no-cache \
    -t $DOCKER_USER/$DOCKER_REPO:$DOCKER_VERS \
    -t $DOCKER_USER/$DOCKER_REPO:$DOCKER_LAST
```

Build for multiple platforms (requires buildx):
```bash
docker buildx build --platform linux/amd64,linux/arm64 \
    -f $DOCKER_PATH $FOLDER \
    -t $DOCKER_USER/$DOCKER_REPO:$DOCKER_VERS \
    -t $DOCKER_USER/$DOCKER_REPO:$DOCKER_LAST \
    --push
```

### Push to Registry

```bash
# Push versioned tag
docker push $DOCKER_USER/$DOCKER_REPO:$DOCKER_VERS

# Push latest tag
docker push $DOCKER_USER/$DOCKER_REPO:$DOCKER_LAST

# Push both at once
docker push $DOCKER_USER/$DOCKER_REPO:$DOCKER_VERS && \
docker push $DOCKER_USER/$DOCKER_REPO:$DOCKER_LAST
```

### Run Docker Container

Run with environment file:
```bash
docker run --name $DOCKER_REPO \
    -p $APPS_PORT:$APPS_PORT \
    --rm \
    --env-file .env \
    --network=$DOCKER_NETW \
    $DOCKER_USER/$DOCKER_REPO:$DOCKER_VERS
```

Run with latest tag:
```bash
docker run --name $DOCKER_REPO \
    -p $APPS_PORT:$APPS_PORT \
    --rm \
    --env-file .env \
    --network=$DOCKER_NETW \
    $DOCKER_USER/$DOCKER_REPO:$DOCKER_LAST
```

Run with inline environment variables:
```bash
docker run --name $DOCKER_REPO \
    -p $APPS_PORT:$APPS_PORT \
    --rm \
    -e JWT_SECRET="your-jwt-secret" \
    -e KITE_ENCRYPT_KEY="your-32-byte-key" \
    -e KITE_USERNAME="admin" \
    -e KITE_PASSWORD="password" \
    -v ~/.kube/config:/root/.kube/config:ro \
    $DOCKER_USER/$DOCKER_REPO:$DOCKER_VERS
```

### Cleanup

Remove old image version:
```bash
docker rmi $DOCKER_USER/$DOCKER_REPO:$DOCKER_VERS_OLD
```

Clean up build cache:
```bash
docker builder prune -f
```

Remove all unused images:
```bash
docker image prune -a -f
```

---

## 🏗️ Local Development

### Backend Development

#### Prerequisites
```bash
# Install Go 1.24+
go version

# Install dependencies
go mod download
```

#### Build Backend
```bash
cd /path/to/kite
go mod download
go build -o kite .
```

#### Run Backend (Development Mode)
```bash
# With hot reload (using air)
go install github.com/air-verse/air@latest
air

# Or run directly
go run main.go
```

#### Build with Custom Flags
```bash
# Build with version info
VERSION=$(git describe --tags --always)
go build -ldflags "-X main.Version=$VERSION" -o kite .
```

### Frontend Development

#### Prerequisites
```bash
# Install Node.js 20+ and pnpm
node --version
npm install -g pnpm
```

#### Install Dependencies
```bash
cd ui
pnpm install
```

#### Build Frontend (Production)
```bash
cd ui
pnpm run build
```

#### Run Frontend (Development Mode with Hot Reload)
```bash
cd ui
pnpm run dev
```
This starts Vite dev server at `http://localhost:5173` with hot module replacement.

#### Type Checking
```bash
cd ui
pnpm run type-check
```

#### Linting
```bash
cd ui
pnpm run lint
pnpm run lint:fix  # Auto-fix issues
```

#### Format Code
```bash
cd ui
pnpm run format
```

---

## 🧪 Testing

### Test Backend
```bash
go test ./...
```

### Test Specific Package
```bash
go test ./pkg/handlers/...
```

### Test with Coverage
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Frontend
```bash
cd ui
pnpm run test
```

---

## 🔍 Debugging

### Debug Backend with Delve
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Run with debugger
dlv debug main.go
```

### Debug Docker Container
```bash
# Run container with shell access
docker run -it --rm \
    --entrypoint /bin/sh \
    $DOCKER_USER/$DOCKER_REPO:$DOCKER_VERS

# Check container logs
docker logs $DOCKER_REPO

# Inspect running container
docker exec -it $DOCKER_REPO /bin/sh
```

---

## 📦 Multi-Stage Build Process

The Dockerfile uses multi-stage builds:

1. **frontend-builder** - Builds React/TypeScript UI
   ```dockerfile
   FROM node:20-alpine AS frontend-builder
   WORKDIR /app/ui
   COPY ui/package*.json ui/pnpm-lock.yaml ./
   RUN npm install -g pnpm && pnpm install
   COPY ui/ ./
   RUN pnpm run build
   ```

2. **backend-builder** - Builds Go binary
   ```dockerfile
   FROM golang:1.24-alpine AS backend-builder
   WORKDIR /app
   COPY go.mod go.sum ./
   RUN go mod download
   COPY . .
   RUN CGO_ENABLED=0 GOOS=linux go build -o kite .
   ```

3. **tools** - Downloads kubectl and helm
   ```dockerfile
   FROM alpine:latest AS tools
   RUN apk add --no-cache curl
   # Download kubectl
   # Download helm
   ```

4. **final stage** - Minimal Alpine image with binaries
   ```dockerfile
   FROM alpine:latest
   COPY --from=backend-builder /app/kite /kite
   COPY --from=frontend-builder /app/ui/dist /ui/dist
   COPY --from=tools /usr/local/bin/kubectl /usr/local/bin/kubectl
   COPY --from=tools /usr/local/bin/helm /usr/local/bin/helm
   ```

---

## 🎯 Common Workflows

### Full Build & Push Workflow
```bash
# 1. Set environment variables
export DOCKER_USER=mycompany
export DOCKER_VERS=2.4.6

# 2. Build image
docker build --no-cache -t $DOCKER_USER/kubedash:$DOCKER_VERS -t $DOCKER_USER/kubedash:latest .

# 3. Test locally
docker run --rm -p 8080:8080 --env-file .env $DOCKER_USER/kubedash:$DOCKER_VERS

# 4. Push to registry
docker push $DOCKER_USER/kubedash:$DOCKER_VERS
docker push $DOCKER_USER/kubedash:latest

# 5. Cleanup
docker builder prune -f
```

### Quick Development Cycle
```bash
# Terminal 1 - Backend
cd /path/to/kite
go run main.go

# Terminal 2 - Frontend
cd ui
pnpm run dev
```

Then access:
- Backend API: `http://localhost:8080`
- Frontend: `http://localhost:5173` (proxies to backend)

---

## 📝 Example .env File

```bash
# Security (REQUIRED)
JWT_SECRET=your-super-secret-jwt-key-change-me
KITE_ENCRYPT_KEY=your-32-byte-encryption-key-now

# Initial Admin (Optional)
KITE_USERNAME=admin
KITE_PASSWORD=ChangeMe123!

# Database
DB_TYPE=sqlite
DB_DSN=file:/data/kite.db

# Network
PORT=8080
HOST=http://localhost:8080

# Features
ANONYMOUS_USER_ENABLED=false
ENABLE_ANALYTICS=false
DEBUG=true
```

---

## 🚀 Production Best Practices

1. **Use specific version tags** - Don't rely on `latest` in production
2. **Multi-arch builds** - Build for both amd64 and arm64
3. **Security scanning** - Run `docker scan $IMAGE` before deploying
4. **Image signing** - Use Docker Content Trust or Cosign
5. **Minimal base image** - Already using Alpine (5MB base)
6. **Non-root user** - Dockerfile runs as non-root by default
7. **Health checks** - Add HEALTHCHECK in Dockerfile if needed

---

## 🔗 Related Documentation

- [Environment Variables](../docs/config/env.md) - Complete ENV variable reference
- [Helm Chart Values](../charts/kite/values.yaml) - Kubernetes deployment config
- [Installation Guide](../docs/guide/installation.md) - Production deployment guide

---

**Last Updated**: November 5, 2025
**Version**: 2.4.6 (Kubedash Fork)