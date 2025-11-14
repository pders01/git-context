package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "context",
	Short: "Git-native temporal snapshot workflow for research and code",
	Long: `context creates immutable research snapshots that capture:
  - exact codebase state
  - research artifacts (notes, POCs, analyses)
  - optional metadata
  - optional vector embeddings for agentic search

This eliminates documentation drift, preserves rationale, and supports
both human developers and agentic tools.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/context/config.toml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		configDir := filepath.Join(home, ".config", "context")
		viper.AddConfigPath(configDir)
		viper.SetConfigType("toml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("retention.days", 90)
	viper.SetDefault("retention.preserve_tags", []string{"important"})
	viper.SetDefault("snapshot.default_mode", "full")
	viper.SetDefault("snapshot.research_dir", "research")
	viper.SetDefault("embeddings.enabled", true)
	viper.SetDefault("embeddings.model", "nomic-embed-text")
	viper.SetDefault("embeddings.ollama_url", "http://localhost:11434")
	viper.SetDefault("search.keyword_weight", 0.3)
	viper.SetDefault("search.semantic_weight", 0.7)

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
