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

	cloudinary "github.com/rootsongjc/cloudinary-go"
	"github.com/spf13/cobra"
)

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove file",
	Run: func(cmd *cobra.Command, args []string) {
		if optRaw == "" && optImg == "" {
			fail("Missing -i or -r option.")
		}
		var prepend string
		if optPath != "" {
			prepend = ensureTrailingSlash(optPath)
		} else if settings.PrependPath != "" {
			prepend = ensureTrailingSlash(settings.PrependPath)
		}
		if optRaw != "" {
			publicID := composePublicID(optRaw)
			printPublicID(publicID)
			step(fmt.Sprintf("Deleting raw file %s", optRaw))
			if err := service.Delete(optRaw, prepend, cloudinary.RawType); err != nil {
				perror(err)
			}
		} else {
			publicID := composePublicID(optImg)
			printPublicID(publicID)
			step(fmt.Sprintf("Deleting image %s", optImg))
			if err := service.Delete(optImg, prepend, cloudinary.ImageType); err != nil {
				perror(err)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(rmCmd)
}
