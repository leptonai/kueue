/*
Copyright The Kubernetes Authors.

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

package create

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/util/templates"

	"sigs.k8s.io/kueue/cmd/kueuectl/app/util"
)

var (
	createExample = templates.Examples(`
		# Create local queue 
  		kueuectl create localqueue my-local-queue -c my-cluster-queue
	`)
)

func NewCreateCmd(clientGetter util.ClientGetter, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a resource",
		Example: createExample,
	}

	util.AddDryRunFlag(cmd)

	cmd.AddCommand(NewLocalQueueCmd(clientGetter, streams))
	cmd.AddCommand(NewClusterQueueCmd(clientGetter, streams))
	cmd.AddCommand(NewResourceFlavorCmd(clientGetter, streams))

	return cmd
}
