# Environment Variables

Kubedash supports several environment variables to configure the application behavior. These can be set via:
- Docker: `docker run --env-file .env ...`
- Kubernetes: ConfigMap or Secret
- Helm Chart: `values.yaml` (recommended)

## 🔐 Required Security Variables

### JWT_SECRET
- **Description**: Secret key used for signing and verifying JWT tokens
- **Required**: Yes
- **Default**: `kite-default-jwt-secret-key-change-in-production` (⚠️ **CHANGE IN PRODUCTION!**)
- **Example**: `your-super-secret-jwt-key-min-32-chars`
- **Security**: Use a strong, random string (minimum 32 characters)
- **Generation**: `openssl rand -base64 32`

### KITE_ENCRYPT_KEY
- **Description**: Secret key used for encrypting sensitive data (passwords, OAuth clientSecret, kubeconfig)
- **Required**: Yes
- **Default**: `kite-default-encryption-key-change-in-production` (⚠️ **CHANGE IN PRODUCTION!**)
- **Example**: `your-encryption-key-must-be-32-bytes`
- **Security**: Must be exactly 32 bytes for AES-256 encryption
- **Generation**: `openssl rand -base64 32 | cut -c1-32`

---

## 👤 Initial Admin User

### KITE_USERNAME
- **Description**: Initial administrator username (created on first startup)
- **Required**: No (can be created via UI initialization page)
- **Default**: None (manual setup required)
- **Example**: `admin`
- **Note**: Only used during first initialization, ignored on subsequent startups

### KITE_PASSWORD
- **Description**: Initial administrator password
- **Required**: No (can be created via UI initialization page)
- **Default**: None (manual setup required)
- **Example**: `MySecureP@ssw0rd123`
- **Note**: Only used during first initialization, ignored on subsequent startups

---

## 🗄️ Database Configuration

### DB_TYPE
- **Description**: Database type for persistent storage
- **Required**: No
- **Default**: `sqlite`
- **Supported Values**:
  - `sqlite` - Embedded database (default, good for testing/small deployments)
  - `postgres` - PostgreSQL (recommended for production)
  - `mysql` - MySQL/MariaDB
- **Example**: `postgres`

### DB_DSN
- **Description**: Database connection string (Data Source Name)
- **Required**: Yes (if using postgres/mysql), No (if using sqlite)
- **Format**:
  - **PostgreSQL**: `postgres://username:password@host:port/database?sslmode=disable`
  - **MySQL**: `username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True`
  - **SQLite**: `file:/path/to/kite.db` (or leave empty for default)
- **Examples**:
  ```bash
  # PostgreSQL
  DB_DSN="postgres://kite_user:mypassword@postgres.example.com:5432/kite_db?sslmode=require"
  
  # MySQL
  DB_DSN="kite_user:mypassword@tcp(mysql.example.com:3306)/kite_db?charset=utf8mb4&parseTime=True"
  
  # SQLite (default)
  DB_DSN="file:/data/kite.db"  # or leave empty
  ```

---

## ☸️ Kubernetes Configuration

### KUBECONFIG
- **Description**: Path to Kubernetes configuration file
- **Required**: No
- **Default**: `~/.kube/config`
- **Example**: `/config/kubeconfig.yaml`
- **Note**: 
  - Used for multi-cluster management
  - When no clusters configured, Kubedash auto-discovers from this file
  - Can import clusters via UI initialization page
  - Not needed when running in-cluster (uses ServiceAccount)

---

## 🌐 Network & Access

### PORT
- **Description**: HTTP port Kubedash listens on
- **Required**: No
- **Default**: `8080`
- **Example**: `3000`
- **Note**: Make sure to update service/ingress if changed

### HOST
- **Description**: External hostname/URL for OAuth redirect URIs
- **Required**: No (auto-detected from request headers)
- **Default**: Auto-detected from `Host` header
- **Example**: `https://kubedash.example.com`
- **When to Use**: 
  - Behind reverse proxy with different external URL
  - OAuth callback URL not matching request headers
  - Custom domain setup

### BASE_PATH
- **Description**: Base URL path when serving under a subpath
- **Required**: No
- **Default**: `/` (root)
- **Example**: `/kubedash`
- **Note**: 
  - Used when deploying under a subpath (e.g., `example.com/kubedash`)
  - Update ingress path to match

---

## 🔓 Access Control

### ANONYMOUS_USER_ENABLED
- **Description**: Enable anonymous access without authentication
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `false`
- **⚠️ Security Warning**: 
  - When `true`, **ALL users get full admin access** without login
  - **NEVER enable in production** unless behind strong network security
  - Only use for local development/testing

---

## 🖥️ Terminal Configuration

### NODE_TERMINAL_IMAGE
- **Description**: Docker image for Node Terminal Agent (exec into nodes)
- **Required**: No
- **Default**: `docker.io/xhilmi/node-agent:latest`

Example:

