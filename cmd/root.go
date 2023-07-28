////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/noob-channel/noobChannel"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/utils"
	"io"
	"log"
	"os"
	"os/signal"
)

var configFilePath string

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "noob-channel-bot",
	Short: "Runs a bot to manage noob channels",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(configFilePath)
		initConfig(configFilePath)
		initLog(viper.GetString(logPathFlag), viper.GetUint(logLevelFlag))

		ndfPath := viper.GetString(ndfPathFlag)
		ndfBytes, err := utils.ReadFile(ndfPath)
		if err != nil {
			jww.FATAL.Panicf("Failed to read NDF from %s: %+v", ndfPath, err)
		}

		adminKeysDir := viper.GetString(adminKeysPathFlag)
		if !utils.DirExists(adminKeysDir) {
			err = os.Mkdir(adminKeysDir, os.ModePerm)
			if err != nil {
				jww.FATAL.Panicf("Failed to create admin key directory at %s: %+v", adminKeysDir, err)
			}
		}

		storageDir := viper.GetString(clientStorageFlag)
		password := viper.GetString(clientPasswordFlag)
		contactOutput := viper.GetString(contactPathFlag)
		ncm, err := noobChannel.Init(string(ndfBytes), storageDir, contactOutput, []byte(password), adminKeysDir, fastRNG.NewStreamGenerator(8, 8, csprng.NewSystemRNG))
		if err != nil {
			jww.FATAL.Panicf("Failed to start noob channel manager: %+v", err)
		}
		jww.INFO.Printf("Running %s", ncm.Name())
		// Wait for kill signal
		sigChan := make(chan os.Signal)
		signal.Notify(sigChan, os.Interrupt)
		<-sigChan
	},
}

// init is the initialization function for Cobra which defines commands
// and flags.
func init() {
	// NOTE: The point of init() is to be declarative.
	// There is one init in each sub command. Do not put variable declarations
	// here, and ensure all the Flags are of the *P variety, unless there's a
	// very good reason not to have them as local Params to sub command."

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVarP(&configFilePath, "config", "c", "",
		"File path to Custom configuration.")

	rootCmd.PersistentFlags().UintP(logLevelFlag, "l", 0,
		"Level of debugging to print (0 = info, 1 = debug, >1 = trace).")
	bindPFlag(rootCmd.PersistentFlags(), logLevelFlag, rootCmd.Use)

	rootCmd.PersistentFlags().StringP(logPathFlag, "", "log/noobChannel.log",
		"Path where log file will be saved.")
	bindPFlag(rootCmd.PersistentFlags(), logPathFlag, rootCmd.Use)

	rootCmd.PersistentFlags().StringP(ndfPathFlag, "", "cmix/ndf.log",
		"Path where ndf file can be found.")
	bindPFlag(rootCmd.PersistentFlags(), ndfPathFlag, rootCmd.Use)

	rootCmd.PersistentFlags().StringP(contactPathFlag, "", "cmix/ncContact.json",
		"Cmix client identity file output.")
	bindPFlag(rootCmd.PersistentFlags(), contactPathFlag, rootCmd.Use)

	rootCmd.PersistentFlags().StringP(clientStorageFlag, "", "cmix/clientBlob",
		"Cmix client storage directory.")
	bindPFlag(rootCmd.PersistentFlags(), clientStorageFlag, rootCmd.Use)

	rootCmd.PersistentFlags().StringP(clientPasswordFlag, "", "",
		"Password for cmix client storage.")
	bindPFlag(rootCmd.PersistentFlags(), clientPasswordFlag, rootCmd.Use)

	rootCmd.PersistentFlags().StringP(adminKeysPathFlag, "", "cmix/adminKeys",
		"Path where admin keys will be stored on channel generation.")
	bindPFlag(rootCmd.PersistentFlags(), adminKeysPathFlag, rootCmd.Use)
}

// bindPFlag binds the key to a pflag.Flag. Panics on error.
func bindPFlag(flagSet *pflag.FlagSet, key, use string) {
	err := viper.BindPFlag(key, flagSet.Lookup(key))
	if err != nil {
		jww.FATAL.Panicf(
			"Failed to bind key %q to a pflag on %s: %+v", key, use, err)
	}
}

// initConfig reads in config file from the file path.
func initConfig(filePath string) {
	// Use default config location if none is passed
	if filePath == "" {
		return
	}

	filePath, err := utils.ExpandPath(filePath)
	if err != nil {
		jww.FATAL.Panicf("Invalid config file path %q: %+v", filePath, err)
	}

	viper.SetConfigFile(filePath)

	viper.AutomaticEnv() // Read in environment variables that match

	// If a config file is found, read it in.
	if err = viper.ReadInConfig(); err != nil {
		jww.FATAL.Panicf("Invalid config file path %q: %+v", filePath, err)
	}
}

// initLog initialises the log to the specified log path filtered to the
// threshold. If the log path is "-" or "", it is printed to stdout.
func initLog(logPath string, threshold uint) {
	if logPath != "-" && logPath != "" {
		// Disable stdout output
		jww.SetStdoutOutput(io.Discard)

		// Use log file
		logOutput, err :=
			os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		jww.SetLogOutput(logOutput)
	}

	if threshold > 1 {
		jww.SetStdoutThreshold(jww.LevelTrace)
		jww.SetLogThreshold(jww.LevelTrace)
		jww.SetFlags(log.LstdFlags | log.Lmicroseconds)
		jww.INFO.Printf("Log level set to TRACE and output to %s", logPath)
	} else if threshold == 1 {
		jww.SetStdoutThreshold(jww.LevelDebug)
		jww.SetLogThreshold(jww.LevelDebug)
		jww.SetFlags(log.LstdFlags | log.Lmicroseconds)
		jww.INFO.Printf("Log level set to DEBUG and output to %s", logPath)
	} else {
		jww.SetStdoutThreshold(jww.LevelInfo)
		jww.SetLogThreshold(jww.LevelInfo)
		jww.INFO.Printf("Log level set to INFO and output to %s", logPath)
	}
}
