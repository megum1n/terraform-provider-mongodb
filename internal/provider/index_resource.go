package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/megum1n/terraform-provider-mongodb/internal/mongodb"
	"go.mongodb.org/mongo-driver/mongo/options"
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

type CollationModel struct {
	Locale          types.String `tfsdk:"locale"`
	CaseLevel       types.Bool   `tfsdk:"case_level"`
	CaseFirst       types.String `tfsdk:"case_first"`
	Strength        types.Int64  `tfsdk:"strength"`
	NumericOrdering types.Bool   `tfsdk:"numeric_ordering"`
	Alternate       types.String `tfsdk:"alternate"`
	MaxVariable     types.String `tfsdk:"max_variable"`
	Backwards       types.Bool   `tfsdk:"backwards"`
}

type IndexResourceModel struct {
	Database                types.String    `tfsdk:"database"`
	Collection              types.String    `tfsdk:"collection"`
	Name                    types.String    `tfsdk:"name"`
	Keys                    types.Set       `tfsdk:"keys"`
	Collation               *CollationModel `tfsdk:"collation"`
	WildcardProjection      types.Map       `tfsdk:"wildcard_projection"`
	PartialFilterExpression types.Map       `tfsdk:"partial_filter_expression"`
	Unique                  types.Bool      `tfsdk:"unique"`
	Sparse                  types.Bool      `tfsdk:"sparse"`
	Hidden                  types.Bool      `tfsdk:"hidden"`
	ExpireAfterSeconds      types.Int64     `tfsdk:"expire_after_seconds"`
	SphereVersion           types.Int64     `tfsdk:"sphere_index_version"`
	Version                 types.Int64     `tfsdk:"version"`
	Bits                    types.Int64     `tfsdk:"bits"`
	Min                     types.Float64   `tfsdk:"min"`
	Max                     types.Float64   `tfsdk:"max"`
	Weights                 types.Map       `tfsdk:"weights"`
	DefaultLanguage         types.String    `tfsdk:"default_language"`
	LanguageOverride        types.String    `tfsdk:"language_override"`
	TextIndexVersion        types.Int64     `tfsdk:"text_index_version"`
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
			"collation": schema.SingleNestedAttribute{
				Description: "Collation settings for string comparison",
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"locale": schema.StringAttribute{
						Description: "The locale for string comparison",
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"case_level": schema.BoolAttribute{
						Description: "Whether to consider case in the 'Level=1' comparison",
						Optional:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
					},
					"case_first": schema.StringAttribute{
						Description: "Whether uppercase or lowercase should sort first",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"strength": schema.Int64Attribute{
						Description: "Comparison level (1-5)",
						Optional:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.RequiresReplace(),
						},
					},
					"numeric_ordering": schema.BoolAttribute{
						Description: "Whether to compare numeric strings as numbers",
						Optional:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
					},
					"alternate": schema.StringAttribute{
						Description: "Whether spaces and punctuation are considered base characters",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"max_variable": schema.StringAttribute{
						Description: "Which characters are affected by 'alternate'",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"backwards": schema.BoolAttribute{
						Description: "Whether to reverse secondary differences",
						Optional:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
					},
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
							Description: "Index type (1, -1, 2dsphere, text, 2d, wildcard)",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								stringvalidator.OneOf("1", "-1", "2dsphere", "text", "2d", "wildcard", "hashed"),
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
			"partial_filter_expression": schema.MapAttribute{
				Description: "Filter expression that limits indexed documents",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"expire_after_seconds": schema.Int64Attribute{
				Description: "TTL in seconds for TTL indexes",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"version": schema.Int64Attribute{
				Description: "The index version number (default: 2)",
				Optional:    true,
				Computed:    true,
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
			"wildcard_projection": schema.MapAttribute{
				Description: "Field inclusion/exclusion for wildcard index (1=include, 0=exclude)",
				Optional:    true,
				ElementType: types.Int64Type,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"hidden": schema.BoolAttribute{
				Description: "Whether the index should be hidden from the query planner",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"bits": schema.Int64Attribute{
				Description: "Number of bits for geospatial index precision",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"min": schema.Float64Attribute{
				Description: "Minimum value for 2d index",
				Optional:    true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.RequiresReplace(),
				},
			},
			"max": schema.Float64Attribute{
				Description: "Maximum value for 2d index",
				Optional:    true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.RequiresReplace(),
				},
			},
			"weights": schema.MapAttribute{
				Description: "Field weights for text index",
				Optional:    true,
				ElementType: types.Int64Type,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"default_language": schema.StringAttribute{
				Description: "Default language for text index",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"language_override": schema.StringAttribute{
				Description: "Field name that contains document language",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"text_index_version": schema.Int64Attribute{
				Description: "Text index version number",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
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

	// Get keys for validation
	var keys []mongodb.IndexKey
	if !config.Keys.IsNull() {
		resp.Diagnostics.Append(config.Keys.ElementsAs(ctx, &keys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Validate TTL index
	if !config.ExpireAfterSeconds.IsNull() {
		isWildcard := false
		for _, key := range keys {
			if key.Type == "wildcard" {
				isWildcard = true
				break
			}
		}

		if isWildcard {
			resp.Diagnostics.AddError(
				"Invalid TTL Index Configuration",
				"TTL index (expire_after_seconds) cannot be used with wildcard indexes")
			return
		}

		// Check for date field
		hasDateField := false
		for _, key := range keys {
			fieldName := strings.ToLower(key.Field)
			if strings.HasSuffix(fieldName, "at") ||
				strings.HasSuffix(fieldName, "date") ||
				strings.HasSuffix(fieldName, "time") {
				hasDateField = true
				break
			}
		}

		if !hasDateField {
			resp.Diagnostics.AddError(
				"Invalid TTL Index Configuration",
				"TTL index (expire_after_seconds) requires a date field")
			return
		}
	}

	// Validate WildcardProjection
	if !config.WildcardProjection.IsNull() {
		var projection map[string]int64
		diags := config.WildcardProjection.ElementsAs(ctx, &projection, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			resp.Diagnostics.AddError(
				"Invalid wildcard projection",
				"Failed to parse wildcard projection data")
			return
		}

		// Validate inclusion/exclusion consistency
		hasInclusion := false
		hasExclusion := false

		for k, v := range projection {
			if v == 1 {
				hasInclusion = true
			} else if v == 0 {
				hasExclusion = true
			} else {
				resp.Diagnostics.AddError(
					"Invalid wildcard projection value",
					fmt.Sprintf("Values must be 1 or 0, got %d for field %q", v, k))
				return
			}
		}

		if hasInclusion && hasExclusion {
			resp.Diagnostics.AddError(
				"Invalid wildcard projection",
				"Cannot mix inclusions (1) and exclusions (0) in wildcard_projection")
			return
		}
	}

	// Validate text index options
	isTextIndex := false
	for _, key := range keys {
		if key.Type == "text" {
			isTextIndex = true
			break
		}
	}

	if isTextIndex {
		// Validate weights map values are positive
		if !config.Weights.IsNull() {
			var weights map[string]int64
			diags := config.Weights.ElementsAs(ctx, &weights, false)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			for field, weight := range weights {
				if weight <= 0 {
					resp.Diagnostics.AddError(
						"Invalid weight value",
						fmt.Sprintf("Weight for field %q must be positive, got: %d", field, weight))
					return
				}
			}
		}

		// Validate text index version if specified
		if !config.TextIndexVersion.IsNull() {
			version := config.TextIndexVersion.ValueInt64()
			if version < 1 || version > 3 {
				resp.Diagnostics.AddError(
					"Invalid text index version",
					"Text index version must be between 1 and 3")
				return
			}
		}
	}

	// Validate PartialFilterExpression
	if !config.PartialFilterExpression.IsNull() {
		var filterExpr map[string]string
		diags := config.PartialFilterExpression.ElementsAs(ctx, &filterExpr, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			resp.Diagnostics.AddError(
				"Error parsing partial filter expression",
				"Failed to parse filter expression data")
			return
		}

		// Validate operators in keys
		validOperators := map[string]bool{
			"$eq": true, "$exists": true, "$gt": true, "$gte": true,
			"$lt": true, "$lte": true, "$type": true, "$and": true,
			"$or": true, "$in": true,
		}

		for k := range filterExpr {
			if strings.Contains(k, ".$") {
				parts := strings.Split(k, ".$")
				if len(parts) > 1 {
					op := "$" + parts[1]
					if !validOperators[op] {
						resp.Diagnostics.AddError(
							"Invalid partial filter expression",
							fmt.Sprintf("Operator %s is not supported. Supported operators: $eq, $exists, $gt, $gte, $lt, $lte, $type, $and, $or, $in", op))
						return
					}
				}
			}
		}
	}
}

// Convert CollationModel to MongoDB options.Collation
func (c *CollationModel) toMongoCollation() *options.Collation {
	if c == nil {
		return nil
	}

	collation := &options.Collation{
		Locale: c.Locale.ValueString(),
	}

	if !c.CaseLevel.IsNull() {
		collation.CaseLevel = c.CaseLevel.ValueBool()
	}
	if !c.CaseFirst.IsNull() {
		collation.CaseFirst = c.CaseFirst.ValueString()
	}
	if !c.Strength.IsNull() {
		collation.Strength = int(c.Strength.ValueInt64())
	}
	if !c.NumericOrdering.IsNull() {
		collation.NumericOrdering = c.NumericOrdering.ValueBool()
	}
	if !c.Alternate.IsNull() {
		collation.Alternate = c.Alternate.ValueString()
	}
	if !c.MaxVariable.IsNull() {
		collation.MaxVariable = c.MaxVariable.ValueString()
	}
	if !c.Backwards.IsNull() {
		collation.Backwards = c.Backwards.ValueBool()
	}

	return collation
}

// Convert string map values to appropriate MongoDB types
func stringMapToMongoTypes(strMap map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range strMap {
		if v == "true" || v == "false" {
			result[k] = v == "true"
		} else if num, err := strconv.ParseInt(v, 10, 64); err == nil {
			result[k] = num
		} else if fnum, err := strconv.ParseFloat(v, 64); err == nil {
			result[k] = fnum
		} else {
			result[k] = v
		}
	}
	return result
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

	// Create the index model
	index := &mongodb.Index{
		Name:       plan.Name.ValueString(),
		Database:   plan.Database.ValueString(),
		Collection: plan.Collection.ValueString(),
		Keys:       keys,
	}

	// Set options based on plan values
	if !plan.Unique.IsNull() {
		index.Options.Unique = plan.Unique.ValueBool()
	}

	if !plan.Sparse.IsNull() {
		index.Options.Sparse = plan.Sparse.ValueBool()
	}

	if !plan.Hidden.IsNull() {
		index.Options.Hidden = plan.Hidden.ValueBool()
	}

	if !plan.ExpireAfterSeconds.IsNull() {
		index.Options.ExpireAfterSeconds = int32(plan.ExpireAfterSeconds.ValueInt64())
	}

	if !plan.SphereVersion.IsNull() {
		index.Options.SphereVersion = int32(plan.SphereVersion.ValueInt64())
	}

	if !plan.Bits.IsNull() {
		index.Options.Bits = int32(plan.Bits.ValueInt64())
	}
	if !plan.Min.IsNull() {
		index.Options.Min = plan.Min.ValueFloat64()
	}
	if !plan.Max.IsNull() {
		index.Options.Max = plan.Max.ValueFloat64()
	}

	// weight DefaultLanguage & LanguageOverride
	if !plan.Weights.IsNull() {
		var weightsInt64 map[string]int64
		diags = plan.Weights.ElementsAs(ctx, &weightsInt64, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		weights := make(map[string]int32)
		for k, v := range weightsInt64 {
			weights[k] = int32(v)
		}
		index.Options.Weights = weights
	}

	if !plan.DefaultLanguage.IsNull() {
		index.Options.DefaultLanguage = plan.DefaultLanguage.ValueString()
	}

	if !plan.LanguageOverride.IsNull() {
		index.Options.LanguageOverride = plan.LanguageOverride.ValueString()
	}

	if !plan.TextIndexVersion.IsNull() {
		index.Options.TextIndexVersion = int32(plan.TextIndexVersion.ValueInt64())
	}

	// Process collation
	if plan.Collation != nil {
		index.Options.Collation = plan.Collation.toMongoCollation()
	}

	// Process wildcard projection
	if !plan.WildcardProjection.IsNull() {
		var projectionInt64 map[string]int64
		diags = plan.WildcardProjection.ElementsAs(ctx, &projectionInt64, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Convert int64 to int32
		projection := make(map[string]int32)
		for k, v := range projectionInt64 {
			projection[k] = int32(v)
		}
		index.Options.WildcardProjection = projection
	}

	// Process partial filter expression
	if !plan.PartialFilterExpression.IsNull() {
		var filterExpr map[string]string
		diags = plan.PartialFilterExpression.ElementsAs(ctx, &filterExpr, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		index.Options.PartialFilterExpression = stringMapToMongoTypes(filterExpr)
	}

	// Create index
	_, err := r.client.CreateIndex(ctx, index)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MongoDB index",
			err.Error(),
		)
		return
	}

	
	plan.Version = types.Int64Value(int64(mongodb.DefaultIndexVersion))

	
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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

	// Update keys
	if index.Keys != nil {
		keysSet, diags := index.Keys.ToTerraformSet(ctx)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Keys = *keysSet
	}

	
	// Handle PartialFilterExpression
	if index.Options.PartialFilterExpression != nil {
		// Convert each value to string for Terraform
		strMap := make(map[string]string)
		for k, v := range index.Options.PartialFilterExpression {
			strMap[k] = fmt.Sprintf("%v", v)
		}

		pfMap, diags := types.MapValueFrom(ctx, types.StringType, strMap)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			state.PartialFilterExpression = pfMap
		}
	} else {
		state.PartialFilterExpression = types.MapNull(types.StringType)
	}

	// Handle Collation
	if index.Options.Collation != nil {
		if state.Collation == nil {
			state.Collation = &CollationModel{}
		}

		state.Collation.Locale = types.StringValue(index.Options.Collation.Locale)
		state.Collation.CaseLevel = types.BoolValue(index.Options.Collation.CaseLevel)
		state.Collation.CaseFirst = types.StringValue(index.Options.Collation.CaseFirst)
		state.Collation.Strength = types.Int64Value(int64(index.Options.Collation.Strength))
		state.Collation.NumericOrdering = types.BoolValue(index.Options.Collation.NumericOrdering)
		state.Collation.Alternate = types.StringValue(index.Options.Collation.Alternate)
		state.Collation.MaxVariable = types.StringValue(index.Options.Collation.MaxVariable)
		state.Collation.Backwards = types.BoolValue(index.Options.Collation.Backwards)
	} else {
		state.Collation = nil
	}

	
	if index.Options.WildcardProjection != nil {
		
		int64Map := make(map[string]int64)
		for k, v := range index.Options.WildcardProjection {
			int64Map[k] = int64(v)
		}

		wpMap, diags := types.MapValueFrom(ctx, types.Int64Type, int64Map)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			state.WildcardProjection = wpMap
		}
	} else {
		state.WildcardProjection = types.MapNull(types.Int64Type)
	}

	// Update 2d index options
	if index.Options.Bits > 0 {
		state.Bits = types.Int64Value(int64(index.Options.Bits))
	} else {
		state.Bits = types.Int64Null()
	}

	if index.Options.Min != 0 {
		state.Min = types.Float64Value(index.Options.Min)
	} else {
		state.Min = types.Float64Null()
	}

	if index.Options.Max != 0 {
		state.Max = types.Float64Value(index.Options.Max)
	} else {
		state.Max = types.Float64Null()
	}

	if index.Options.Weights != nil {
		weights := make(map[string]int64)
		for k, v := range index.Options.Weights {
			weights[k] = int64(v)
		}
		weightMap, diags := types.MapValueFrom(ctx, types.Int64Type, weights)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			state.Weights = weightMap
		}
	}

	if index.Options.DefaultLanguage != "" {
		state.DefaultLanguage = types.StringValue(index.Options.DefaultLanguage)
	} else {
		state.DefaultLanguage = types.StringNull()
	}

	if index.Options.LanguageOverride != "" {
		state.LanguageOverride = types.StringValue(index.Options.LanguageOverride)
	} else {
		state.LanguageOverride = types.StringNull()
	}

	if index.Options.TextIndexVersion > 0 {
		state.TextIndexVersion = types.Int64Value(int64(index.Options.TextIndexVersion))
	} else {
		state.TextIndexVersion = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IndexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Let RequiresReplace handle changes
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
