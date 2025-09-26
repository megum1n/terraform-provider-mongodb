package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/megum1n/terraform-provider-mongodb/internal/mongodb"
)

const (
	externalDatabase = "$external"
)

var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithConfigure = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	client *mongodb.Client
}

type UserResourceModel struct {
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	Database   types.String `tfsdk:"database"`
	Roles      types.Set    `tfsdk:"roles"`
	Mechanisms types.Set    `tfsdk:"mechanisms"`
}

func newUserResourceModel() UserResourceModel {
	return UserResourceModel{
		Roles:      types.SetNull(types.ObjectType{AttrTypes: mongodb.ShortRoleAttributeTypes}),
		Mechanisms: types.SetNull(types.StringType),
	}
}

func (u *UserResourceModel) GetMechanisms(ctx context.Context, ptr *[]string) diag.Diagnostics {
	diags := diag.Diagnostics{}

	mechanismsStrings := make([]types.String, 0, len(u.Mechanisms.Elements()))
	diags.Append(u.Mechanisms.ElementsAs(ctx, &mechanismsStrings, false)...)

	mechanisms := make([]string, 0, len(mechanismsStrings))

	for _, mechanism := range mechanismsStrings {
		mechanisms = append(mechanisms, mechanism.ValueString())
	}

	*ptr = mechanisms

	return diags
}

func (u *UserResourceModel) updateState(ctx context.Context, user *mongodb.User) diag.Diagnostics {
	diags := diag.Diagnostics{}

	u.Username = types.StringValue(user.Username)
	u.Database = types.StringValue(user.Database)

	roles, d := user.Roles.ToTerraformSet(ctx)
	diags.Append(d...)

	u.Roles = *roles

	// DocumentDB does not return mechanisms.
	// If that's the case - keep the value same as in plan.
	if len(user.Mechanisms) > 0 {
		u.Mechanisms, d = types.SetValueFrom(ctx, types.StringType, user.Mechanisms)
		diags.Append(d...)
	}

	return diags
}

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "MongoDB User resource",

		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "The name of the new user",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The user's password. "+
					"Must be empty for %q database", externalDatabase),
				Optional:  true,
				Sensitive: true,
			},
			"database": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Auth database name (auth source). "+
					"%q is used by default", defaultDatabase),
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(defaultDatabase),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"roles": schema.SetNestedAttribute{
				MarkdownDescription: "The roles granted to the user",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							MarkdownDescription: "Role name",
							Required:            true,
						},
						"db": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf("Target database name. "+
								"%q is used by default", defaultDatabase),
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString(defaultDatabase),
						},
					},
				},
			},
			"mechanisms": schema.SetAttribute{
				MarkdownDescription: "Specify the specific SCRAM mechanism " +
					"or mechanisms for creating SCRAM user credentials. Defaults to `SCRAM-SHA-256`.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default: setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("SCRAM-SHA-256"),
				})),
			},
		},
	}
}

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	p, ok := req.ProviderData.(*MongodbProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *MongodbProvider, got: %T. "+
				"Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = p.client
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.checkClient(resp.Diagnostics) {
		return
	}

	var plan = newUserResourceModel()

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse roles
	var roles []mongodb.ShortRole

	resp.Diagnostics.Append(plan.Roles.ElementsAs(ctx, &roles, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse mechanisms
	var mechanisms []string

	if !plan.Mechanisms.IsUnknown() {
		resp.Diagnostics.Append(plan.GetMechanisms(ctx, &mechanisms)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	user, err := r.client.UpsertUser(ctx, &mongodb.User{
		Username:   plan.Username.ValueString(),
		Password:   plan.Password.ValueString(),
		Database:   plan.Database.ValueString(),
		Roles:      roles,
		Mechanisms: mechanisms,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to upsert user",
			err.Error(),
		)

		return
	}

	resp.Diagnostics.Append(plan.updateState(ctx, user)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "user created")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.checkClient(resp.Diagnostics) {
		return
	}

	var plan = newUserResourceModel()

	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetUser(ctx, &mongodb.GetUserOptions{
		Username: plan.Username.ValueString(),
		Database: plan.Database.ValueString(),
	})
	if err != nil {
		if !errors.As(err, &mongodb.NotFoundError{}) {
			resp.Diagnostics.AddError(
				"failed to get user",
				err.Error(),
			)

			return
		}

		tflog.Debug(ctx, "user not found, removing from state")
		resp.State.RemoveResource(ctx)

		return
	}

	resp.Diagnostics.Append(plan.updateState(ctx, user)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.checkClient(resp.Diagnostics) {
		return
	}

	var plan = newUserResourceModel()

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse roles
	var roles []mongodb.ShortRole

	resp.Diagnostics.Append(plan.Roles.ElementsAs(ctx, &roles, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse mechanisms
	var mechanisms []string

	if !plan.Mechanisms.IsUnknown() {
		resp.Diagnostics.Append(plan.GetMechanisms(ctx, &mechanisms)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	user, err := r.client.UpsertUser(ctx, &mongodb.User{
		Username:   plan.Username.ValueString(),
		Password:   plan.Password.ValueString(),
		Database:   plan.Database.ValueString(),
		Roles:      roles,
		Mechanisms: mechanisms,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to upsert user",
			err.Error(),
		)

		return
	}

	resp.Diagnostics.Append(plan.updateState(ctx, user)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "user updated")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.checkClient(resp.Diagnostics) {
		return
	}

	var plan = newUserResourceModel()

	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteUser(ctx, &mongodb.DeleteUserOptions{
		Username: plan.Username.ValueString(),
		Database: plan.Database.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to delete user",
			err.Error(),
		)
	}

	tflog.Trace(ctx, "user deleted")
	resp.State.RemoveResource(ctx)
}

func (r *UserResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	if !r.checkClient(resp.Diagnostics) {
		return
	}

	idParts := strings.Split(req.ID, ".")

	var username, database string

	switch {
	case len(idParts) == 2:
		database = idParts[0]
		username = idParts[1]
	case len(idParts) == 1:
		username = idParts[0]
		database = defaultDatabase
	default:
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: '[<db>.]<username>'. Got: %q", req.ID),
		)

		return
	}

	var plan = newUserResourceModel()

	user, err := r.client.GetUser(ctx, &mongodb.GetUserOptions{
		Username: username,
		Database: database,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get user",
			err.Error(),
		)

		return
	}

	resp.Diagnostics.Append(plan.updateState(ctx, user)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) checkClient(diag diag.Diagnostics) bool {
	if r.client == nil {
		diag.AddError(
			"MongoDB client is not configured",
			"Expected configured MongoDB client. Please report this issue to the provider developers.",
		)

		return false
	}

	return true
}
