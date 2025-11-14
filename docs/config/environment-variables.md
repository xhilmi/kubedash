# Environment Variables Reference

Kite dapat dikonfigurasi menggunakan berbagai environment variables. Berikut adalah daftar lengkap semua variable yang tersedia:

## 🔐 Security & Authentication

### `JWT_SECRET`
- **Deskripsi**: Secret key untuk signing JWT tokens
- **Default**: `kite-default-jwt-secret-key-change-in-production`
- **Wajib untuk production**: ✅ Ya
- **Contoh**: `JWT_SECRET=your-super-secret-key-here`
- **⚠️ Warning**: Harus diganti di production untuk keamanan!

### `KITE_ENCRYPT_KEY`
- **Deskripsi**: Key untuk enkripsi data sensitif (seperti kubeconfig di database)
- **Default**: `kite-default-encryption-key-change-in-production`
- **Wajib untuk production**: ✅ Ya
- **Contoh**: `KITE_ENCRYPT_KEY=your-encryption-key-here`
- **⚠️ Warning**: Harus diganti di production untuk keamanan!

### `SESSION_TIMEOUT_MINUTES`
- **Deskripsi**: Session inactivity timeout dalam menit
- **Default**: `60` (1 jam)
- **Contoh**: `SESSION_TIMEOUT_MINUTES=120` (2 jam)
- **Fitur**: Session akan expire setelah tidak ada aktivitas selama waktu yang ditentukan

## 👤 User Management

### `KITE_USERNAME`
- **Deskripsi**: Username untuk super user yang dibuat otomatis saat startup (jika database kosong)
- **Default**: tidak ada
- **Contoh**: `KITE_USERNAME=admin`
- **Catatan**: Hanya bekerja di startup pertama kali

### `KITE_PASSWORD`
- **Deskripsi**: Password untuk super user yang dibuat otomatis
- **Default**: tidak ada
- **Contoh**: `KITE_PASSWORD=admin123`
- **Catatan**: Harus digunakan bersamaan dengan `KITE_USERNAME`

### `ANONYMOUS_USER_ENABLED`
- **Deskripsi**: Enable akses anonymous tanpa login
- **Default**: `false`
- **Contoh**: `ANONYMOUS_USER_ENABLED=true`
- **⚠️ Warning**: Tidak aman untuk production!

## 🌐 Server Configuration

### `PORT`
- **Deskripsi**: Port untuk HTTP server
- **Default**: `8080`
- **Contoh**: `PORT=3000`

### `HOST`
- **Deskripsi**: Host binding address
- **Default**: `""` (bind ke semua interface)
- **Contoh**: `HOST=0.0.0.0`

### `KITE_BASE`
- **Deskripsi**: Base path untuk aplikasi (berguna jika di belakang reverse proxy dengan subpath)
- **Default**: `""`
- **Contoh**: `KITE_BASE=/kite` atau `KITE_BASE=/dashboard`
- **URL akan menjadi**: `http://domain.com/kite/...`

## 💾 Database Configuration

### `DB_TYPE`
- **Deskripsi**: Tipe database yang digunakan
- **Default**: `sqlite`
- **Pilihan**: `sqlite`, `mysql`, `postgres`
- **Contoh**: `DB_TYPE=postgres`

### `DB_DSN`
- **Deskripsi**: Database connection string (DSN - Data Source Name)
- **Default**: `dev.db` (untuk SQLite)
- **Contoh SQLite**: `DB_DSN=/data/kite.db`
- **Contoh PostgreSQL**: `DB_DSN=host=localhost user=kite password=secret dbname=kite port=5432 sslmode=disable`
- **Contoh MySQL**: `DB_DSN=user:password@tcp(localhost:3306)/kite?charset=utf8mb4&parseTime=True&loc=Local`

## ☸️ Kubernetes Configuration

### `KUBECONFIG`
- **Deskripsi**: Path ke kubeconfig file untuk import cluster otomatis
- **Default**: `~/.kube/config`
- **Contoh**: `KUBECONFIG=/etc/kite/kubeconfig.yaml`
- **Catatan**: Clusters akan di-import otomatis saat startup pertama kali

### `DISABLE_CACHE`
- **Deskripsi**: Disable Kubernetes client cache
- **Default**: `false`
- **Contoh**: `DISABLE_CACHE=true`
- **Catatan**: Berguna untuk debugging, tapi akan lebih lambat

## 🎛️ Helm Configuration

### `HELM_MAX_REVISIONS`
- **Deskripsi**: Jumlah maksimal revision history yang ditampilkan untuk Helm releases
- **Default**: `20`
- **Contoh**: `HELM_MAX_REVISIONS=50`
- **Range**: Minimal 1
- **Catatan**: Nilai lebih besar akan menampilkan lebih banyak history tapi bisa lebih lambat

## 🖥️ Terminal & Node Access

### `NODE_TERMINAL_IMAGE`
- **Deskripsi**: Docker image yang digunakan untuk node terminal pod
- **Default**: `busybox:latest`
- **Contoh**: `NODE_TERMINAL_IMAGE=alpine:latest`
- **Catatan**: Image harus memiliki shell (sh/bash)

## 🔧 Feature Flags

### `ENABLE_ANALYTICS`
- **Deskripsi**: Enable anonymous analytics
- **Default**: `false`
- **Contoh**: `ENABLE_ANALYTICS=true`

