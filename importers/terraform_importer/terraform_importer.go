package terraformimporter

import (
	"log"
	"path/filepath"
	"sort"
	"strings"

	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/jhump/protoreflect/desc/builder"

	"github.com/protoconf/protoconf/importers"
	"github.com/protoconf/protoconf/importers/terraform_importer/meta"
)

// Generator is used to create `terrafrom.proto` file with all of its
// available resources from the provider used in the current directory
// context.
type Generator struct {
	Providers  map[string]*ProviderImporter
	ImportPath string
	Importer   *importers.Importer
}

// NewGenerator creates a new Generator
func NewGenerator(importPath, outputPath string) *Generator {
	providers := make(map[string]*ProviderImporter)
	return &Generator{
		ImportPath: importPath,
		Importer:   importers.NewImporter("terraform/terraform.proto", outputPath),
		Providers:  providers,
	}
}

// PopulateProviders finds all providers available for the runtime context
// and populate the Generator with proto schemas from `providerimporter`
func (g *Generator) PopulateProviders() error {
	abs, err := filepath.Abs(g.ImportPath)
	if err != nil {
		return err
	}
	log.Println(abs)
	dirs, err := filepath.Glob(abs)
	if err != nil {
		return err
	}
	log.Println(dirs)
	meta := discovery.FindPlugins("provider", dirs)
	log.Println(meta)

	for c := range meta {
		config := tfplugin.ClientConfig(c)
		client, err := NewGRPCClient(config)
		if err != nil {
			return err
		}
		p, err := NewProviderImporter(c.Name, client)
		if err != nil {
			return err
		}
		g.Providers[c.Name] = p
	}

	return nil
}

func addVariableMessage(main *builder.MessageBuilder) error {
	local := builder.NewMessage("Variable")
	fieldType := builder.NewField("type", builder.FieldTypeString())
	fieldDesc := builder.NewField("description", builder.FieldTypeString())

	// default should be replaced eith google.protobuf.Value
	fieldDefault := builder.NewField("default", builder.FieldTypeString())
	local.AddField(fieldType)
	local.AddField(fieldDesc)
	local.AddField(fieldDefault)
	main.AddNestedMessage(local)
	main.AddField(builder.NewMapField("variable", builder.FieldTypeString(), builder.FieldTypeMessage(local)))
	return nil
}

func addOutputMessage(main *builder.MessageBuilder) error {
	local := builder.NewMessage("Output")
	fieldValue := builder.NewField("value", builder.FieldTypeString())
	local.AddField(fieldValue)
	main.AddNestedMessage(local)
	main.AddField(builder.NewMapField("output", builder.FieldTypeString(), builder.FieldTypeMessage(local)))
	return nil
}

func addLocalMessage(main *builder.MessageBuilder) error {
	main.AddField(builder.NewMapField("locals", builder.FieldTypeString(), builder.FieldTypeString()))
	return nil
}

func addModuleMessage(main *builder.MessageBuilder) error {
	local := builder.NewMessage("Module")
	main.AddNestedMessage(local)
	main.AddField(builder.NewField("module", builder.FieldTypeMessage(local)))
	return nil
}

func addTerraformLocalBackend(b *builder.MessageBuilder, oof *builder.OneOfBuilder) error {
	l := builder.NewMessage("BackendLocal")
	l.AddField(builder.NewField("path", builder.FieldTypeString()))
	l.AddField(builder.NewField("workspace_dir", builder.FieldTypeString()).SetJsonName("workspace_dir"))
	b.AddNestedMessage(l)
	oof.AddChoice(builder.NewField("local", builder.FieldTypeMessage(l)))
	return nil
}

func defaultStringField(m *builder.MessageBuilder, name string, comments ...string) {
	c := builder.Comments{LeadingComment: strings.Join(comments, "\n")}
	m.AddField(builder.NewField(name, builder.FieldTypeString()).SetJsonName(name).SetComments(c))
}

