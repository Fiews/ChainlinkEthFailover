package main

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	if err := generateCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func generateCmd() *cobra.Command {
	v := viper.New()
	newcmd := &cobra.Command{
		Use:  "cl-eth-failover [endpoints] [flags]",
		Args: cobra.MinimumNArgs(1),
		Long: "Failover from ETH nodes going offline and reduce Chainlink node downtime.",
		Run:  func(_ *cobra.Command, args []string) { runCallback(v, args, startService) },
	}

	newcmd.Flags().String("strategy", string(PrimaryInstant), "The reconnect strategy to use")
	if err := v.BindPFlag("strategy", newcmd.Flags().Lookup("strategy")); err != nil {
		panic(err)
	}

	newcmd.Flags().Int("max-attempts", 3, "Maximum failed attempts before connecting to next endpoint (primary* strategies)")
	if err := v.BindPFlag("max-attempts", newcmd.Flags().Lookup("max-attempts")); err != nil {
		panic(err)
	}

	newcmd.Flags().Int("reconnect-timeout", 3, "Timeout before reattempting to connect to primary endpoint (primary-async strategy)")
	if err := v.BindPFlag("reconnect-timeout", newcmd.Flags().Lookup("reconnect-timeout")); err != nil {
		panic(err)
	}

	newcmd.Flags().Int("port", 4000, "Port to start WS server on")
	if err := v.BindPFlag("port", newcmd.Flags().Lookup("port")); err != nil {
		panic(err)
	}

	newcmd.Flags().Int("blockheader-timeout", 300, "Number of seconds without a block header notification before disconnecting")
	if err := v.BindPFlag("blockheader-timeout", newcmd.Flags().Lookup("blockheader-timeout")); err != nil {
		panic(err)
	}

	return newcmd
}

var requiredConfig = []string{
	"strategy",
	"max-attempts",
	"reconnect-timeout",
	"port",
	"blockheader-timeout",
}

// runner type matches the function signature of synchronizeForever
type runner = func(Config, []string)

type Config struct {
	Strategy         Strategy
	MaxAttempts      int
	ReconnectTimeout time.Duration
	Port             int
	HeaderTimeout    time.Duration
}

func runCallback(v *viper.Viper, args []string, runner runner) {
	if err := validateParams(v, args, requiredConfig); err != nil {
		fmt.Println(err)
		return
	}

	config := Config{
		Strategy:         Strategy(v.GetString("strategy")),
		MaxAttempts:      v.GetInt("max-attempts"),
		ReconnectTimeout: time.Duration(v.GetInt("reconnect-timeout")) * time.Second,
		Port:             v.GetInt("port"),
		HeaderTimeout:    time.Duration(v.GetInt("blockheader-timeout")) * time.Second,
	}

	runner(config, args)
}

func validateParams(v *viper.Viper, args []string, required []string) error {
	var missing []string
	for _, k := range required {
		if v.GetString(k) == "" {
			msg := fmt.Sprintf("%s flag must be set", k)
			fmt.Println(msg)
			missing = append(missing, msg)
		}
	}
	if len(missing) > 0 {
		return errors.New(strings.Join(missing, ","))
	}

	for _, a := range args {
		u, err := url.Parse(a)
		if err != nil || !strings.HasPrefix(u.Scheme, "ws") {
			msg := fmt.Sprintf("Invalid URL provided: %v", a)
			fmt.Println(msg)
			return errors.Wrap(err, msg)
		}
	}

	return nil
}
