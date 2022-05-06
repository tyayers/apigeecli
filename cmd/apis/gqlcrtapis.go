// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apis

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/apigee/apigeecli/apiclient"
	bundle "github.com/apigee/apigeecli/bundlegen"
	proxybundle "github.com/apigee/apigeecli/bundlegen/proxybundle"
	"github.com/apigee/apigeecli/client/apis"
	"github.com/apigee/apigeecli/clilog"
	"github.com/spf13/cobra"
)

var GqlCreateCmd = &cobra.Command{
	Use:     "graphql",
	Aliases: []string{"gql"},
	Short:   "Creates an API proxy from a GraphQL schema",
	Long:    "Creates an API proxy from a GraphQL schema",
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if gqlFile == "" && gqlURI == "" {
			return fmt.Errorf("Either gqlfile or gqlurl must be passed")
		}
		return apiclient.SetApigeeOrg(org)
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var content []byte
		var gqlDocName string
		if gqlFile != "" {
			gqlDocName, content, err = readSchemaFile()
		} else {
			gqlDocName, content, err = readSchemaURL()
		}
		if err != nil {
			return err
		}

		location, keyName, err := getKeyNameAndLocation()
		if err != nil {
			return err
		}

		//Generate the apiproxy struct
		err = bundle.GenerateAPIProxyDefFromGQL(name,
			gqlDocName,
			basePath,
			targetUrlRef,
			apiKeyLocation,
			skipPolicy,
			addCORS)

		if err != nil {
			return err
		}

		//Create the API proxy bundle
		err = proxybundle.GenerateAPIProxyBundleFromGQL(name,
			string(content),
			gqlDocName,
			action,
			location,
			keyName,
			skipPolicy,
			addCORS,
			targetUrlRef)

		if err != nil {
			return err
		}

		if importProxy {
			_, err = apis.CreateProxy(name, name+".zip")
		}

		return err
	},
}

var gqlFile, gqlURI, basePath, action, apiKeyLocation string

func init() {
	GqlCreateCmd.Flags().StringVarP(&name, "name", "n",
		"", "API Proxy name")
	GqlCreateCmd.Flags().StringVarP(&gqlFile, "gqlfile", "f",
		"", "GraphQL schema file")
	GqlCreateCmd.Flags().StringVarP(&gqlURI, "gqluri", "u",
		"", "GraphQL schema URI location")
	GqlCreateCmd.Flags().StringVarP(&basePath, "basepath", "p",
		"", "Base Path of the API Proxy")
	GqlCreateCmd.Flags().StringVarP(&action, "action", "",
		"verify", "GraphQL policy action, must be oneOf parse, verify or parse_verify. Default is verify")
	GqlCreateCmd.Flags().StringVarP(&targetUrlRef, "target-url-ref", "",
		"", "Set a reference variable containing the target endpoint")
	GqlCreateCmd.Flags().StringVarP(&targetUrlRef, "apikey-location", "",
		"", "Set the location of the API key, ex: request.header.x-api-key")
	GqlCreateCmd.Flags().BoolVarP(&importProxy, "import", "",
		true, "Import API Proxy after generation from spec")
	GqlCreateCmd.Flags().BoolVarP(&skipPolicy, "skip-policy", "",
		false, "Skip adding the GraphQL Validate policy")
	GqlCreateCmd.Flags().BoolVarP(&addCORS, "add-cors", "",
		false, "Add a CORS policy")

	_ = GqlCreateCmd.MarkFlagRequired("name")
	_ = GqlCreateCmd.MarkFlagRequired("target-url-ref")
	_ = GqlCreateCmd.MarkFlagRequired("basepath")
}

func readSchemaFile() (string, []byte, error) {
	schemaFile, err := ioutil.ReadFile(gqlFile)
	if err != nil {
		clilog.Error.Println("Error reading file: ", err)
		return "", nil, err
	}
	return filepath.Base(gqlFile), schemaFile, nil
}

func readSchemaURL() (string, []byte, error) {
	u, err := url.Parse(gqlURI)
	if err != nil {
		clilog.Error.Println("Error reading uri: ", err)
		return "", nil, err
	}
	resp, err := apiclient.DownloadFile(gqlURI, false)
	if err != nil {
		clilog.Error.Println("Error downloading file: ", err)
		return "", nil, err
	}
	if resp != nil {
		defer resp.Body.Close()
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		clilog.Error.Println("error in response: ", err)
		return "", nil, err
	}
	return path.Base(u.Path), respBody, err
}

func getKeyNameAndLocation() (location string, key string, err error) {
	if apiKeyLocation == "" {
		return "", "", nil
	}

	apikey := strings.Split(apiKeyLocation, ".")
	if len(apikey) != 3 {
		return "", "", fmt.Errorf("api key location must be of the form request.header.foo or request.queryparam.bar")
	}
	return apikey[1], apikey[2], nil
}