func addTerraformRemoteBackend(b *builder.MessageBuilder, oof *builder.OneOfBuilder) error {
	l := builder.NewMessage("BackendRemote")
	defaultStringField(l, "hostname",
		"(Optional) The remote backend hostname to connect to. Defaults to app.terraform.io.",
	)
	defaultStringField(l, "organization", "(Required) The name of the organization containing the targeted workspace(s).")
	defaultStringField(l, "token", "(Optional) The token used to authenticate with the remote backend. We recommend omitting the token from the configuration, and instead using `terraform login` or manually configuring `credentials` in the CLI config file.")

	c := builder.Comments{LeadingComment: "(Required) A block specifying which remote workspace(s) to use. The workspaces block supports the following keys"}
	ws := builder.NewMessage("Workspace")
	defaultStringField(ws, "name", "(Optional) The full name of one remote workspace. When configured, only the default workspace can be used. This option conflicts with prefix.")
	defaultStringField(ws, "prefix", "(Optional) A prefix used in the names of one or more remote workspaces, all of which can be used with this configuration. The full workspace names are used in Terraform Cloud, and the short names (minus the prefix) are used on the command line for Terraform CLI workspaces. If omitted, only the default workspace can be used. This option conflicts with name.")
	l.AddNestedMessage(ws)
	l.AddField(builder.NewField("workspaces", builder.FieldTypeMessage(ws)).SetComments(c))

	b.AddNestedMessage(l)
	oof.AddChoice(builder.NewField("remote", builder.FieldTypeMessage(l)))
	return nil
}

func addTerraformS3Backend(b *builder.MessageBuilder, oof *builder.OneOfBuilder) error {
	l := builder.NewMessage("BackendS3")
	defaultStringField(l, "region")
	defaultStringField(l, "access_key")
	defaultStringField(l, "secret_key")
	defaultStringField(l, "iam_endpoint")                // (Optional) Custom endpoint for the AWS Identity and Access Management (IAM) API. This can also be sourced from the AWS_IAM_ENDPOINT environment variable.
	defaultStringField(l, "max_retries")                 // (Optional) The maximum number of times an AWS API request is retried on retryable failure. Defaults to 5.
	defaultStringField(l, "profile")                     // (Optional) Name of AWS profile in AWS shared credentials file (e.g. ~/.aws/credentials) or AWS shared configuration file (e.g. ~/.aws/config) to use for credentials and/or configuration. This can also be sourced from the AWS_PROFILE environment variable.
	defaultStringField(l, "shared_credentials_file")     // (Optional) Path to the AWS shared credentials file. Defaults to ~/.aws/credentials.
	defaultStringField(l, "skip_credentials_validation") // (Optional) Skip credentials validation via the STS API.
	defaultStringField(l, "skip_region_validation")      // (Optional) Skip validation of provided region name.
	defaultStringField(l, "skip_metadata_api_check")     // (Optional) Skip usage of EC2 Metadata API.
	defaultStringField(l, "sts_endpoint")                // (Optional) Custom endpoint for the AWS Security Token Service (STS) API. This can also be sourced from the AWS_STS_ENDPOINT environment variable.
	defaultStringField(l, "token")                       // (Optional) Multi-Factor Authentication (MFA) token. This can also be sourced from the AWS_SESSION_TOKEN environment variable.

	defaultStringField(l, "assume_role_duration_seconds")    // (Optional) Number of seconds to restrict the assume role session duration.
	defaultStringField(l, "assume_role_policy")              // (Optional) IAM Policy JSON describing further restricting permissions for the IAM Role being assumed.
	defaultStringField(l, "assume_role_policy_arns")         // (Optional) Set of Amazon Resource Names (ARNs) of IAM Policies describing further restricting permissions for the IAM Role being assumed.
	defaultStringField(l, "assume_role_tags")                // (Optional) Map of assume role session tags.
	defaultStringField(l, "assume_role_transitive_tag_keys") // (Optional) Set of assume role session tag keys to pass to any subsequent sessions.
	defaultStringField(l, "external_id")                     // (Optional) External identifier to use when assuming the role.
	defaultStringField(l, "role_arn")                        // (Optional) Amazon Resource Name (ARN) of the IAM Role to assume.
	defaultStringField(l, "session_name")                    // (Optional) Session name to use when assuming the role.

	defaultStringField(l, "bucket") // (Required) Name of the S3 Bucket.
	defaultStringField(l, "key")    // (Required) Path to the state file inside the S3 Bucket. When using a non-default workspace, the state path will be /workspace_key_prefix/workspace_name/key (see also the workspace_key_prefix configuration).

	defaultStringField(l, "acl")                  // (Optional) Canned ACL to be applied to the state file.
	defaultStringField(l, "encrypt")              // (Optional) Enable server side encryption of the state file.
	defaultStringField(l, "endpoint")             // (Optional) Custom endpoint for the AWS S3 API. This can also be sourced from the AWS_S3_ENDPOINT environment variable.
	defaultStringField(l, "force_path_style")     // (Optional) Enable path-style S3 URLs (https://<HOST>/<BUCKET> instead of https://<BUCKET>.<HOST>).
	defaultStringField(l, "kms_key_id")           // (Optional) Amazon Resource Name (ARN) of a Key Management Service (KMS) Key to use for encrypting the state.
	defaultStringField(l, "sse_customer_key")     // (Optional) The key to use for encrypting state with Server-Side Encryption with Customer-Provided Keys (SSE-C). This is the base64-encoded value of the key, which must decode to 256 bits. This can also be sourced from the AWS_SSE_CUSTOMER_KEY environment variable, which is recommended due to the sensitivity of the value. Setting it inside a terraform file will cause it to be persisted to disk in terraform.tfstate.
	defaultStringField(l, "workspace_key_prefix") // (Optional) Prefix applied to the state path inside the bucket. This is only relevant when using a non-default workspace. Defaults to env:.

	defaultStringField(l, "dynamodb_endpoint") // (Optional) Custom endpoint for the AWS DynamoDB API. This can also be sourced from the AWS_DYNAMODB_ENDPOINT environment variable.
	defaultStringField(l, "dynamodb_table")    // (Optional) Name of DynamoDB Table to use for state locking and consistency. The table must have a primary key named LockID with type of string. If not configured, state locking will be disabled.
	b.AddNestedMessage(l)
	oof.AddChoice(builder.NewField("s3", builder.FieldTypeMessage(l)))
	return nil
}

