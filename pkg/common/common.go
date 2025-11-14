package common

import (
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

const (
	JWTExpirationSeconds = 24 * 60 * 60 // 24 hours

	NodeTerminalPodName = "kite-node-terminal-agent"

	KubectlAnnotation = "kubectl.kubernetes.io/last-applied-configuration"

	// db connection max idle time
	DBMaxIdleTime  = 10 * time.Minute
	DBMaxOpenConns = 100
	DBMaxIdleConns = 10

	// Default session inactivity timeout (1 hour)
	DefaultSessionTimeoutMinutes = 60

	// Default helm max revisions to fetch
	DefaultHelmMaxRevisions = 20
)

var (
	Port            = "8080"
	JwtSecret       = "kite-default-jwt-secret-key-change-in-production"
	EnableAnalytics = false
	Host            = ""
	Base            = ""

	NodeTerminalImage = "busybox:latest"
	DBType            = "sqlite"
	DBDSN             = "dev.db"

	KiteEncryptKey = "kite-default-encryption-key-change-in-production"

	AnonymousUserEnabled = false

	CookieExpirationSeconds = 2 * JWTExpirationSeconds // double jwt

	DisableGZIP         = true
	DisableVersionCheck = false

	APIKeyProvider = "api_key"

	// Session inactivity timeout in minutes (configurable via SESSION_TIMEOUT_MINUTES env)
	SessionTimeoutMinutes = DefaultSessionTimeoutMinutes

	// Helm max revisions to fetch (configurable via HELM_MAX_REVISIONS env)
	HelmMaxRevisions = DefaultHelmMaxRevisions
)

func LoadEnvs() {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		JwtSecret = secret
	}

	if port := os.Getenv("PORT"); port != "" {
		Port = port
	}

	if analytics := os.Getenv("ENABLE_ANALYTICS"); analytics == "true" {
		EnableAnalytics = true
	}

	if nodeTerminalImage := os.Getenv("NODE_TERMINAL_IMAGE"); nodeTerminalImage != "" {
		NodeTerminalImage = nodeTerminalImage
	}

	if dbDSN := os.Getenv("DB_DSN"); dbDSN != "" {
		DBDSN = dbDSN
	}

	if dbType := os.Getenv("DB_TYPE"); dbType != "" {
		if dbType != "sqlite" && dbType != "mysql" && dbType != "postgres" {
			klog.Fatalf("Invalid DB_TYPE: %s, must be one of sqlite, mysql, postgres", dbType)
		}
		DBType = dbType
	}

	if key := os.Getenv("KITE_ENCRYPT_KEY"); key != "" {
		KiteEncryptKey = key
	} else {
		klog.Warningf("KITE_ENCRYPT_KEY is not set, using default key, this is not secure for production!")
	}

	if v := os.Getenv("ANONYMOUS_USER_ENABLED"); v == "true" {
		AnonymousUserEnabled = true
		klog.Warningf("Anonymous user is enabled, this is not secure for production!")
	}
	if v := os.Getenv("HOST"); v != "" {
		Host = v
	}
	if v := os.Getenv("DISABLE_GZIP"); v != "" {
		DisableGZIP = v == "true"
	}

	if v := os.Getenv("DISABLE_VERSION_CHECK"); v == "true" {
		DisableVersionCheck = true
	}

	if v := os.Getenv("KITE_BASE"); v != "" {
		if v[0] != '/' {
			v = "/" + v
		}
		Base = strings.TrimRight(v, "/")
		klog.Infof("Using base path: %s", Base)
	}

	if v := os.Getenv("SESSION_TIMEOUT_MINUTES"); v != "" {
		if timeout, err := time.ParseDuration(v + "m"); err == nil {
			SessionTimeoutMinutes = int(timeout.Minutes())
			klog.Infof("Session timeout set to %d minutes", SessionTimeoutMinutes)
		} else {
			klog.Warningf("Invalid SESSION_TIMEOUT_MINUTES value: %s, using default %d minutes", v, DefaultSessionTimeoutMinutes)
		}
	}

	if v := os.Getenv("HELM_MAX_REVISIONS"); v != "" {
		if maxRevisions, err := strconv.Atoi(v); err == nil && maxRevisions > 0 {
			HelmMaxRevisions = maxRevisions
			klog.Infof("Helm max revisions set to %d", HelmMaxRevisions)
		} else {
			klog.Warningf("Invalid HELM_MAX_REVISIONS value: %s, using default %d", v, DefaultHelmMaxRevisions)
		}
	}
}
