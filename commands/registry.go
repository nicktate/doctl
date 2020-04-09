/*
Copyright 2018 The Doctl Authors All rights reserved.
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

package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	k8sapiv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

type dockerConfig struct {
	Auths map[string]struct {
		Auth string `json:"auth,omitempty"`
	} `json:"auths"`
}

// Registry creates the registry command
func Registry() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "registry",
			Aliases: []string{"reg", "r"},
			Short:   "[Beta] Display commands for working with container registries",
			Long:    "[Beta] The subcommands of `doctl registry` create, manage, and allow access to your private container registry.",
			Hidden:  true,
		},
	}

	createRegDesc := "This command creates a new private container registry with the provided name."
	CmdBuilder(cmd, RunRegistryCreate, "create <registry-name>",
		"Create a private container registry", createRegDesc, Writer)

	getRegDesc := "This command retrieves details about a private container registry including its name and the endpoint used to access it."
	CmdBuilder(cmd, RunRegistryGet, "get", "Retrieve details about a container registry",
		getRegDesc, Writer, aliasOpt("g"), displayerType(&displayers.Registry{}))

	deleteRegDesc := "This command permanently deletes a private container registry and all of its contents."
	cmdRunRegistryDelete := CmdBuilder(cmd, RunRegistryDelete, "delete",
		"Delete a container registry", deleteRegDesc, Writer, aliasOpt("d", "del", "rm"))
	AddBoolFlag(cmdRunRegistryDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force registry delete")

	cmd.AddCommand(Repository())

	loginRegDesc := "This command logs in Docker so that pull and push commands to your private container registry will be authenticated."
	CmdBuilder(cmd, RunRegistryLogin, "login", "Log in Docker to a container registry",
		loginRegDesc, Writer)

	logoutRegDesc := "This command logs Docker out of the private container registry, revoking access to it."
	CmdBuilder(cmd, RunRegistryLogout, "logout", "Log out Docker from a container registry",
		logoutRegDesc, Writer)

	kubeManifestDesc := `This command outputs a YAML-formated Kubernetes secret manifest that can be used to grant a Kubernetes cluster pull access to your private container registry.

Redirect the command's output to a file to save the manifest for later use or pipe it directly to kubectl to create the secret in your cluster:

    doctl registry kubernetes-manifest | kubectl apply -f -
`
	cmdRunKubernetesManifest := CmdBuilder(cmd, RunKubernetesManifest, "kubernetes-manifest",
		"Generate a Kubernetes secret manifest for a registry",
		kubeManifestDesc, Writer, aliasOpt("k8s"))
	AddStringFlag(cmdRunKubernetesManifest, doctl.ArgObjectName, "", "",
		"The secret name to create. Defaults to the registry name prefixed with \"registry-\"")
	AddStringFlag(cmdRunKubernetesManifest, doctl.ArgObjectNamespace, "",
		"default", "The Kubernetes namespace to hold the secret")

	return cmd
}

// Repository creates the repository sub-command
func Repository() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "repository",
			Aliases: []string{"repo", "r"},
			Short:   "[Beta] Display commands for working with repositories in a container registry",
			Long:    "[Beta] The subcommands of `doctl registry repository` help you command actions related to a repository.",
			Hidden:  true,
		},
	}

	listRepositoriesDesc := `
	This command retrieves information about repositories in a registry, including:

	- The repository name
	- The latest tag of the repository
	- The compressed size for the latest tag
	- The manifest digest for the latest tag
	- The last updated timestamp
	`
	CmdBuilder(cmd, RunListRepositories, "list",
		"List repositories for a container registry", listRepositoriesDesc, Writer, aliasOpt("ls"), displayerType(&displayers.Repository{}))

	listRepositoryTagsDesc := `
	This command retrieves information about tags in a repository, including:

	- The tag name
	- The compressed size 
	- The manifest digest 
	- The last updated timestamp
	`
	CmdBuilder(cmd, RunListRepositoryTags, "list-tags <repository>",
		"List tags for a repository in a container registry", listRepositoryTagsDesc, Writer, aliasOpt("lt"), displayerType(&displayers.RepositoryTag{}))

	deleteTagDesc := "This command permanently deletes a repository tag."
	cmdRunRepositoryDeleteTag := CmdBuilder(
		cmd,
		RunRepositoryDeleteTag,
		"delete-tag <repository> <tag>",
		"Delete a container repository tag",
		deleteTagDesc,
		Writer,
		aliasOpt("dt"),
	)
	AddBoolFlag(cmdRunRepositoryDeleteTag, doctl.ArgForce, doctl.ArgShortForce, false, "Force tag deletion")

	bulkDeleteTagsDesc := "This command permanently deletes one or more repository tags."
	cmdRunRepositoryBulkDeleteTags := CmdBuilder(
		cmd,
		RunRepositoryBulkDeleteTags,
		"bulk-delete-tags <repository> <tag>...",
		"Bulk delete one or more container repository tags",
		bulkDeleteTagsDesc,
		Writer,
		aliasOpt("bdt"),
	)
	AddBoolFlag(cmdRunRepositoryBulkDeleteTags, doctl.ArgForce, doctl.ArgShortForce, false, "Force tag deletion")

	tagDeletionStatusDesc := "This command retrieves the deletion status information from one or more tags in a bulk delete request."
	CmdBuilder(
		cmd,
		RunRepositoryTagDeletionStatus,
		"tag-deletion-status <repository> <tag>...",
		"Get status of a container repository tag deletion",
		tagDeletionStatusDesc,
		Writer,
		aliasOpt("tds"),
		displayerType(&displayers.DeletionStatus{}),
	)

	deleteManifestDesc := "This command permanently deletes one ore more repository manifests."
	cmdRunRepositoryDeleteManifest := CmdBuilder(
		cmd,
		RunRepositoryDeleteManifest,
		"delete-manifest <repository> <manifest-digest>",
		"Delete a container repository manifest",
		deleteManifestDesc,
		Writer,
		aliasOpt("dm"),
	)
	AddBoolFlag(cmdRunRepositoryDeleteManifest, doctl.ArgForce, doctl.ArgShortForce, false, "Force manifest deletion")

	bulkDeleteManifestsDesc := "This command permanently deletes one or more repository manifests."
	cmdRunRepositoryBulkDeleteManifests := CmdBuilder(
		cmd,
		RunRepositoryBulkDeleteManifests,
		"bulk-delete-manifests <repository> <manifest-digest>...",
		"Bulk delete one or more container repository manifests",
		bulkDeleteManifestsDesc,
		Writer,
		aliasOpt("bdm"),
	)
	AddBoolFlag(cmdRunRepositoryBulkDeleteManifests, doctl.ArgForce, doctl.ArgShortForce, false, "Force manifest deletion")

	manifestDeletionStatusDesc := "This command retrieves the deletion status information from one or more manifests in a bulk delete request."
	CmdBuilder(
		cmd,
		RunRepositoryTagDeletionStatus,
		"manifest-deletion-status <repository> <manifest-digest>...",
		"Get status of a container repository manifest deletion",
		manifestDeletionStatusDesc,
		Writer,
		aliasOpt("mds"),
		displayerType(&displayers.DeletionStatus{}),
	)

	return cmd
}

// RunRegistryCreate creates a registry
func RunRegistryCreate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	name := c.Args[0]
	rs := c.Registry()

	rcr := &godo.RegistryCreateRequest{Name: name}
	r, err := rs.Create(rcr)
	if err != nil {
		return err
	}

	return displayRegistries(c, *r)
}

// RunRegistryGet returns the registry
func RunRegistryGet(c *CmdConfig) error {
	reg, err := c.Registry().Get()
	if err != nil {
		return err
	}

	return displayRegistries(c, *reg)
}

// RunRegistryDelete delete the registry
func RunRegistryDelete(c *CmdConfig) error {
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirm("delete registry") != nil {
		return fmt.Errorf("operation aborted")
	}

	return c.Registry().Delete()
}

// RunListRepositories lists repositories for the registry
func RunListRepositories(c *CmdConfig) error {
	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	repositories, err := c.Registry().ListRepositories(registry.Name)
	if err != nil {
		return err
	}

	return displayRepositories(c, repositories...)
}

// RunListRepositoryTags lists tags for the repository in a registry
func RunListRepositoryTags(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	tags, err := c.Registry().ListRepositoryTags(registry.Name, c.Args[0])
	if err != nil {
		return err
	}

	return displayRepositoryTags(c, tags...)
}

// RunRepositoryDeleteTag deletes the repository tag
func RunRepositoryDeleteTag(c *CmdConfig) error {
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	if !force && AskForConfirm("delete repository tag") != nil {
		return fmt.Errorf("operation aborted")
	}

	repository := c.Args[0]
	tag := c.Args[1]

	return c.Registry().DeleteTag(registry.Name, repository, tag)
}

// RunRepositoryBulkDeleteTags deletes one or more repository tags
func RunRepositoryBulkDeleteTags(c *CmdConfig) error {
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	repository := c.Args[0]
	tags := c.Args[1:]

	if !force && AskForConfirm(fmt.Sprintf("delete %d repository tag(s)", len(tags))) != nil {
		return fmt.Errorf("operation aborted")
	}

	return c.Registry().BulkDeleteTags(registry.Name, repository, &godo.RepositoryBulkDeleteTagsRequest{
		Tags: tags,
	})
}

// RunRepositoryTagDeletionStatus gets the status of a tag deletion.
func RunRepositoryTagDeletionStatus(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	repository := c.Args[0]
	references := c.Args[1:]
	var statuses []do.DeletionStatus

	for _, ref := range references {
		status, err := c.Registry().GetTagDeletionStatus(registry.Name, repository, ref)
		if err != nil {
			return err
		}

		statuses = append(statuses, *status)
	}

	return displayDeletionStatuses(c, statuses...)
}

// RunRepositoryDeleteManifest deletes the repository manifest by digest
func RunRepositoryDeleteManifest(c *CmdConfig) error {
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	if !force && AskForConfirm("delete repository manifest by digest") != nil {
		return fmt.Errorf("operation aborted")
	}

	repository := c.Args[0]
	digest := c.Args[1]

	return c.Registry().DeleteManifest(registry.Name, repository, digest)
}

// RunRepositoryBulkDeleteManifests deletes one or more repository tags
func RunRepositoryBulkDeleteManifests(c *CmdConfig) error {
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	repository := c.Args[0]
	digests := c.Args[1:]

	if !force && AskForConfirm(fmt.Sprintf("delete %d repository manifest(s)", len(digests))) != nil {
		return fmt.Errorf("operation aborted")
	}

	return c.Registry().BulkDeleteManifests(registry.Name, repository, &godo.RepositoryBulkDeleteManifestsRequest{
		ManifestDigests: digests,
	})
}

// RunRepositoryManifestDeletionStatus gets the status of a tag deletion.
func RunRepositoryManifestDeletionStatus(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	repository := c.Args[0]
	references := c.Args[1:]
	var statuses []do.DeletionStatus

	for _, ref := range references {
		status, err := c.Registry().GetManifestDeletionStatus(registry.Name, repository, ref)
		if err != nil {
			return err
		}

		statuses = append(statuses, *status)
	}

	return displayDeletionStatuses(c, statuses...)
}

// store execCommand in a variable. Lets us override it while testing
var execCommand = exec.Command

// RunRegistryLogin logs in Docker to the registry
func RunRegistryLogin(c *CmdConfig) error {
	// check if docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("unable to find the Docker CLI binary. Make sure docker is installed")
	}

	fmt.Printf("Logging Docker in to %s\n", c.Registry().Endpoint())

	creds, err := c.Registry().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
		ReadWrite: true,
	})
	if err != nil {
		return err
	}

	var dc dockerConfig
	err = json.Unmarshal(creds.DockerConfigJSON, &dc)
	if err != nil {
		return err
	}

	// read the login credentials from the docker config
	for host, conf := range dc.Auths {
		// decode and split into username + password
		creds, err := base64.StdEncoding.DecodeString(conf.Auth)
		if err != nil {
			return err
		}

		splitCreds := strings.Split(string(creds), ":")
		if len(splitCreds) != 2 {
			return fmt.Errorf("got invalid docker credentials")
		}
		user, pass := splitCreds[0], splitCreds[1]

		// log in via the docker cli
		args := []string{
			"login", host,
			"-u", user,
			"--password-stdin",
		}
		cmd := execCommand("docker", args...)
		cmd.Stdin = strings.NewReader(pass)
		cmd.Stdout = c.Out
		cmd.Stderr = c.Out

		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}

// RunKubernetesManifest prints a Kubernetes manifest that provides read/pull access to the registry
func RunKubernetesManifest(c *CmdConfig) error {
	secretName, err := c.Doit.GetString(c.NS, doctl.ArgObjectName)
	if err != nil {
		return err
	}
	secretNamespace, err := c.Doit.GetString(c.NS, doctl.ArgObjectNamespace)
	if err != nil {
		return err
	}

	// if no secret name supplied, use the registry name
	if secretName == "" {
		reg, err := c.Registry().Get()
		if err != nil {
			return err
		}
		secretName = "registry-" + reg.Name
	}

	// fetch docker config
	dockerCreds, err := c.Registry().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
		ReadWrite: false,
	})
	if err != nil {
		return err
	}

	// create the manifest for the secret
	secret := &k8sapiv1.Secret{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
		Type: k8sapiv1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": dockerCreds.DockerConfigJSON,
		},
	}

	serializer := k8sjson.NewSerializerWithOptions(
		k8sjson.DefaultMetaFactory, nil, nil,
		k8sjson.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)

	return serializer.Encode(secret, c.Out)
}

// RunRegistryLogout logs Docker out of the registry
func RunRegistryLogout(c *CmdConfig) error {
	// check if docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("unable to find the Docker CLI binary. Make sure docker is installed")
	}

	cmd := execCommand("docker", "logout", c.Registry().Endpoint())
	cmd.Stdout = c.Out
	cmd.Stderr = c.Out

	return cmd.Run()
}

func displayRegistries(c *CmdConfig, registries ...do.Registry) error {
	item := &displayers.Registry{
		Registries: registries,
	}
	return c.Display(item)
}

func displayRepositoryTags(c *CmdConfig, tags ...do.RepositoryTag) error {
	item := &displayers.RepositoryTag{
		Tags: tags,
	}
	return c.Display(item)
}

func displayRepositories(c *CmdConfig, repositories ...do.Repository) error {
	item := &displayers.Repository{
		Repositories: repositories,
	}
	return c.Display(item)
}

func displayDeletionStatuses(c *CmdConfig, statuses ...do.DeletionStatus) error {
	item := &displayers.DeletionStatus{
		Statuses: statuses,
	}
	return c.Display(item)
}