```bash
NODE_TERMINAL_IMAGE=docker.io/xhilmi/node-agent:latest
- **Example**: `custom-registry/node-agent:v1.0.0`
- **Note**: 
  - Used for web terminal access to Kubernetes nodes
  - Must have bash/sh shell available
  - Requires privileged access and hostPID=true

---

## 📊 Analytics & Telemetry

### ENABLE_ANALYTICS
- **Description**: Enable anonymous usage analytics
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `false`
- **Note**: 
  - Collects anonymous usage data to improve product
  - No sensitive data collected (no cluster names, resource names, etc.)
  - Opt-in only (disabled by default)

---

## 🐛 Development & Debugging

### DEBUG
- **Description**: Enable debug logging
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `true`
- **Note**: 
  - Enables verbose logging for troubleshooting
  - **Not recommended for production** (performance impact)

---

## 📝 Complete Example

### Docker Compose
```yaml
version: '3.8'
services:
  kubedash:
    image: xhilmi/kite:latest
    ports:
      - "8080:8080"
    environment:
      # Security (REQUIRED - CHANGE IN PRODUCTION!)
      JWT_SECRET: "your-super-secret-jwt-key-change-me"
      KITE_ENCRYPT_KEY: "your-32-byte-encryption-key-now"
      
      # Initial Admin (Optional)
      KITE_USERNAME: "admin"
      KITE_PASSWORD: "ChangeMe123!"
      
      # Database (PostgreSQL Example)
      DB_TYPE: "postgres"
      DB_DSN: "postgres://kite:password@postgres:5432/kite?sslmode=disable"
      
      # Network
      PORT: "8080"
      HOST: "https://kubedash.example.com"
      
      # Access Control
      ANONYMOUS_USER_ENABLED: "false"
      
      # Optional
      ENABLE_ANALYTICS: "false"
      DEBUG: "false"
    volumes:
      - ~/.kube/config:/root/.kube/config:ro
```

### Kubernetes Secret (Helm)
```yaml
# Create secret
kubectl create secret generic kubedash-secrets \
  --from-literal=JWT_SECRET='your-jwt-secret-here' \
  --from-literal=KITE_ENCRYPT_KEY='your-32-byte-key-here' \
  --from-literal=DB_DSN='postgres://user:pass@host:5432/db' \
  -n kube-system

# Use in Helm
helm install kubedash kite/kite \
  --set secret.create=false \
  --set secret.existingSecret=kubedash-secrets \
  --set db.type=postgres \
  -n kube-system
```

### Environment File (.env)
```bash
# Security - REQUIRED (CHANGE THESE!)
JWT_SECRET=your-super-secret-jwt-key-min-32-chars
KITE_ENCRYPT_KEY=your-encryption-key-exactly-32bytes

# Initial Admin - Optional (can setup via UI)
KITE_USERNAME=admin
KITE_PASSWORD=MySecurePassword123!

# Database - PostgreSQL
DB_TYPE=postgres
DB_DSN=postgres://kite_user:mypass@localhost:5432/kite_db?sslmode=disable

# Network
PORT=8080
HOST=https://kubedash.company.com

# Access Control
ANONYMOUS_USER_ENABLED=false

# Optional Features
ENABLE_ANALYTICS=false
DEBUG=false
NODE_TERMINAL_IMAGE=docker.io/xhilmi/node-agent:latest
```

---

## 🔧 Helm Chart Values Mapping

If using Helm, configure via `values.yaml` instead of ENV variables:

| Environment Variable | Helm Chart Path | Notes |
|---------------------|-----------------|-------|
| `JWT_SECRET` | `jwtSecret` or `secret.existingSecret` | Use existingSecret in prod |
| `KITE_ENCRYPT_KEY` | `encryptKey` or `secret.existingSecret` | Use existingSecret in prod |
| `KITE_USERNAME` | `superUser.username` | Only if `superUser.create=true` |
| `KITE_PASSWORD` | `superUser.password` | Only if `superUser.create=true` |
| `DB_TYPE` | `db.type` | sqlite/postgres/mysql |
| `DB_DSN` | `db.dsn` or `secret.existingSecret` | Use existingSecret in prod |
| `PORT` | `service.port` | Default 8080 |
| `HOST` | `host` | For OAuth callbacks |
| `BASE_PATH` | `basePath` | For subpath deployment |
| `ANONYMOUS_USER_ENABLED` | `anonymousUserEnabled` | **Never true in prod!** |
| `DEBUG` | `debug` | Development only |

See [Helm Chart Configuration](./chart-values.md) for complete Helm values documentation.

---

## ⚠️ Security Best Practices

1. **Always change default secrets in production**:
   - Generate strong JWT_SECRET: `openssl rand -base64 32`
   - Generate 32-byte KITE_ENCRYPT_KEY: `openssl rand -base64 32 | cut -c1-32`

2. **Use Kubernetes Secrets for sensitive data**:
   - Never commit secrets to Git
   - Use `secret.existingSecret` in Helm
   - Use sealed-secrets or external-secrets in production

3. **Never enable ANONYMOUS_USER_ENABLED in production**:
   - Only for local development/testing
   - Grants full admin access to everyone

4. **Use PostgreSQL/MySQL for production**:
   - SQLite is for testing/single-node only
   - Enable persistence if using SQLite

5. **Secure database connections**:
   - Use SSL/TLS (`sslmode=require` for PostgreSQL)
   - Strong database passwords
   - Network policies to restrict DB access

---

**Last Updated**: November 5, 2025
**Version**: 2.4.6 (Kubedash Fork)
