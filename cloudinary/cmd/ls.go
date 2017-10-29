// Copyright © 2017 Jimmy Song
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
	"fmt"
	"os"
	"strings"

	cloudinary "github.com/rootsongjc/cloudinary-go"
	"github.com/spf13/cobra"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List files",
	Run: func(cmd *cobra.Command, args []string) {
		// list all resources
		if optImg == "" && optRaw == "" {
			fmt.Println("==> Raw resources:")
			printResources(service.Resources(cloudinary.RawType))
			fmt.Println("==> Images:")
			printResources(service.Resources(cloudinary.ImageType))
		} else { // list image resources
			var publicID string
			if optImg != "" {
				publicID = composePublicID(optImg)
				printPublicID(publicID)
				fmt.Println("==> Image Details:")
			}
			if optRaw != "" {
				fmt.Println("List raw resource details Not support")
				os.Exit(0)
			}
			printResourceDetails(service.ResourceDetails(publicID))
		}
	},
}

func init() {
	RootCmd.AddCommand(lsCmd)
}

func printResources(res []*cloudinary.Resource, err error) {
	if err != nil {
		fail(err.Error())
	}
	if len(res) == 0 {
		fmt.Println("No resource found.")
		return
	}
	fmt.Printf("%-30s %-10s %-5s %s\n", "public_id", "Version", "Type", "Size")
	fmt.Println(strings.Repeat("-", 70))
	for _, r := range res {
		fmt.Printf("%-30s %d %s %10d\n", r.PublicId, r.Version, r.ResourceType, r.Size)
	}
}

func printResourceDetails(res *cloudinary.ResourceDetails, err error) {
	if err != nil {
		fail(err.Error())
	}
	if res == nil || len(res.PublicId) == 0 {
		fmt.Println("No resource details found.")
		return
	}
	fmt.Printf("%-30s %-6s %-10s %-5s %-8s %-6s %-6s %-s\n", "public_id", "Format", "Version", "Type", "Size(KB)", "Width", "Height", "Url")
	fmt.Printf("%-30s %-6s %-10d %-5s %-8d %-6d %-6d %-s\n", res.PublicId, res.Format, res.Version, res.ResourceType, res.Size/1024, res.Width, res.Height, res.Url)

	fmt.Println()

	for i, d := range res.Derived {
		if i == 0 {
			fmt.Printf("%-25s %-8s %-s\n", "transformation", "Size", "Url")
		}
		fmt.Printf("%-25s %-8d %-s\n", d.Transformation, d.Size, d.Url)
	}
}

func fail(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	os.Exit(1)
}
