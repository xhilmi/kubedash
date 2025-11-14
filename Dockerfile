FROM node:20-alpine AS frontend-builder

WORKDIR /app/ui

COPY ui/package.json ui/pnpm-lock.yaml ./

RUN npm install -g pnpm && \
    pnpm install --frozen-lockfile

COPY ui/ ./
RUN pnpm run build

FROM golang:1.24-alpine AS backend-builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

COPY --from=frontend-builder /app/static ./static
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o kite .

FROM alpine:latest AS tools

# Install helm and kubectl CLI with latest versions
# Helm 3.19.1 (latest, 0 CVEs)
# kubectl 1.34.1 (latest stable, has 10 stdlib CVEs from Go 1.24.6)
# 
# Note on kubectl vulnerabilities:
# - All 10 CVEs are in Go stdlib (not kubectl code itself)
# - These are low-risk for CLI tool usage (not network-exposed service)
# - Will be auto-resolved when Kubernetes upstream rebuilds with Go 1.25.2+
# - Current vulnerabilities: CVE-2025-47912, CVE-2025-58183, CVE-2025-58186,
#   CVE-2025-58187, CVE-2025-58188, CVE-2025-61724, CVE-2025-58185, 
#   CVE-2025-58189, CVE-2025-61723, CVE-2025-61725
RUN apk add --no-cache curl bash openssl && \
    # Install Helm 3.19.1
    curl -fsSL -o helm.tar.gz https://get.helm.sh/helm-v3.19.1-linux-amd64.tar.gz && \
    tar -zxvf helm.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    rm -rf helm.tar.gz linux-amd64 && \
    # Install kubectl 1.34.1
    curl -LO "https://dl.k8s.io/release/v1.34.1/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && mv kubectl /usr/local/bin/

FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy binaries from tools stage
COPY --from=tools /usr/local/bin/helm /usr/local/bin/helm
COPY --from=tools /usr/local/bin/kubectl /usr/local/bin/kubectl

COPY --from=backend-builder /app/kite .

EXPOSE 8080

# Create non-root user
RUN addgroup -g 1000 kite && \
    adduser -D -u 1000 -G kite kite && \
    chown -R kite:kite /app

USER kite

CMD ["./kite"]
