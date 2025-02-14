package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/megum1n/terraform-provider-mongodb/internal/mongodb"
)

var (
	_ resource.Resource                   = &IndexResource{}
	_ resource.ResourceWithConfigure      = &IndexResource{}
	_ resource.ResourceWithImportState    = &IndexResource{}
	_ resource.ResourceWithValidateConfig = &IndexResource{}
)

type IndexResource struct {
	client *mongodb.Client
}

type IndexResourceModel struct {
	Database                types.String `tfsdk:"database"`
	Collection              types.String `tfsdk:"collection"`
	Name                    types.String `tfsdk:"name"`
	Keys                    types.Set    `tfsdk:"keys"`
	Unique                  types.Bool   `tfsdk:"unique"`
	ExpireAfterSeconds      types.Int64  `tfsdk:"expire_after_seconds"`
	Sparse                  types.Bool   `tfsdk:"sparse"`
	Hidden                  types.Bool   `tfsdk:"hidden"`
	PartialFilterExpression types.String `tfsdk:"partial_filter_expression"`
	SphereIndexVersion      types.Int64  `tfsdk:"sphere_index_version"`
	WildcardProjection      types.String `tfsdk:"wildcard_projection"`
	Collation               types.String `tfsdk:"collation"`
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
			"collation": schema.StringAttribute{
				Description: "Collation settings for string comparison as JSON string. Required field: locale. " +
					"Optional fields: caseLevel, caseFirst, strength, numericOrdering, alternate, maxVariable, backwards.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"keys": schema.SetNestedAttribute{
				Description: "Index key fields",
				Required:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"field": schema.StringAttribute{
							Description: "Field name",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"type": schema.StringAttribute{
							Description: "Index type (1, -1, 2dsphere, text)",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								stringvalidator.OneOf("1", "-1", "2dsphere", "text"),
							},
						},
					},
				},
			},
			"unique": schema.BoolAttribute{
				Description: "Whether the index enforces unique values",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"partial_filter_expression": schema.StringAttribute{
				Description: "Partial filter expression for the index as a JSON string. " +
					"The index only references documents that match this expression. " +
					"Supported expressions include: " +
					"equality expressions (field: value or $eq), " +
					"$exists: true, " +
					"$gt, $gte, $lt, $lte, " +
					"$type, " +
					"$and, " +
					"$or, " +
					"$in",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"expire_after_seconds": schema.Int64Attribute{
				Description: "TTL in seconds for TTL indexes",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"sparse": schema.BoolAttribute{
				Description: "Whether the index should be sparse",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"sphere_index_version": schema.Int64Attribute{
				Description: "The index version number for a 2dsphere index",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"wildcard_projection": schema.StringAttribute{
				Description: "JSON string defining field inclusion/exclusion for wildcard index. Format: " +
					"{\"field1\": 1|0, \"field2\": 1|0}. 1 to include, 0 to exclude.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hidden": schema.BoolAttribute{
				Description: "Whether the index should be hidden from the query planner",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
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

func (r *IndexResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config IndexResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.ExpireAfterSeconds.IsNull() {
		var keys []mongodb.IndexKey
		resp.Diagnostics.Append(config.Keys.ElementsAs(ctx, &keys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Check if any key is a date field
		hasDateField := false
		for _, key := range keys {
			if strings.HasSuffix(strings.ToLower(key.Field), "at") ||
				strings.HasSuffix(strings.ToLower(key.Field), "date") ||
				strings.HasSuffix(strings.ToLower(key.Field), "time") {
				hasDateField = true
				break
			}
		}

		if !hasDateField {
			resp.Diagnostics.AddError(
				"Invalid TTL Index Configuration",
				"TTL index (expire_after_seconds) requires a date field",
			)
		}
	}

	if !config.WildcardProjection.IsNull() {
		var projection map[string]interface{}
		if err := json.Unmarshal([]byte(config.WildcardProjection.ValueString()), &projection); err != nil {
			resp.Diagnostics.AddError(
				"Invalid wildcard projection",
				fmt.Sprintf("Invalid JSON in wildcard_projection: %v", err),
			)
			return
		}

		// Check for mixing inclusion/exclusion
		hasInclusion := false
		hasExclusion := false
		for _, v := range projection {
			if val, ok := v.(float64); ok {
				if val == 1 {
					hasInclusion = true
				} else if val == 0 {
					hasExclusion = true
				}
			}
		}
		if hasInclusion && hasExclusion {
			resp.Diagnostics.AddError(
				"Invalid wildcard projection",
				"Cannot mix inclusions (1) and exclusions (0) in wildcard_projection",
			)
			return
		}
	}

	if !config.Collation.IsNull() {
		var collation map[string]interface{}
		if err := json.Unmarshal([]byte(config.Collation.ValueString()), &collation); err != nil {
			resp.Diagnostics.AddError(
				"Invalid collation",
				fmt.Sprintf("Invalid JSON in collation: %v", err),
			)
			return
		}

		if _, ok := collation["locale"].(string); !ok {
			resp.Diagnostics.AddError(
				"Invalid collation",
				"Collation must include 'locale' field as string",
			)
			return
		}
	}

	if !config.PartialFilterExpression.IsNull() {
		var filterExpr map[string]interface{}
		if err := json.Unmarshal([]byte(config.PartialFilterExpression.ValueString()), &filterExpr); err != nil {
			resp.Diagnostics.AddError(
				"Error parsing partial filter expression",
				fmt.Sprintf("Invalid JSON in partial_filter_expression: %v", err),
			)
			return
		}

		validOperators := map[string]bool{
			"$eq":     true,
			"$exists": true,
			"$gt":     true,
			"$gte":    true,
			"$lt":     true,
			"$lte":    true,
			"$type":   true,
			"$and":    true,
			"$or":     true,
			"$in":     true,
		}

		var checkOperators func(v interface{}) bool
		checkOperators = func(v interface{}) bool {
			switch val := v.(type) {
			case map[string]interface{}:
				for k, v := range val {
					if strings.HasPrefix(k, "$") {
						if !validOperators[k] {
							resp.Diagnostics.AddError(
								"Invalid partial filter expression",
								fmt.Sprintf("Operator %s is not supported. Supported operators are: $eq, $exists, $gt, $gte, $lt, $lte, $type, $and, $or, $in", k),
							)
							return false
						}
					}
					if !checkOperators(v) {
						return false
					}
				}
			case []interface{}:
				for _, item := range val {
					if !checkOperators(item) {
						return false
					}
				}
			}
			return true
		}

		checkOperators(filterExpr)
	}
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

	state := IndexResourceModel{
		Name:                    plan.Name,
		Database:                plan.Database,
		Collection:              plan.Collection,
		Keys:                    plan.Keys,
		PartialFilterExpression: plan.PartialFilterExpression,
	}

	if !plan.Unique.IsNull() {
		state.Unique = plan.Unique
	}
	if !plan.ExpireAfterSeconds.IsNull() {
		state.ExpireAfterSeconds = plan.ExpireAfterSeconds
	}
	if !plan.Sparse.IsNull() {
		state.Sparse = plan.Sparse
	}
	if !plan.Hidden.IsNull() {
		state.Hidden = plan.Hidden
	}
	if !plan.Collation.IsNull() {
		state.Collation = plan.Collation // Preserve the collation state
	}

	index := &mongodb.Index{
		Name:       plan.Name.ValueString(),
		Database:   plan.Database.ValueString(),
		Collection: plan.Collection.ValueString(),
		Keys:       keys,
	}

	if !plan.Unique.IsNull() {
		unique := plan.Unique.ValueBool()
		index.Options.Unique = &unique
	}
	if !plan.ExpireAfterSeconds.IsNull() {
		eas := int32(plan.ExpireAfterSeconds.ValueInt64())
		index.Options.ExpireAfterSeconds = &eas
	}
	if !plan.Sparse.IsNull() {
		sparse := plan.Sparse.ValueBool()
		index.Options.Sparse = &sparse
	}
	if !plan.Hidden.IsNull() {
		hidden := plan.Hidden.ValueBool()
		index.Options.Hidden = &hidden
	}
	if !plan.SphereIndexVersion.IsNull() {
		version := int32(plan.SphereIndexVersion.ValueInt64())
		index.Options.SphereIndexVersion = &version
	}
	if !plan.Collation.IsNull() {
		var collation map[string]interface{}
		json.Unmarshal([]byte(plan.Collation.ValueString()), &collation)
		index.Options.Collation = collation
	}
	if !plan.WildcardProjection.IsNull() {
		var projection map[string]interface{}
		json.Unmarshal([]byte(plan.WildcardProjection.ValueString()), &projection)
		index.Options.WildcardProjection = projection

		state.WildcardProjection = plan.WildcardProjection
	}

	_, err := r.client.CreateIndex(ctx, index)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MongoDB index",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IndexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.checkClient(resp.Diagnostics) {
		return
	}

	var state IndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	index, err := r.client.GetIndex(ctx, &mongodb.GetIndexOptions{
		Name:       state.Name.ValueString(),
		Database:   state.Database.ValueString(),
		Collection: state.Collection.ValueString(),
	})
	if err != nil {
		if errors.As(err, &mongodb.NotFoundError{}) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading MongoDB index",
			err.Error(),
		)
		return
	}

	if index.Keys == nil {
		return
	}

	var currentKeys []mongodb.IndexKey
	diags := state.Keys.ElementsAs(ctx, &currentKeys, false)
	resp.Diagnostics.Append(diags...)

	tflog.Debug(ctx, "Current vs MongoDB Keys", map[string]interface{}{
		"current_keys": currentKeys,
		"mongo_keys":   index.Keys,
	})

	if !reflect.DeepEqual(currentKeys, index.Keys) {
		keysSet, diags := index.Keys.ToTerraformSet(ctx)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Keys = *keysSet
	}

	if index.Options.Unique != nil && (state.Unique.IsNull() || state.Unique.ValueBool() != *index.Options.Unique) {
		state.Unique = types.BoolValue(*index.Options.Unique)
	}

	if index.Options.PartialFilterExpression != nil {
		if exprBytes, err := json.Marshal(index.Options.PartialFilterExpression); err == nil {
			state.PartialFilterExpression = types.StringValue(string(exprBytes))
		} else {
			state.PartialFilterExpression = types.StringNull()
		}
	} else {
		state.PartialFilterExpression = types.StringNull()
	}

	if index.Options.Collation != nil {
		if collationBytes, err := json.Marshal(index.Options.Collation); err == nil {
			state.Collation = types.StringValue(string(collationBytes))
		} else {
			state.Collation = types.StringNull()
		}
	} else {
		state.Collation = types.StringNull()
	}

	if index.Options.ExpireAfterSeconds != nil && (state.ExpireAfterSeconds.IsNull() || state.ExpireAfterSeconds.ValueInt64() != int64(*index.Options.ExpireAfterSeconds)) {
		state.ExpireAfterSeconds = types.Int64Value(int64(*index.Options.ExpireAfterSeconds))
	}

	if index.Options.Sparse != nil && (state.Sparse.IsNull() || state.Sparse.ValueBool() != *index.Options.Sparse) {
		state.Sparse = types.BoolValue(*index.Options.Sparse)
	}

	if index.Options.SphereIndexVersion != nil {
		state.SphereIndexVersion = types.Int64Value(int64(*index.Options.SphereIndexVersion))
	} else {
		state.SphereIndexVersion = types.Int64Null()
	}

	if index.Options.WildcardProjection != nil {
		if projBytes, err := json.Marshal(index.Options.WildcardProjection); err == nil {
			state.WildcardProjection = types.StringValue(string(projBytes))
		} else {
			state.WildcardProjection = types.StringNull()
		}
	} else {
		state.WildcardProjection = types.StringNull()
	}

	if index.Options.Hidden != nil && (state.Hidden.IsNull() || state.Hidden.ValueBool() != *index.Options.Hidden) {
		state.Hidden = types.BoolValue(*index.Options.Hidden)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IndexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, req.Plan)...)
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
	}
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
