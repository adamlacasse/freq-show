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
	defaultMusicBrainzBase           = "https://musicbrainz.org/ws/2"
	defaultMusicBrainzApp            = "freq-show"
	defaultMusicBrainzVer            = "1.0"
	defaultMusicBrainzContact        = "adamlacasse@outlook.com"
	defaultMusicBrainzTimeoutSeconds = 6
	defaultMusicBrainzMinIntervalMS  = 1100
	defaultWikipediaBase             = "https://en.wikipedia.org/api/rest_v1"
	defaultWikipediaUserAgent        = "FreqShow/1.0 (https://github.com/adamlacasse/freq-show)"
	defaultWikipediaTimeoutSeconds   = 8
	defaultReviewsUserAgent          = "FreqShow/1.0 (https://github.com/adamlacasse/freq-show)"
	defaultReviewsTimeoutSeconds     = 10
	defaultDiscoveryEmbeddingsProv   = "voyage"
	defaultDiscoveryLLMProv          = "huggingface"

	// exampleDatabaseURL appears in the startup error when DATABASE_URL is
	// missing. There is deliberately no default for it: a relative path
	// silently creates an empty database in whatever directory the process
	// happens to start from, and migrate() then runs CREATE TABLE IF NOT
	// EXISTS against it, so the server looks perfectly healthy while serving
	// no data.
	exampleDatabaseURL = "file:/var/data/freqshow.db?_pragma=foreign_keys(1)"

	shutdownTimeoutEnv              = "SHUTDOWN_TIMEOUT_SECONDS"
	portEnv                         = "PORT"
	httpPortEnv                     = "HTTP_PORT"
	environmentEnv                  = "APP_ENV"
	databaseDriverEnv               = "DATABASE_DRIVER"
	databaseURLEnv                  = "DATABASE_URL"
	musicBrainzBaseURLEnv           = "MUSICBRAINZ_BASE_URL"
	musicBrainzTimeoutEnv           = "MUSICBRAINZ_TIMEOUT_SECONDS"
	musicBrainzMinIntervalEnv       = "MUSICBRAINZ_MIN_INTERVAL_MS"
	musicBrainzAppNameEnv           = "MUSICBRAINZ_APP_NAME"
	musicBrainzAppVersionEnv        = "MUSICBRAINZ_APP_VERSION"
	musicBrainzContactEnv           = "MUSICBRAINZ_CONTACT"
	wikipediaBaseURLEnv             = "WIKIPEDIA_BASE_URL"
	wikipediaTimeoutEnv             = "WIKIPEDIA_TIMEOUT_SECONDS"
	wikipediaUserAgentEnv           = "WIKIPEDIA_USER_AGENT"
	reviewsUserAgentEnv             = "REVIEWS_USER_AGENT"
	reviewsTimeoutEnv               = "REVIEWS_TIMEOUT_SECONDS"
	reviewsDiscogsTokenEnv          = "REVIEWS_DISCOGS_TOKEN"
	reviewsDiscogsConsumerKeyEnv    = "REVIEWS_DISCOGS_CONSUMER_KEY"
	reviewsDiscogsConsumerSecretEnv = "REVIEWS_DISCOGS_CONSUMER_SECRET"

	discoveryEmbeddingsProviderEnv = "DISCOVERY_EMBEDDINGS_PROVIDER"
	discoveryEmbeddingsAPIKeyEnv   = "DISCOVERY_EMBEDDINGS_API_KEY"
	discoveryEmbeddingsModelEnv    = "DISCOVERY_EMBEDDINGS_MODEL"
	discoveryEmbeddingsBaseURLEnv  = "DISCOVERY_EMBEDDINGS_BASE_URL"
	discoveryLLMProviderEnv        = "DISCOVERY_LLM_PROVIDER"
	discoveryLLMAPIKeyEnv          = "DISCOVERY_LLM_API_KEY"
	discoveryLLMModelEnv           = "DISCOVERY_LLM_MODEL"
)

// Config captures runtime configuration derived from environment variables.
type Config struct {
	Env             string
	Port            string
	ShutdownTimeout time.Duration
	MusicBrainz     MusicBrainzConfig
	Wikipedia       WikipediaConfig
	Reviews         ReviewsConfig
	Database        DatabaseConfig
	Discovery       DiscoveryConfig
}

// MusicBrainzConfig describes how the MusicBrainz client should connect.
type MusicBrainzConfig struct {
	BaseURL     string
	AppName     string
	AppVersion  string
	Contact     string
	Timeout     time.Duration
	MinInterval time.Duration
}

// WikipediaConfig describes how the Wikipedia client should connect.
type WikipediaConfig struct {
	BaseURL   string
	UserAgent string
	Timeout   time.Duration
}

// ReviewsConfig describes how the reviews client should connect.
type ReviewsConfig struct {
	UserAgent             string
	Timeout               time.Duration
	DiscogsToken          string
	DiscogsConsumerKey    string
	DiscogsConsumerSecret string
}

// DatabaseConfig describes how application persistence should be configured.
type DatabaseConfig struct {
	Driver string
	URL    string
}

