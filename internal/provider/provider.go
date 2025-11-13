package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/megum1n/terraform-provider-mongodb/internal/mongodb"
)

var (
	_ provider.Provider = &MongodbProvider{}
)

const (
	defaultDatabase = "admin"
)

type MongodbProvider struct {
	Version string
	client  *mongodb.Client
}

type MongodbProviderModel struct {
	Hosts              types.List   `tfsdk:"hosts"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	AuthSource         types.String `tfsdk:"auth_source"`
	AuthMechanism      types.String `tfsdk:"auth_mechanism"`
	ReplicaSet         types.String `tfsdk:"replica_set"`
	TLS                types.Bool   `tfsdk:"tls"`
	Certificate        types.String `tfsdk:"certificate"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	DirectConnection   types.Bool   `tfsdk:"direct_connection"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MongodbProvider{
			Version: version,
		}
	}
}

func (p *MongodbProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "mongodb"
	resp.Version = p.Version
}

func (p *MongodbProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "MongoDB resources management",

		Attributes: map[string]schema.Attribute{
			"hosts": schema.ListAttribute{
				MarkdownDescription: "MongoDB hosts",
				ElementType:         types.StringType,
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username",
				Optional:            true,
				Sensitive:           true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password",
				Optional:            true,
				Sensitive:           true,
			},
			"auth_source": schema.StringAttribute{
				MarkdownDescription: "AuthSource database",
				Optional:            true,
			},
			"auth_mechanism": schema.StringAttribute{
				MarkdownDescription: "Authentication mechanism (e.g., MONGODB-AWS, SCRAM-SHA-256)",
				Optional:            true,
			},
			"replica_set": schema.StringAttribute{
				MarkdownDescription: "Replica set name",
				Optional:            true,
			},
			"tls": schema.BoolAttribute{
				MarkdownDescription: "Enable TLS",
				Optional:            true,
			},
			"certificate": schema.StringAttribute{
				MarkdownDescription: "Certificate PEM string",
				Optional:            true,
			},
			"insecure_skip_verify": schema.BoolAttribute{
				MarkdownDescription: "Insecure TLS",
				Optional:            true,
			},
			"direct_connection": schema.BoolAttribute{
				MarkdownDescription: "Direct connection to MongoDB",
				Optional:            true,
			},
		},
	}
}

func (p *MongodbProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	var data MongodbProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.AuthSource.IsNull() {
		data.AuthSource = types.StringValue(defaultDatabase)
	}

	var err error
	var hosts []string

	diag := data.Hosts.ElementsAs(ctx, &hosts, false)
	resp.Diagnostics.Append(diag...)

	if resp.Diagnostics.HasError() {
		return
	}

	p.client, err = mongodb.New(ctx, &mongodb.ClientOptions{
		Hosts:              hosts,
		Username:           data.Username.ValueString(),
		Password:           data.Password.ValueString(),
		AuthSource:         data.AuthSource.ValueString(),
		AuthMechanism:      data.AuthMechanism.ValueString(),
		ReplicaSet:         data.ReplicaSet.ValueString(),
		TLS:                data.TLS.ValueBool(),
		Certificate:        data.Certificate.ValueString(),
		InsecureSkipVerify: data.InsecureSkipVerify.ValueBool(),
		DirectConnection:   data.DirectConnection.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to connect to MongoDB",
			err.Error(),
		)
	}

	resp.ResourceData = p
}

func (p *MongodbProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *MongodbProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewRoleResource,
		NewIndexResource,
	}
}
