package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/megum1n/terraform-provider-mongodb/internal/mongodb"
)

var (
	_ resource.Resource                   = &IndexResource{}
	_ resource.ResourceWithConfigure      = &IndexResource{}
	_ resource.ResourceWithImportState    = &IndexResource{}
	_ resource.ResourceWithValidateConfig = &IndexResource{}
)

func NewIndexResource() resource.Resource {
	return &IndexResource{}
}

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

func (c CollationModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"locale":           types.StringType,
		"case_level":       types.BoolType,
		"case_first":       types.StringType,
		"strength":         types.Int64Type,
		"numeric_ordering": types.BoolType,
		"alternate":        types.StringType,
		"max_variable":     types.StringType,
		"backwards":        types.BoolType,
	}
}

type IndexResourceModel struct {
	Database                types.String  `tfsdk:"database"`
	Collection              types.String  `tfsdk:"collection"`
	Name                    types.String  `tfsdk:"name"`
	Keys                    types.Map     `tfsdk:"keys"`
	Collation               types.Object  `tfsdk:"collation"`
	WildcardProjection      types.Map     `tfsdk:"wildcard_projection"`
	PartialFilterExpression types.Map     `tfsdk:"partial_filter_expression"`
	Unique                  types.Bool    `tfsdk:"unique"`
	Sparse                  types.Bool    `tfsdk:"sparse"`
	Hidden                  types.Bool    `tfsdk:"hidden"`
	ExpireAfterSeconds      types.Int32   `tfsdk:"expire_after_seconds"`
	SphereVersion           types.Int32   `tfsdk:"sphere_index_version"`
	Bits                    types.Int32   `tfsdk:"bits"`
	Min                     types.Float64 `tfsdk:"min"`
	Max                     types.Float64 `tfsdk:"max"`
	Weights                 types.Map     `tfsdk:"weights"`
	DefaultLanguage         types.String  `tfsdk:"default_language"`
	LanguageOverride        types.String  `tfsdk:"language_override"`
	TextIndexVersion        types.Int32   `tfsdk:"text_index_version"`
}