### `DISABLE_GZIP`
- **Deskripsi**: Disable GZIP compression untuk response
- **Default**: `true`
- **Contoh**: `DISABLE_GZIP=false`

### `DISABLE_VERSION_CHECK`
- **Deskripsi**: Disable automatic version check
- **Default**: `false`
- **Contoh**: `DISABLE_VERSION_CHECK=true`

## 📝 Complete Example Configuration

Berikut adalah contoh lengkap konfigurasi untuk production:

```bash
# Security (WAJIB diganti!)
JWT_SECRET=your-super-secret-jwt-key-minimum-32-characters
KITE_ENCRYPT_KEY=your-encryption-key-minimum-32-characters

# Session timeout (2 jam)
SESSION_TIMEOUT_MINUTES=120

# Server
PORT=8080
HOST=0.0.0.0

# Database PostgreSQL
DB_TYPE=postgres
DB_DSN=host=postgres.example.com user=kite password=secretpass dbname=kite_prod port=5432 sslmode=require

# Helm configuration
HELM_MAX_REVISIONS=30

# Optional: Base path jika di belakang reverse proxy
# KITE_BASE=/kite

# Optional: Custom node terminal image
# NODE_TERMINAL_IMAGE=alpine:3.18

# Optional: Initial super user (hanya untuk first setup)
# KITE_USERNAME=admin
# KITE_PASSWORD=change-this-password

# Feature flags
ENABLE_ANALYTICS=false
DISABLE_VERSION_CHECK=false
```

## 🐳 Docker Compose Example

```yaml
version: '3.8'
services:
  kite:
    image: xhilmi/kubedash:latest
    ports:
      - "8080:8080"
    environment:
      # Security
      JWT_SECRET: ${JWT_SECRET}
      KITE_ENCRYPT_KEY: ${KITE_ENCRYPT_KEY}
      
      # Session
      SESSION_TIMEOUT_MINUTES: 120
      
      # Database
      DB_TYPE: postgres
      DB_DSN: host=postgres user=kite password=${DB_PASSWORD} dbname=kite sslmode=disable
      
      # Helm
      HELM_MAX_REVISIONS: 30
      
      # Initial user
      KITE_USERNAME: admin
      KITE_PASSWORD: ${ADMIN_PASSWORD}
    depends_on:
      - postgres
    volumes:
      - ./kubeconfig:/root/.kube/config:ro
  
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: kite
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: kite
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

## 🎯 Kubernetes Helm Values Example

```yaml
# values.yaml
image:
  repository: xhilmi/kubedash
  tag: latest

env:
  # Security
  JWT_SECRET:
    valueFrom:
      secretKeyRef:
        name: kite-secrets
        key: jwt-secret
  
  KITE_ENCRYPT_KEY:
    valueFrom:
      secretKeyRef:
        name: kite-secrets
        key: encrypt-key
  
  # Session
  SESSION_TIMEOUT_MINUTES: "120"
  
  # Database
  DB_TYPE: "postgres"
  DB_DSN:
    valueFrom:
      secretKeyRef:
        name: kite-secrets
        key: db-dsn
  
  # Helm
  HELM_MAX_REVISIONS: "30"
  
  # Server
  PORT: "8080"
  HOST: "0.0.0.0"

service:
  type: ClusterIP
  port: 8080

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: kite.example.com
      paths:
        - path: /
          pathType: Prefix
```

## 🔒 Security Best Practices

1. **Selalu ganti** `JWT_SECRET` dan `KITE_ENCRYPT_KEY` di production
2. **Gunakan secrets management** untuk menyimpan credentials (Kubernetes Secrets, HashiCorp Vault, etc)
3. **Disable** `ANONYMOUS_USER_ENABLED` di production
4. **Set** session timeout sesuai kebutuhan keamanan organisasi Anda
5. **Gunakan** PostgreSQL atau MySQL untuk production (bukan SQLite)
6. **Enable** SSL/TLS untuk database connections (`sslmode=require`)

## 📊 Performance Tuning

### Untuk environment dengan banyak revisions:
```bash
# Tampilkan lebih banyak revision history
HELM_MAX_REVISIONS=100
```

### Untuk environment dengan cluster banyak:
```bash
# Disable cache jika ada masalah
DISABLE_CACHE=true
```

### Untuk environment dengan traffic tinggi:
```bash
# Enable GZIP compression
DISABLE_GZIP=false
```

## ℹ️ Catatan Tambahan

- Environment variables hanya dibaca saat aplikasi startup
- Jika mengubah env vars, restart aplikasi untuk apply changes
- Untuk production, gunakan external secrets management
- Semua default values dapat dilihat di `pkg/common/common.go`

## 🆘 Troubleshooting

**Q: Session timeout tidak bekerja?**
- A: Pastikan nilai dalam menit (integer), contoh: `SESSION_TIMEOUT_MINUTES=120`

**Q: HELM_MAX_REVISIONS tidak apply?**
- A: Restart aplikasi setelah mengubah env variable, dan pastikan nilai adalah integer positif

**Q: Database connection gagal?**
- A: Periksa format DSN sesuai dengan database type yang digunakan

**Q: JWT token invalid setelah restart?**
- A: Jika `JWT_SECRET` diganti, semua token lama akan invalid. User harus login ulang.
