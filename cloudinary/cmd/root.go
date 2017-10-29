// Copyright © 2017 Jimmy Song <rootsongjc@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	cloudinary "github.com/rootsongjc/cloudinary-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var optVerbose bool
var optSimulate bool
var optPath string
var optImg string
var optRaw string
var service *cloudinary.Service
var settings = &Config{}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "cloudinary",
	Short: "A CLI tool to upload static assets to the Cloudinary service.",
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cloudinary.toml)")
	RootCmd.PersistentFlags().StringVarP(&optPath, "path", "p", "", "flle prepend path")
	RootCmd.PersistentFlags().StringVarP(&optImg, "image", "i", "", "image filename or public id")
	RootCmd.PersistentFlags().StringVarP(&optRaw, "raw", "r", "", "raw filename or public id")
	optSimulate = *RootCmd.PersistentFlags().BoolP("simulate", "s", false, "simulate, do nothing (dry run)")
	optVerbose = *RootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".cloudinary") // name of config file (without extension)
	viper.AddConfigPath("$HOME")       // adding home directory as first search path
	viper.AutomaticEnv()               // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
	var err error
	settings, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.Arg(1), err.Error())
		os.Exit(1)
	}
	service, err = cloudinary.Dial(settings.CloudinaryURI.String())
	service.Verbose(optVerbose)
	service.Simulate(optSimulate)
	service.KeepFiles(settings.KeepFilesPattern)
	if settings.MongoURI != nil {
		if err := service.UseDatabase(settings.MongoURI.String()); err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to mongoDB: %s\n", err.Error())
			os.Exit(1)
		}
	}

	if err != nil {
		fail(err.Error())
	}

	if optSimulate {
		fmt.Println("*** DRY RUN MODE ***")
	}
	if len(settings.PrependPath) > 0 {
		fmt.Println("Default remote prepend path set to: ", settings.PrependPath)
	} else {
		fmt.Println("No default remote prepend path set")
	}
}

// Config for cloudinary
type Config struct {
	// Url to the Cloudinary service.
	CloudinaryURI *url.URL
	// Url to a MongoDB instance, used to track files and upload
	// only changed. Optional.
	MongoURI *url.URL
	// Regexp pattern to prevent remote file deletion.
	KeepFilesPattern string
	// An optional remote prepend path, used to generate a unique
	// data path to a remote resource. This can be useful if public
	// ids are not random (i.e provided as request arguments) to solve
	// any caching issue: a different prepend path generates a new path
	// to the remote resource.
	PrependPath string
	// ProdTag is an alias to PrependPath. If PrependPath is empty but
	// ProdTag is set (with at prodtag= line in the [global] section of
	// the config file), PrependPath is set to ProdTag. For example, it
	// can be used with a DVCS commit tag to force new remote data paths
	// to remote resources.
	ProdTag string
}

// LoadConfig parses a config file and sets global settings
// variables to be used at runtime. Note that returning an error
// will cause the application to exit with code error 1.
func LoadConfig() (*Config, error) {
	// Cloudinary settings
	var cURI *url.URL
	var uri string
	var err error

	if uri = viper.GetString("cloudinary.uri"); err != nil {
		return nil, err
	}
	if cURI, err = url.Parse(uri); err != nil {
		return nil, errors.New(fmt.Sprint("cloudinary URI: ", err.Error()))
	}
	settings.CloudinaryURI = cURI

	// An optional remote prepend path
	prepend := viper.GetString("cloudinary.prepend")
	settings.PrependPath = cloudinary.EnsureTrailingSlash(prepend)
	settings.ProdTag = viper.GetString("global.prodtag")

	// Keep files regexp? (optional)
	var pattern string
	pattern = viper.GetString("cloudinary.keepfiles")
	if pattern != "" {
		settings.KeepFilesPattern = pattern
	}

	// mongodb section (optional)
	uri = viper.GetString("database.uri")
	if uri != "" {
		var mURI *url.URL
		if mURI, err = url.Parse(uri); err != nil {
			return nil, errors.New(fmt.Sprint("mongoDB URI: ", err.Error()))
		}
		settings.MongoURI = mURI
	} else {
		fmt.Fprintf(os.Stderr, "Warning: database not set (upload sync disabled)\n")
	}
	// Looks for env variables, perform substitutions if needed
	if err := settings.handleEnvVars(); err != nil {
		return nil, err
	}
	return settings, nil
}

// Parses all structure fields values, looks for any
// variable defined as ${VARNAME} and substitute it by
// calling os.Getenv().
//
// The reflect package is not used here since we cannot
// set a private field (not exported) within a struct using
// reflection.
func (c *Config) handleEnvVars() error {
	// [cloudinary]
	if c.CloudinaryURI != nil {
		curi, err := handleQuery(c.CloudinaryURI)
		if err != nil {
			return err
		}
		c.CloudinaryURI = curi
	}
	if len(c.PrependPath) == 0 {
		// [global]
		if len(c.ProdTag) > 0 {
			ptag, err := replaceEnvVars(c.ProdTag)
			if err != nil {
				return err
			}
			c.PrependPath = cloudinary.EnsureTrailingSlash(ptag)
		}
	}

	// [database]
	if c.MongoURI != nil {
		muri, err := handleQuery(c.MongoURI)
		if err != nil {
			return err
		}
		c.MongoURI = muri
	}
	return nil
}

// replaceEnvVars replaces all ${VARNAME} with their value
// using os.Getenv().
func replaceEnvVars(src string) (string, error) {
	r, err := regexp.Compile(`\${([A-Z_]+)}`)
	if err != nil {
		return "", err
	}
	envs := r.FindAllString(src, -1)
	for _, varname := range envs {
		evar := os.Getenv(varname[2 : len(varname)-1])
		if evar == "" {
			return "", errors.New(fmt.Sprintf("error: env var %s not defined", varname))
		}
		src = strings.Replace(src, varname, evar, -1)
	}
	return src, nil
}

func handleQuery(uri *url.URL) (*url.URL, error) {
	qs, err := url.QueryUnescape(uri.String())
	if err != nil {
		return nil, err
	}
	r, err := replaceEnvVars(qs)
	if err != nil {
		return nil, err
	}
	wuri, err := url.Parse(r)
	if err != nil {
		return nil, err
	}
	return wuri, nil
}
func ensureTrailingSlash(dirname string) string {
	if !strings.HasSuffix(dirname, "/") {
		dirname += "/"
	}
	return dirname
}
func perror(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	os.Exit(1)
}

func step(caption string) {
	fmt.Printf("==> %s\n", caption)
}

func printPublicID(publicID string) {
	fmt.Println("==> PublicID:", publicID)
}

func composePublicID(opt string) string {
	var prepend string
	if optPath != "" {
		prepend = ensureTrailingSlash(optPath)
	} else if settings.PrependPath != "" {
		prepend = ensureTrailingSlash(settings.PrependPath)
	}
	if optRaw != "" {
		return prepend + opt
	}
	return cloudinary.CleanExtensionNameWithPrepend(opt, prepend)
}