// DiscoveryConfig describes how the AI music discovery pipeline should reach
// its hosted embedding and LLM providers. Empty API keys are tolerated at
// load time — the discovery service surfaces a clear "unconfigured" state
// at request time. This keeps the server bootable for development paths
// that don't exercise the discovery feature yet.
type DiscoveryConfig struct {
	EmbeddingsProvider string
	EmbeddingsAPIKey   string
	EmbeddingsModel    string
	EmbeddingsBaseURL  string
	LLMProvider        string
	LLMAPIKey          string
	LLMModel           string
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

	wikipedia, err := resolveWikipedia()
	if err != nil {
		return nil, err
	}

	reviews, err := resolveReviews()
	if err != nil {
		return nil, err
	}

	database, err := resolveDatabase()
	if err != nil {
		return nil, err
	}

	discovery := resolveDiscovery()

	env := strings.TrimSpace(envOrDefault(environmentEnv, defaultEnv))

	return &Config{
		Env:             env,
		Port:            port,
		ShutdownTimeout: shutdownTimeout,
		MusicBrainz:     musicBrainz,
		Wikipedia:       wikipedia,
		Reviews:         reviews,
		Database:        database,
		Discovery:       discovery,
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
		url, ok := lookupNonEmpty(databaseURLEnv)
		if !ok {
			return DatabaseConfig{}, fmt.Errorf(
				"%s is required when %s=sqlite (for example %s). Refusing to fall back to a relative path, which would create an empty database wherever the process starts",
				databaseURLEnv, databaseDriverEnv, exampleDatabaseURL,
			)
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

	minInterval := time.Duration(defaultMusicBrainzMinIntervalMS) * time.Millisecond
	if rawMinInterval, ok := lookupNonEmpty(musicBrainzMinIntervalEnv); ok {
		millis, err := strconv.Atoi(rawMinInterval)
		if err != nil {
			return MusicBrainzConfig{}, fmt.Errorf("invalid %s value %q: %w", musicBrainzMinIntervalEnv, rawMinInterval, err)
		}
		if millis > 0 {
			minInterval = time.Duration(millis) * time.Millisecond
		}
	}

	appName := envOrDefault(musicBrainzAppNameEnv, defaultMusicBrainzApp)
	appVersion := envOrDefault(musicBrainzAppVersionEnv, defaultMusicBrainzVer)
	contact := envOrDefault(musicBrainzContactEnv, defaultMusicBrainzContact)

	return MusicBrainzConfig{
		BaseURL:     strings.TrimRight(baseURL, "/"),
		AppName:     strings.TrimSpace(appName),
		AppVersion:  strings.TrimSpace(appVersion),
		Contact:     strings.TrimSpace(contact),
		Timeout:     timeout,
		MinInterval: minInterval,
	}, nil
}

func resolveWikipedia() (WikipediaConfig, error) {
	baseURL := envOrDefault(wikipediaBaseURLEnv, defaultWikipediaBase)
	userAgent := envOrDefault(wikipediaUserAgentEnv, defaultWikipediaUserAgent)
	timeout := time.Duration(defaultWikipediaTimeoutSeconds) * time.Second

	if rawTimeout, ok := lookupNonEmpty(wikipediaTimeoutEnv); ok {
		seconds, err := strconv.Atoi(rawTimeout)
		if err != nil {
			return WikipediaConfig{}, fmt.Errorf("invalid %s value %q: %w", wikipediaTimeoutEnv, rawTimeout, err)
		}
		if seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	return WikipediaConfig{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		UserAgent: strings.TrimSpace(userAgent),
		Timeout:   timeout,
	}, nil
}

func resolveReviews() (ReviewsConfig, error) {
	userAgent := envOrDefault(reviewsUserAgentEnv, defaultReviewsUserAgent)
	discogsToken := envOrDefault(reviewsDiscogsTokenEnv, "")
	discogsConsumerKey := envOrDefault(reviewsDiscogsConsumerKeyEnv, "")
	discogsConsumerSecret := envOrDefault(reviewsDiscogsConsumerSecretEnv, "")
	timeout := time.Duration(defaultReviewsTimeoutSeconds) * time.Second

	if rawTimeout, ok := lookupNonEmpty(reviewsTimeoutEnv); ok {
		seconds, err := strconv.Atoi(rawTimeout)
		if err != nil {
			return ReviewsConfig{}, fmt.Errorf("invalid %s value %q: %w", reviewsTimeoutEnv, rawTimeout, err)
		}
		if seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	return ReviewsConfig{
		UserAgent:             strings.TrimSpace(userAgent),
		DiscogsToken:          strings.TrimSpace(discogsToken),
		DiscogsConsumerKey:    strings.TrimSpace(discogsConsumerKey),
		DiscogsConsumerSecret: strings.TrimSpace(discogsConsumerSecret),
		Timeout:               timeout,
	}, nil
}

func resolveDiscovery() DiscoveryConfig {
	embedProvider := strings.ToLower(strings.TrimSpace(envOrDefault(discoveryEmbeddingsProviderEnv, defaultDiscoveryEmbeddingsProv)))
	if embedProvider == "" {
		embedProvider = defaultDiscoveryEmbeddingsProv
	}
	llmProvider := strings.ToLower(strings.TrimSpace(envOrDefault(discoveryLLMProviderEnv, defaultDiscoveryLLMProv)))
	if llmProvider == "" {
		llmProvider = defaultDiscoveryLLMProv
	}

	return DiscoveryConfig{
		EmbeddingsProvider: embedProvider,
		EmbeddingsAPIKey:   strings.TrimSpace(envOrDefault(discoveryEmbeddingsAPIKeyEnv, "")),
		EmbeddingsModel:    strings.TrimSpace(envOrDefault(discoveryEmbeddingsModelEnv, "")),
		EmbeddingsBaseURL:  strings.TrimRight(strings.TrimSpace(envOrDefault(discoveryEmbeddingsBaseURLEnv, "")), "/"),
		LLMProvider:        llmProvider,
		LLMAPIKey:          strings.TrimSpace(envOrDefault(discoveryLLMAPIKeyEnv, "")),
		LLMModel:           strings.TrimSpace(envOrDefault(discoveryLLMModelEnv, "")),
	}
}
