/*
Copyright 2017 Kinvolk GmbH

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const tableFmt = "%-10s  %s"

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "print the created environments",
		Run:   runList,
	}
)

func init() {
	kubespawnCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		log.Fatalf("too many arguments: %v", args)
	}

	ksDir := viper.GetString("dir")

	matches, err := filepath.Glob(path.Join(ksDir, "*"))
	if err != nil {
		log.Fatal(err)

	}

	var found [][]string
	for _, m := range matches {
		name := filepath.Base(m)
		// skip .cache
		if strings.HasPrefix(name, ".") {
			continue
		}
		fi, err := os.Stat(m)
		if err != nil {
			log.Fatal(err)
		}
		if fi.IsDir() {
			found = append(found, []string{
				fi.Name(),
				fi.ModTime().Format(time.Stamp),
			})
		}
	}

	if len(found) < 1 {
		log.Printf("no environments found")
		return
	}

	printTable(found)
}

func printTable(found [][]string) {
	log.Printf(tableFmt, "ENV NAME", "LAST MODIFIED")
	for _, e := range found {
		log.Printf(tableFmt, e[0], e[1])
	}
	log.Printf("\n%d environment(s) found", len(found))
}
