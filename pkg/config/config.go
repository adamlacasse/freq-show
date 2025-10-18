package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort                      = "8080"
	defaultEnv                       = "development"
	defaultShutdownSeconds           = 10
	defaultDatabaseDriver            = "sqlite"
	defaultDatabaseURL               = "file:freqshow.db?_fk=1"
	defaultMusicBrainzBase           = "https://musicbrainz.org/ws/2"
	defaultMusicBrainzApp            = "freq-show"
	defaultMusicBrainzVer            = "dev"
	defaultMusicBrainzContact        = "dev@localhost"
	defaultMusicBrainzTimeoutSeconds = 6

	shutdownTimeoutEnv       = "SHUTDOWN_TIMEOUT_SECONDS"
	portEnv                  = "PORT"
	httpPortEnv              = "HTTP_PORT"
	environmentEnv           = "APP_ENV"
	databaseDriverEnv        = "DATABASE_DRIVER"
	databaseURLEnv           = "DATABASE_URL"
	musicBrainzBaseURLEnv    = "MUSICBRAINZ_BASE_URL"
	musicBrainzTimeoutEnv    = "MUSICBRAINZ_TIMEOUT_SECONDS"
	musicBrainzAppNameEnv    = "MUSICBRAINZ_APP_NAME"
	musicBrainzAppVersionEnv = "MUSICBRAINZ_APP_VERSION"
	musicBrainzContactEnv    = "MUSICBRAINZ_CONTACT"
)

// Config captures runtime configuration derived from environment variables.
type Config struct {
	Env             string
	Port            string
	ShutdownTimeout time.Duration
	MusicBrainz     MusicBrainzConfig
	Database        DatabaseConfig
}

// MusicBrainzConfig describes how the MusicBrainz client should connect.
type MusicBrainzConfig struct {
	BaseURL    string
	AppName    string
	AppVersion string
	Contact    string
	Timeout    time.Duration
}

// DatabaseConfig describes how application persistence should be configured.
type DatabaseConfig struct {
	Driver string
	URL    string
}

// Load reads environment variables and assembles a Config instance.
func Load() (*Config, error) {
	port, err := resolvePort()
	if err != nil {
		return nil, err
	}

	shutdownTimeout, err := resolveShutdownTimeout()
	if err != nil {
		return nil, err
	}

	musicBrainz, err := resolveMusicBrainz()
	if err != nil {
		return nil, err
	}

	database, err := resolveDatabase()
	if err != nil {
		return nil, err
	}

	env := strings.TrimSpace(envOrDefault(environmentEnv, defaultEnv))

	return &Config{
		Env:             env,
		Port:            port,
		ShutdownTimeout: shutdownTimeout,
		MusicBrainz:     musicBrainz,
		Database:        database,
	}, nil
}

// Address returns the value to assign to net/http.Server.Addr.
func (c *Config) Address() string {
	if strings.Contains(c.Port, ":") {
		return c.Port
	}
	return ":" + c.Port
}

func envOrDefault(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return val
	}
	return fallback
}

func resolvePort() (string, error) {
	for _, key := range []string{portEnv, httpPortEnv} {
		if val, ok := lookupNonEmpty(key); ok {
			return normalizePort(val)
		}
	}
	return normalizePort(defaultPort)
}

func resolveShutdownTimeout() (time.Duration, error) {
	val, ok := lookupNonEmpty(shutdownTimeoutEnv)
	if !ok {
		return time.Duration(defaultShutdownSeconds) * time.Second, nil
	}

	seconds, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q: %w", shutdownTimeoutEnv, val, err)
	}
	if seconds <= 0 {
		seconds = defaultShutdownSeconds
	}
	return time.Duration(seconds) * time.Second, nil
}

func normalizePort(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("port value cannot be empty")
	}

	if strings.Contains(trimmed, ":") {
		host, port, found := strings.Cut(trimmed, ":")
		if !found || port == "" {
			return "", fmt.Errorf("invalid port value %q", raw)
		}
		port = strings.TrimSpace(port)
		if _, err := strconv.Atoi(port); err != nil {
			return "", fmt.Errorf("invalid port value %q: %w", raw, err)
		}
		host = strings.TrimSpace(host)
		if host == "" {
			return ":" + port, nil
		}
		return host + ":" + port, nil
	}

	if _, err := strconv.Atoi(trimmed); err != nil {
		return "", fmt.Errorf("invalid port value %q: %w", raw, err)
	}
	return trimmed, nil
}

func lookupNonEmpty(key string) (string, bool) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func resolveDatabase() (DatabaseConfig, error) {
	driver := strings.TrimSpace(envOrDefault(databaseDriverEnv, defaultDatabaseDriver))
	if driver == "" {
		driver = defaultDatabaseDriver
	}
	driver = strings.ToLower(driver)

	switch driver {
	case "sqlite":
		url := strings.TrimSpace(envOrDefault(databaseURLEnv, defaultDatabaseURL))
		if url == "" {
			return DatabaseConfig{}, fmt.Errorf("database url required for sqlite driver")
		}
		return DatabaseConfig{Driver: driver, URL: url}, nil
	case "memory":
		return DatabaseConfig{Driver: driver, URL: ""}, nil
	default:
		return DatabaseConfig{}, fmt.Errorf("unsupported database driver %q", driver)
	}
}

func resolveMusicBrainz() (MusicBrainzConfig, error) {
	baseURL := envOrDefault(musicBrainzBaseURLEnv, defaultMusicBrainzBase)
	timeout := time.Duration(defaultMusicBrainzTimeoutSeconds) * time.Second
	if rawTimeout, ok := lookupNonEmpty(musicBrainzTimeoutEnv); ok {
		seconds, err := strconv.Atoi(rawTimeout)
		if err != nil {
			return MusicBrainzConfig{}, fmt.Errorf("invalid %s value %q: %w", musicBrainzTimeoutEnv, rawTimeout, err)
		}
		if seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	appName := envOrDefault(musicBrainzAppNameEnv, defaultMusicBrainzApp)
	appVersion := envOrDefault(musicBrainzAppVersionEnv, defaultMusicBrainzVer)
	contact := envOrDefault(musicBrainzContactEnv, defaultMusicBrainzContact)

	return MusicBrainzConfig{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		AppName:    strings.TrimSpace(appName),
		AppVersion: strings.TrimSpace(appVersion),
		Contact:    strings.TrimSpace(contact),
		Timeout:    timeout,
	}, nil
}