func (ind *IndexResourceModel) updateState(ctx context.Context, index *mongodb.Index) diag.Diagnostics {
	diags := diag.Diagnostics{}

	ind.Database = types.StringValue(index.Database)
	ind.Collection = types.StringValue(index.Collection)
	ind.Name = types.StringValue(index.Name)

	// Parse keys
	keys, d := types.MapValueFrom(ctx, types.StringType, index.Keys.ToStringMap())

	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	ind.Keys = keys

	// Parse collation
	if index.Options.Collation != nil {
		collation := CollationModel{
			Locale:          types.StringValue(index.Options.Collation.Locale),
			CaseLevel:       types.BoolValue(index.Options.Collation.CaseLevel),
			CaseFirst:       types.StringValue(index.Options.Collation.CaseFirst),
			Strength:        types.Int64Value(int64(index.Options.Collation.Strength)),
			NumericOrdering: types.BoolValue(index.Options.Collation.NumericOrdering),
			Alternate:       types.StringValue(index.Options.Collation.Alternate),
			MaxVariable:     types.StringValue(index.Options.Collation.MaxVariable),
			Backwards:       types.BoolValue(index.Options.Collation.Backwards),
		}

		ind.Collation, d = types.ObjectValueFrom(ctx, collation.AttributeTypes(), collation)
	} else {
		ind.Collation = types.ObjectNull(CollationModel{}.AttributeTypes())
	}

	// Parse wildcard projection
	wildcardProjection, d := types.MapValueFrom(ctx, types.Int32Type, index.Options.WildcardProjection)

	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	ind.WildcardProjection = wildcardProjection

	// Parse partial filter expression
	partialFilterExpression, d := types.MapValueFrom(ctx, types.StringType, index.Options.PartialFilterExpression)

	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	ind.PartialFilterExpression = partialFilterExpression

	// Parse weights
	weights, d := types.MapValueFrom(ctx, types.Int32Type, index.Options.Weights)

	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	ind.Weights = weights

	// Simple data types
	if index.Options.Unique != nil {
		ind.Unique = types.BoolPointerValue(index.Options.Unique)
	}

	if index.Options.Sparse != nil {
		ind.Sparse = types.BoolPointerValue(index.Options.Sparse)
	}

	if index.Options.Hidden != nil {
		ind.Hidden = types.BoolPointerValue(index.Options.Hidden)
	}

	if index.Options.SphereVersion != nil {
		ind.SphereVersion = types.Int32PointerValue(index.Options.SphereVersion)
	}

	if index.Options.Bits != nil {
		ind.Bits = types.Int32PointerValue(index.Options.Bits)
	}

	if index.Options.Min != nil {
		ind.Min = types.Float64PointerValue(index.Options.Min)
	}

	if index.Options.Max != nil {
		ind.Max = types.Float64PointerValue(index.Options.Max)
	}

	if index.Options.TextIndexVersion != nil {
		ind.TextIndexVersion = types.Int32PointerValue(index.Options.TextIndexVersion)
	}

	ind.ExpireAfterSeconds = types.Int32PointerValue(index.Options.ExpireAfterSeconds)
	ind.DefaultLanguage = types.StringPointerValue(index.Options.DefaultLanguage)
	ind.LanguageOverride = types.StringPointerValue(index.Options.LanguageOverride)

	return diags
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
				Computed:    true,
				Default:     objectdefault.StaticValue(types.ObjectNull(CollationModel{}.AttributeTypes())),
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
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
					},
					"case_first": schema.StringAttribute{
						Description: "Whether uppercase or lowercase should sort first",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("off"),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.OneOf("upper", "lower", "off"),
						},
					},
					"strength": schema.Int64Attribute{
						Description: "Comparison level (1-5)",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(3),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.RequiresReplace(),
						},
						Validators: []validator.Int64{
							int64validator.Between(1, 5),
						},
					},
					"numeric_ordering": schema.BoolAttribute{
						Description: "Whether to compare numeric strings as numbers",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
					},
					"alternate": schema.StringAttribute{
						Description: "Whether spaces and punctuation are considered base characters",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("non-ignorable"),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.OneOf("non-ignorable", "shifted"),
						},
					},
					"max_variable": schema.StringAttribute{
						Description: "Which characters are affected by 'alternate'",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("punct"),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.OneOf("punct", "space"),
						},
					},
					"backwards": schema.BoolAttribute{
						Description: "Whether to reverse secondary differences",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"keys": schema.MapAttribute{
				Description: "Index key fields",
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Map{
					mapvalidator.ValueStringsAre(stringvalidator.OneOf("1", "-1", "2d", "2dsphere", "text", "hashed")),
				},
			},
			"unique": schema.BoolAttribute{
				Description: "Whether the index enforces unique values",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"partial_filter_expression": schema.MapAttribute{
				// TODO: Implement proper document support
				Description: "Filter expression that limits indexed documents. Only supports strings.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-zA-Z0-9_.]+(\.\$[a-zA-Z0-9]+)?$`),
							"Valid field name or field with operator",
						),
					),
				},
			},
			"expire_after_seconds": schema.Int32Attribute{
				Description: "TTL in seconds for TTL indexes",
				Optional:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
				},
			},
			"sparse": schema.BoolAttribute{
				Description: "Whether the index should be sparse",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"sphere_index_version": schema.Int32Attribute{
				Description: "The index version number for a 2dsphere index",
				Optional:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int32{
					int32validator.Between(1, 3),
				},
			},
			"wildcard_projection": schema.MapAttribute{
				Description: "Field inclusion/exclusion for wildcard index (1=include, 0=exclude)",
				Optional:    true,
				ElementType: types.Int32Type,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Map{
					mapvalidator.ValueInt32sAre(
						int32validator.OneOf(0, 1),
					),
				},
			},
			"hidden": schema.BoolAttribute{
				Description: "Whether the index should be hidden from the query planner",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"bits": schema.Int32Attribute{
				Description: "Number of bits for geospatial index precision",
				Optional:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int32{
					int32validator.Between(1, 32),
				},
			},
			"min": schema.Float64Attribute{
				Description: "Minimum value for 2d index",
				Optional:    true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Float64{
					float64validator.Between(-180.0, 180.0),
				},
			},
			"max": schema.Float64Attribute{
				Description: "Maximum value for 2d index",
				Optional:    true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Float64{
					float64validator.Between(-180.0, 180.0),
				},
			},
			"weights": schema.MapAttribute{
				Description: "Field weights for text index",
				Optional:    true,
				ElementType: types.Int32Type,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Map{
					mapvalidator.ValueInt32sAre(int32validator.AtLeast(1)),
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
			"text_index_version": schema.Int32Attribute{
				Description: "Text index version number",
				Optional:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int32{
					int32validator.Between(1, 3),
				},
			},
		},
	}
}

func (r *IndexResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var config IndexResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Keys.IsNull() || config.Keys.IsUnknown() {
		return
	}

	var keysMap map[string]string
	resp.Diagnostics.Append(config.Keys.ElementsAs(ctx, &keysMap, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.ExpireAfterSeconds.IsNull() {
		isWildcard := false
		if _, exists := keysMap["$**"]; exists {
			isWildcard = true
		}

		if isWildcard {
			resp.Diagnostics.AddError(
				"Invalid TTL Index Configuration",
				"TTL index (expire_after_seconds) cannot be used with wildcard indexes")

			return
		}
	}

	// Validate partial filter expression operators
	if config.PartialFilterExpression.IsNull() {
		return
	}

	var filterExpr map[string]string

	diags := config.PartialFilterExpression.ElementsAs(ctx, &filterExpr, false)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		resp.Diagnostics.AddError(
			"Error parsing partial filter expression",
			"Failed to parse filter expression data")

		return
	}

	validOperators := map[string]bool{
		"$eq": true, "$exists": true, "$gt": true, "$gte": true,
		"$lt": true, "$lte": true, "$type": true, "$and": true,
		"$or": true, "$in": true,
	}

	for k := range filterExpr {
		if !strings.Contains(k, ".$") {
			continue
		}

		parts := strings.Split(k, ".$")
		if len(parts) <= 1 {
			continue
		}

		op := "$" + parts[1]
		if !validOperators[op] {
			resp.Diagnostics.AddError(
				"Invalid partial filter expression",
				fmt.Sprintf("Operator %s is not supported. "+
					"Supported operators: $eq, $exists, $gt, $gte, $lt, $lte, $type, $and, $or, $in", op))

			return
		}
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

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	index := &mongodb.Index{
		Database:   plan.Database.ValueString(),
		Collection: plan.Collection.ValueString(),
		Name:       plan.Name.ValueString(),

		Options: mongodb.IndexOptions{
			Unique:             plan.Unique.ValueBoolPointer(),
			Sparse:             plan.Sparse.ValueBoolPointer(),
			Hidden:             plan.Hidden.ValueBoolPointer(),
			ExpireAfterSeconds: plan.ExpireAfterSeconds.ValueInt32Pointer(),
			SphereVersion:      plan.SphereVersion.ValueInt32Pointer(),
			Bits:               plan.Bits.ValueInt32Pointer(),
			Min:                plan.Min.ValueFloat64Pointer(),
			Max:                plan.Max.ValueFloat64Pointer(),
			DefaultLanguage:    plan.DefaultLanguage.ValueStringPointer(),
			LanguageOverride:   plan.LanguageOverride.ValueStringPointer(),
			TextIndexVersion:   plan.TextIndexVersion.ValueInt32Pointer(),
		},
	}

	if !plan.Collation.IsNull() && !plan.Collation.IsUnknown() {
		collation := &CollationModel{}
		resp.Diagnostics.Append(plan.Collation.As(ctx, collation, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		index.Options.Collation = &options.Collation{
			Locale:          collation.Locale.ValueString(),
			CaseLevel:       collation.CaseLevel.ValueBool(),
			CaseFirst:       collation.CaseFirst.ValueString(),
			Strength:        int(collation.Strength.ValueInt64()),
			NumericOrdering: collation.NumericOrdering.ValueBool(),
			Alternate:       collation.Alternate.ValueString(),
			MaxVariable:     collation.MaxVariable.ValueString(),
			Backwards:       collation.Backwards.ValueBool(),
		}

	}

	// Parse keys
	if !plan.Keys.IsNull() && !plan.Keys.IsUnknown() {
		indexKeys := map[string]string{}
		resp.Diagnostics.Append(plan.Keys.ElementsAs(ctx, &indexKeys, false)...)

		if resp.Diagnostics.HasError() {
			return
		}

		index.Keys = mongodb.ConvertMap(indexKeys, true)
	}

	// Parse WildcardProjection
	if !plan.WildcardProjection.IsNull() && !plan.WildcardProjection.IsUnknown() {
		wildcardProjection := make(map[string]int32)
		resp.Diagnostics.Append(plan.WildcardProjection.ElementsAs(ctx, &wildcardProjection, false)...)

		if resp.Diagnostics.HasError() {
			return
		}

		index.Options.WildcardProjection = wildcardProjection
	}

	// Parse PartialFilterExpression
	if !plan.PartialFilterExpression.IsNull() && !plan.PartialFilterExpression.IsUnknown() {
		partialFilterExpression := make(map[string]string)
		resp.Diagnostics.Append(plan.PartialFilterExpression.ElementsAs(ctx, &partialFilterExpression, false)...)

		if resp.Diagnostics.HasError() {
			return
		}

		index.Options.PartialFilterExpression = mongodb.ConvertMap(partialFilterExpression, false)
	}

	// Parse Weights
	if !plan.Weights.IsNull() && !plan.Weights.IsUnknown() {
		weights := make(map[string]int32)
		resp.Diagnostics.Append(plan.Weights.ElementsAs(ctx, &weights, false)...)

		if resp.Diagnostics.HasError() {
			return
		}

		index.Options.Weights = weights
	}

	dbIndex, err := r.client.CreateIndex(ctx, index)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MongoDB index",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(plan.updateState(ctx, dbIndex)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IndexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.checkClient(resp.Diagnostics) {
		return
	}

	var plan IndexResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	index, err := r.client.GetIndex(ctx, &mongodb.GetIndexOptions{
		Name:       plan.Name.ValueString(),
		Database:   plan.Database.ValueString(),
		Collection: plan.Collection.ValueString(),
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

	// Use the helper function to set state
	resp.Diagnostics.Append(plan.updateState(ctx, index)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IndexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// MongoDB indexes are immutable, so just setting the plan as the new state
	var plan IndexResourceModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IndexResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.checkClient(resp.Diagnostics) {
		return
	}

	var plan IndexResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteIndex(ctx, &mongodb.GetIndexOptions{
		Name:       plan.Name.ValueString(),
		Database:   plan.Database.ValueString(),
		Collection: plan.Collection.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting MongoDB index",
			err.Error(),
		)
	}

	tflog.Trace(ctx, "Index deleted")
	resp.State.RemoveResource(ctx)
}

func (r *IndexResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, ".")
	if len(idParts) < 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID should be in the format: database.collection.index_name",
		)

		return
	}

	database := idParts[0]
	collection := idParts[1]
	indexName := strings.Join(idParts[2:], ".")

	var plan IndexResourceModel

	index, err := r.client.GetIndex(ctx, &mongodb.GetIndexOptions{
		Name:       indexName,
		Database:   database,
		Collection: collection,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing index",
			fmt.Sprintf("Failed to read index %s: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(plan.updateState(ctx, index)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
