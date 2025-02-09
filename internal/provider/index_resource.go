package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/megum1n/terraform-provider-mongodb/internal/mongodb"
)

var (
	_ resource.Resource                = &IndexResource{}
	_ resource.ResourceWithConfigure   = &IndexResource{}
	_ resource.ResourceWithImportState = &IndexResource{}
)

type IndexResource struct {
	client *mongodb.Client
}

type IndexResourceModel struct {
	Database           types.String `tfsdk:"database"`
	Collection         types.String `tfsdk:"collection"`
	Name               types.String `tfsdk:"name"`
	Keys               types.Set    `tfsdk:"keys"`
	Unique             types.Bool   `tfsdk:"unique"`
	ExpireAfterSeconds types.Int64  `tfsdk:"expire_after_seconds"`
}

func NewIndexResource() resource.Resource {
	return &IndexResource{}
}

func (r *IndexResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_index"
}

func (r *IndexResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages MongoDB indexes",
		Attributes: map[string]schema.Attribute{
			"database": schema.StringAttribute{
				Description: "Database name",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"collection": schema.StringAttribute{
				Description: "Collection name",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Index name",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"keys": schema.SetNestedAttribute{
				Description: "Index key fields",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"field": schema.StringAttribute{
							Description: "Field name",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "Index type (1 for ascending, -1 for descending)",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("1", "-1"),
							},
						},
					},
				},
			},
			"unique": schema.BoolAttribute{
				Description: "Whether the index should enforce uniqueness",
				Optional:    true,
			},
			"expire_after_seconds": schema.Int64Attribute{
				Description: "TTL in seconds for TTL indexes",
				Optional:    true,
			},
		},
	}
}

func (r *IndexResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	p, ok := req.ProviderData.(*MongodbProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *MongodbProvider, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = p.client
}

func (r *IndexResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.checkClient(resp.Diagnostics) {
		return
	}

	var plan IndexResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert keys
	var keys []mongodb.IndexKey
	diags = plan.Keys.ElementsAs(ctx, &keys, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	index := &mongodb.Index{
		Name:       plan.Name.ValueString(),
		Database:   plan.Database.ValueString(),
		Collection: plan.Collection.ValueString(),
		Keys:       keys,
	}

	if !plan.Unique.IsNull() {
		unique := plan.Unique.ValueBool()
		index.Unique = &unique
	}

	if !plan.ExpireAfterSeconds.IsNull() {
		eas := int32(plan.ExpireAfterSeconds.ValueInt64())
		index.ExpireAfterSeconds = &eas
	}

	_, err := r.client.CreateIndex(ctx, index)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MongoDB index",
			err.Error(),
		)
		return
	}

	// Create a new set value for keys
	keyType := types.ObjectType{
		AttrTypes: mongodb.IndexKeyAttributeTypes,
	}
	keySet, diags := types.SetValueFrom(ctx, keyType, keys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update plan with the created keys
	plan.Keys = keySet
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *IndexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.checkClient(resp.Diagnostics) {
		return
	}

	var state IndexResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	index, err := r.client.GetIndex(ctx, &mongodb.GetIndexOptions{
		Name:       state.Name.ValueString(),
		Database:   state.Database.ValueString(),
		Collection: state.Collection.ValueString(),
	})
	if err != nil {
		if _, ok := err.(mongodb.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading MongoDB index",
			err.Error(),
		)
		return
	}

	// Update state
	state.Name = types.StringValue(index.Name)
	state.Database = types.StringValue(index.Database)
	state.Collection = types.StringValue(index.Collection)

	// Convert and set keys
	keysSet, diags := index.Keys.ToTerraformSet(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Keys = *keysSet

	// Update properties
	if index.Unique != nil {
		state.Unique = types.BoolValue(*index.Unique)
	} else {
		state.Unique = types.BoolNull()
	}

	if index.ExpireAfterSeconds != nil {
		state.ExpireAfterSeconds = types.Int64Value(int64(*index.ExpireAfterSeconds))
	} else {
		state.ExpireAfterSeconds = types.Int64Null()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Read index %s.%s.%s",
		state.Database.ValueString(),
		state.Collection.ValueString(),
		state.Name.ValueString(),
	))
}

func (r *IndexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// MongoDB indexes cannot be updated, only dropped and recreated
	resp.Diagnostics.AddError(
		"Update not supported",
		"MongoDB indexes cannot be updated directly. Delete and recreate the index instead.",
	)
}

func (r *IndexResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.checkClient(resp.Diagnostics) {
		return
	}

	var state IndexResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteIndex(ctx, &mongodb.GetIndexOptions{
		Name:       state.Name.ValueString(),
		Database:   state.Database.ValueString(),
		Collection: state.Collection.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting MongoDB index",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Deleted index %s.%s.%s",
		state.Database.ValueString(),
		state.Collection.ValueString(),
		state.Name.ValueString(),
	))
}

func (r *IndexResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ".")
	if len(idParts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID should be in the format: database.collection.index_name",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("collection"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[2])...)
}

func (r *IndexResource) checkClient(diag diag.Diagnostics) bool {
	if r.client == nil {
		diag.AddError(
			"MongoDB client is not configured",
			"Expected configured MongoDB client. Please report this issue to the provider developers.",
		)
		return false
	}
	return true
}
