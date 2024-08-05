package main

import (
	"betwiz"
	"betwiz/http"
	"betwiz/postgres"
	"betwiz/scrpr"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"

	_ "net/http/pprof"

	"github.com/pelletier/go-toml/v2"
)


// Entry point for the web application
// Sets up and runs an HTTP server with services like database connections and web scraping, using configuration settings loaded from a file.
// Handling for graceful shutdown and command-line flag parsing to manage runtime configurations.

var(
	version string
	commit string
)

func main(){
	// Set the version and commit variables from the global values
	betwiz.Version = strings.TrimPrefix(version, "")
	betwiz.Commit = commit

	// Setup signal handlers to allow for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt) // Listen for CTRL + C
	go func() { <-c; cancel() }()  // Cancel the context when an interrupt signal is received

	// Create a new instance of Main
	m := NewMain()

	// Parse command-line flags and handle errors
	if err := m.ParseFlags(ctx, os.Args[1:]); err == flag.ErrHelp {
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Run the main application logic and handle errors
	if err := m.Run(ctx); err != nil {
		m.Close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Wait for CTRL + C to exit
	<- ctx.Done()

	// Clean up resources before exiting
	if err := m.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Main struct holds configuration and services used by the application
type Main struct {
	Config      Config      // Parsed config data
	ConfigPath  string      // Path to the config file

	DB          *postgres.DB
	HTTPServer  *http.Server
}

// NewMain initializes a new instance of Main with default values
func NewMain() *Main {
	return &Main{
		Config:      DefaultConfig(),
		ConfigPath:  DefaultConfigPath,

		DB:          postgres.NewDB(""),
		HTTPServer:  http.NewServer(),
	}
}

// Close shuts down the HTTP server and database connection
func (m *Main) Close() error {
	if m.HTTPServer != nil {
		if err := m.HTTPServer.Close(); err != nil {
			return err
		}
	}

	if m.DB != nil {
		if err := m.DB.Close(); err != nil {  
			return err
		}
	}

	return nil
}

// ParseFlags parses command-line flags and loads the configuration file
func (m *Main) ParseFlags(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("betwiz", flag.ContinueOnError)
	fs.StringVar(&m.ConfigPath, "config", DefaultConfigPath, "config path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Expand the config path to handle '~' for the home directory
	configPath, err := expand(m.ConfigPath)
	if err != nil {
		return err
	}

	// Read and parse the config file
	config, err := ReadConfigFile(configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", m.ConfigPath)
	} else if err != nil {
		return err
	}
	m.Config = config

	return nil
}

// Run executes the main application logic
func (m *Main) Run(ctx context.Context) (err error) {
	// Expand the DSN (Data Source Name) for the database connection
	if m.DB.DSN, err = expandDSN(m.Config.DB.DSN); err != nil {
		return fmt.Errorf("cannot expand dsn: %w", err)
	}

	// Open the database connection
	if err := m.DB.Open(); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}

	// Instantiate services and attach them to the HTTP server
	scrprService := postgres.NewScrprService(m.DB)
	m.HTTPServer.ScraperController = *scrpr.NewController(2, scrprService)

	// Copy config settings to the HTTP server
	m.HTTPServer.Addr = m.Config.HTTP.Addr
	m.HTTPServer.Domain = m.Config.HTTP.Domain
	m.HTTPServer.HashKey = m.Config.HTTP.HashKey
	m.HTTPServer.BlockKey = m.Config.HTTP.BlockKey

	// Attach services to the HTTP server
	m.HTTPServer.ScrprService = scrprService // Fixed typo: was m.HTTPServer.sScrprService

	// Start the HTTP server
	if err := m.HTTPServer.Open(); err != nil {
		return err
	}

	// If TLS is enabled, redirect non-TLS connections to TLS
	if m.HTTPServer.UseTLS() {
		go func() {
			log.Fatal(http.ListenAndServeTLSRedirect(m.Config.HTTP.Domain))
		}()
	}

	// Run all scrapers in the background
	go m.HTTPServer.ScraperController.RunAll(ctx)

	// Start the debug HTTP server for performance profiling
	go func() { http.ListenAndServeDebug() }()

	// Log the running server details
	log.Printf("running: url=%q debug=http://localhost:6060 dsn=%q", m.HTTPServer.URL(), m.Config.DB.DSN)

	return nil
}

// Default paths and configuration values
const (
	DefaultConfigPath = "./betwiz.conf"
	DefaultDSN        = "~/.betwiz/db"
)

// Config struct holds the application's configuration settings
type Config struct {
	DB struct {
		DSN string `toml:"dsn"`
	} `toml:"db"`

	HTTP struct {
		Addr     string `toml:"addr"`
		Domain   string `toml:"domain"`
		HashKey  string `toml:"hash-key"`
		BlockKey string `toml:"block-key"`
	} `toml:"http"`
}

// DefaultConfig returns the default configuration values
func DefaultConfig() Config {
	var config Config
	config.DB.DSN = DefaultDSN  // Fixed typo: was DNS instead of DSN

	return config
}

// ReadConfigFile reads and parses the configuration file
func ReadConfigFile(filename string) (Config, error) {
	config := DefaultConfig()
	if buf, err := os.ReadFile(filename); err != nil {
		return config, err
	} else if err := toml.Unmarshal(buf, &config); err != nil {
		return config, err
	}

	return config, nil
}

// expand expands a path containing '~' to the user's home directory
func expand(path string) (string, error) {
	if path != "-" && !strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
		return path, nil
	}

	u, err := user.Current()
	if err != nil {
		return path, err
	} else if u.HomeDir == "" {
		return path, fmt.Errorf("home directory unset")
	}

	if path == "~" {
		return u.HomeDir, nil
	}

	return filepath.Join(u.HomeDir, strings.TrimPrefix(path, "~"+string(os.PathSeparator))), nil
}

// expandDSN expands the DSN path to handle '~' for the home directory
func expandDSN(dsn string) (string, error) {
	return expand(dsn)
}