func addTerraformBackend(tf *builder.MessageBuilder) error {
	backend := builder.NewMessage("Backend")
	oneof := builder.NewOneOf("config")
	addTerraformLocalBackend(backend, oneof)
	addTerraformRemoteBackend(backend, oneof)
	addTerraformS3Backend(backend, oneof)

	backend.AddOneOf(oneof)
	tf.AddNestedMessage(backend)
	tf.AddField(builder.NewField("backend", builder.FieldTypeMessage(backend)))
	return nil
}

func addTerraformConfigMessage(main *builder.MessageBuilder) error {
	local := builder.NewMessage("TerraformSettings")
	fieldTerraformVersion := builder.NewField("required_version", builder.FieldTypeString()).SetJsonName("required_version")
	msgProvider := builder.NewMessage("Provider")
	fieldSource := builder.NewField("source", builder.FieldTypeString())
	fieldVersion := builder.NewField("version", builder.FieldTypeString())
	msgProvider.AddField(fieldSource)
	msgProvider.AddField(fieldVersion)

	local.AddField(fieldTerraformVersion)
	local.AddNestedMessage(msgProvider)
	local.AddField(builder.NewMapField("required_providers", builder.FieldTypeString(), builder.FieldTypeMessage(msgProvider)).SetJsonName("required_providers"))
	addTerraformBackend(local)

	main.AddNestedMessage(local)
	main.AddField(builder.NewField("terraform", builder.FieldTypeMessage(local)))
	return nil
}

// Save will write all protofiles to disk
func (g *Generator) Save() error {
	file := g.Importer.MasterFile
	file.SetProto3(true)
	file.SetPackageName("terraform")

	main := builder.NewMessage("Terraform")
	resources := builder.NewMessage("Resources")
	data := builder.NewMessage("Datasources")
	providers := builder.NewMessage("Providers")

	main.AddNestedMessage(resources)
	main.AddField(builder.NewField("resource", builder.FieldTypeMessage(resources)).SetJsonName("resource"))
	main.AddNestedMessage(data)
	main.AddField(builder.NewField("data", builder.FieldTypeMessage(data)).SetJsonName("data"))
	main.AddNestedMessage(providers)
	main.AddField(builder.NewField("provider", builder.FieldTypeMessage(providers)).SetJsonName("provider"))
	addVariableMessage(main)
	addOutputMessage(main)
	addLocalMessage(main)
	addModuleMessage(main)
	addTerraformConfigMessage(main)

	metaFile := meta.MetaFile()
	protoFiles := []*builder.FileBuilder{file, metaFile}

	keys := []string{}
	for name := range g.Providers {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	for _, name := range keys {
		p := g.Providers[name]
		log.Println("saving", name)
		protoFiles = append(protoFiles, p.Resources)
		protoFiles = append(protoFiles, p.Datasources)
		protoFiles = append(protoFiles, p.Provider)
		for _, b := range p.Provider.GetMessage("resources").GetChildren() {
			f := p.Provider.GetMessage("resources").GetField(b.GetName())
			resources.AddField(f)
		}
		for _, b := range p.Provider.GetMessage("data").GetChildren() {
			f := p.Provider.GetMessage("data").GetField(b.GetName())
			data.AddField(f)
		}
		for _, b := range p.Provider.GetMessage("provider").GetChildren() {
			f := p.Provider.GetMessage("provider").GetField(b.GetName())
			providers.AddField(f)
		}
	}
	file.AddMessage(main)
	for _, f := range protoFiles {
		g.Importer.RegisterFile(f)
	}

	return g.Importer.SaveAll()

}
