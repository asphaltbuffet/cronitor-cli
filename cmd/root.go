package cmd

import (
	"fmt"
	"os"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sync"
	"net/http"
	"time"
	"log"
	"io/ioutil"
	"net/url"
)

var cfgFile string
var apiKey string
var verbose bool
var dev bool
var version string
var userAgent string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "cronitor",
	Short: "Command line tools for cronitor.io",
	Long: ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	version = "0.1.0"
	userAgent = fmt.Sprintf("CronitorAgent/%s", version)
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", cfgFile, "config file (default: .cronitor.json)")
	RootCmd.PersistentFlags().StringVarP(&apiKey,"api-key", "k", apiKey, "Cronitor API Key")
	RootCmd.PersistentFlags().StringVarP(&apiKey,"hostname", "n", apiKey, "A unique identifier for this host (default: system hostname)")
	RootCmd.PersistentFlags().BoolVarP(&verbose,"verbose", "v", verbose, "Verbose output")

	RootCmd.PersistentFlags().BoolVar(&dev,"use-dev",dev, "Dev mode")
	RootCmd.PersistentFlags().MarkHidden("use-dev")

	viper.BindPFlag("CRONITOR-API-KEY", RootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("CRONITOR-HOSTNAME", RootCmd.PersistentFlags().Lookup("hostname"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory
		viper.AddConfigPath(home)
		viper.SetConfigName(".cronitor")
	}

	viper.AutomaticEnv() // read in environment variables that match
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Println("Reading config from", viper.ConfigFileUsed())
	}
}

func sendPing(endpoint string, uniqueIdentifier string, message string, group *sync.WaitGroup) {
	Client := &http.Client{
		Timeout: time.Second * 3,
	}

	message = url.QueryEscape(message)

	for i:=1; i<=6; i++  {
		// Determine the ping API host. After a few failed attempts, try using cronitor.io instead
		var host string
		if dev {
			host = "http://dev.cronitor.io"
		} else if i > 2 && host == "https://cronitor.link" {
			host = "https://cronitor.io"
		} else {
			host = "https://cronitor.link"
		}

		uri := fmt.Sprintf("%s/%s/%s?try=%d&msg=%s", host, uniqueIdentifier, endpoint, i, message)

		if verbose {
			fmt.Println("Sending ping", uri)
		}

		request, err := http.NewRequest("GET", uri, nil)
		request.Header.Add("User-Agent", userAgent)
		response, err := Client.Do(request)

		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
			return
		}

		_, err = ioutil.ReadAll(response.Body)
		if err == nil && response.StatusCode < 400 {
			break
		}

		response.Body.Close()
	}

	group.Done()
}

func effectiveHostname() string {
	if len(viper.GetString("CRONITOR-HOSTNAME")) > 0 {
		return viper.GetString("CRONITOR-HOSTNAME")
	}

	hostname, _ := os.Hostname()
	return hostname
}
