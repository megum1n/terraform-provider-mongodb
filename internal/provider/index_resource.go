package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	Keys                    types.Map       `tfsdk:"keys"`
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
			"keys": schema.MapAttribute{
				Description: "Index key fields",
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
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
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-zA-Z0-9_\.]+(\.\$[a-zA-Z0-9]+)?$`),
							"Valid field name or field with operator",
						),
					),
				},
			},
			"expire_after_seconds": schema.Int64Attribute{
				Description: "TTL in seconds for TTL indexes",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
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
				Validators: []validator.Map{
					mapvalidator.ValueInt64sAre(
						int64validator.OneOf(0, 1),
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
				Validators: []validator.Map{
					mapvalidator.ValueInt64sAre(int64validator.AtLeast(1)),
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
				Validators: []validator.Int64{
					int64validator.Between(1, 3),
				},
			},
		},
	}
}

func (r *IndexResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config IndexResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Keys.IsNull() || config.Keys.IsUnknown() {
		return
	}

	var keysMap map[string]string
	if !config.Keys.IsNull() {
		resp.Diagnostics.Append(config.Keys.ElementsAs(ctx, &keysMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
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

	// Safely handle Keys map
	indexKeys := make(mongodb.IndexKeys)
	if !plan.Keys.IsNull() && !plan.Keys.IsUnknown() {
		var keysMap map[string]string
		diags = plan.Keys.ElementsAs(ctx, &keysMap, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for field, typeStr := range keysMap {
			if field == "$**" && typeStr == "wildcard" {
				indexKeys[field] = 1
			} else {
				var value interface{}
				switch typeStr {
				case "1", "asc":
					value = 1
				case "-1", "desc":
					value = -1
				case "2d", "2dsphere", "text", "hashed":
					value = typeStr
				default:
					value = typeStr
				}
				indexKeys[field] = value
			}
		}
	}

	index := &mongodb.Index{
		Name:       plan.Name.ValueString(),
		Database:   plan.Database.ValueString(),
		Collection: plan.Collection.ValueString(),
		Keys:       indexKeys,
	}

	if !plan.Unique.IsNull() && !plan.Unique.IsUnknown() {
		index.Options.Unique = plan.Unique.ValueBool()
	}

	if !plan.Sparse.IsNull() && !plan.Sparse.IsUnknown() {
		index.Options.Sparse = plan.Sparse.ValueBool()
	}

	if !plan.Hidden.IsNull() && !plan.Hidden.IsUnknown() {
		index.Options.Hidden = plan.Hidden.ValueBool()
	}

	if !plan.ExpireAfterSeconds.IsNull() && !plan.ExpireAfterSeconds.IsUnknown() {
		index.Options.ExpireAfterSeconds = int32(plan.ExpireAfterSeconds.ValueInt64())
	}

	if !plan.SphereVersion.IsNull() && !plan.SphereVersion.IsUnknown() {
		index.Options.SphereVersion = int32(plan.SphereVersion.ValueInt64())
	}

	if !plan.Bits.IsNull() && !plan.Bits.IsUnknown() {
		index.Options.Bits = int32(plan.Bits.ValueInt64())
	}

	if !plan.Min.IsNull() && !plan.Min.IsUnknown() {
		index.Options.Min = plan.Min.ValueFloat64()
	}

	if !plan.Max.IsNull() && !plan.Max.IsUnknown() {
		index.Options.Max = plan.Max.ValueFloat64()
	}

	if !plan.Weights.IsNull() && !plan.Weights.IsUnknown() {
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

	if !plan.DefaultLanguage.IsNull() && !plan.DefaultLanguage.IsUnknown() {
		index.Options.DefaultLanguage = plan.DefaultLanguage.ValueString()
	}

	if !plan.LanguageOverride.IsNull() && !plan.LanguageOverride.IsUnknown() {
		index.Options.LanguageOverride = plan.LanguageOverride.ValueString()
	}

	if !plan.TextIndexVersion.IsNull() && !plan.TextIndexVersion.IsUnknown() {
		index.Options.TextIndexVersion = int32(plan.TextIndexVersion.ValueInt64())
	}

	if plan.Collation != nil {
		index.Options.Collation = plan.Collation.toMongoCollation()
	}

	if !plan.WildcardProjection.IsNull() && !plan.WildcardProjection.IsUnknown() {
		var projectionInt64 map[string]int64
		diags = plan.WildcardProjection.ElementsAs(ctx, &projectionInt64, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		projection := make(map[string]int32)
		for k, v := range projectionInt64 {
			projection[k] = int32(v)
		}
		index.Options.WildcardProjection = projection
	}

	if !plan.PartialFilterExpression.IsNull() && !plan.PartialFilterExpression.IsUnknown() {
		var filterExpr map[string]string
		diags = plan.PartialFilterExpression.ElementsAs(ctx, &filterExpr, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		index.Options.PartialFilterExpression = stringMapToMongoTypes(filterExpr)
	}

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

func (r *IndexResource) setStateFromIndex(ctx context.Context, state *IndexResourceModel, index *mongodb.Index) diag.Diagnostics {
	diags := diag.Diagnostics{}

	state.Database = types.StringValue(index.Database)
	state.Collection = types.StringValue(index.Collection)
	state.Name = types.StringValue(index.Name)

	keysMap := make(map[string]string)
	for field, value := range index.Keys {
		var typeStr string
		switch v := value.(type) {
		case int, int32, int64:
			if field == "$**" && (v == 1 || v == int64(1) || v == int32(1)) {
				typeStr = "wildcard"
			} else if v == 1 || v == int64(1) || v == int32(1) {
				typeStr = "1"
			} else if v == -1 || v == int64(-1) || v == int32(-1) {
				typeStr = "-1"
			} else {
				typeStr = fmt.Sprintf("%v", v)
			}
		case string:
			typeStr = v
		default:
			typeStr = fmt.Sprintf("%v", v)
		}
		keysMap[field] = typeStr
	}

	keysValue, d := types.MapValueFrom(ctx, types.StringType, keysMap)
	diags.Append(d...)
	if !diags.HasError() {
		state.Keys = keysValue
	}

	if len(index.Options.PartialFilterExpression) > 0 {
		strMap := make(map[string]string)
		for k, v := range index.Options.PartialFilterExpression {
			strMap[k] = fmt.Sprintf("%v", v)
		}

		pfMap, d := types.MapValueFrom(ctx, types.StringType, strMap)
		diags.Append(d...)
		if !diags.HasError() {
			state.PartialFilterExpression = pfMap
		}
	} else {
		state.PartialFilterExpression = types.MapNull(types.StringType)
	}

	if !state.Unique.IsNull() || index.Options.Unique {
		state.Unique = types.BoolValue(index.Options.Unique)
	} else {
		state.Unique = types.BoolNull()
	}

	if !state.Sparse.IsNull() || index.Options.Sparse {
		state.Sparse = types.BoolValue(index.Options.Sparse)
	} else {
		state.Sparse = types.BoolNull()
	}

	if !state.Hidden.IsNull() || index.Options.Hidden {
		state.Hidden = types.BoolValue(index.Options.Hidden)
	} else {
		state.Hidden = types.BoolNull()
	}

	if index.Options.Collation != nil {
		state.Collation = &CollationModel{
			Locale:          types.StringValue(index.Options.Collation.Locale),
			CaseLevel:       types.BoolValue(index.Options.Collation.CaseLevel),
			CaseFirst:       types.StringValue(index.Options.Collation.CaseFirst),
			Strength:        types.Int64Value(int64(index.Options.Collation.Strength)),
			NumericOrdering: types.BoolValue(index.Options.Collation.NumericOrdering),
			Alternate:       types.StringValue(index.Options.Collation.Alternate),
			MaxVariable:     types.StringValue(index.Options.Collation.MaxVariable),
			Backwards:       types.BoolValue(index.Options.Collation.Backwards),
		}
	} else {
		state.Collation = nil
	}

	if len(index.Options.WildcardProjection) > 0 {
		int64Map := make(map[string]int64)
		for k, v := range index.Options.WildcardProjection {
			int64Map[k] = int64(v)
		}

		wpMap, d := types.MapValueFrom(ctx, types.Int64Type, int64Map)
		diags.Append(d...)
		if !diags.HasError() {
			state.WildcardProjection = wpMap
		}
	} else {
		state.WildcardProjection = types.MapNull(types.Int64Type)
	}

	if index.Options.ExpireAfterSeconds > 0 {
		state.ExpireAfterSeconds = types.Int64Value(int64(index.Options.ExpireAfterSeconds))
	} else {
		state.ExpireAfterSeconds = types.Int64Null()
	}

	state.Version = types.Int64Value(int64(index.Options.Version))

	if index.Options.SphereVersion > 0 {
		state.SphereVersion = types.Int64Value(int64(index.Options.SphereVersion))
	} else {
		state.SphereVersion = types.Int64Null()
	}

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

	if len(index.Options.Weights) > 0 {
		weights := make(map[string]int64)
		for k, v := range index.Options.Weights {
			weights[k] = int64(v)
		}
		weightMap, d := types.MapValueFrom(ctx, types.Int64Type, weights)
		diags.Append(d...)
		if !diags.HasError() {
			state.Weights = weightMap
		}
	} else {
		state.Weights = types.MapNull(types.Int64Type)
	}

	state.DefaultLanguage = types.StringValue(index.Options.DefaultLanguage)
	if index.Options.DefaultLanguage == "" {
		state.DefaultLanguage = types.StringNull()
	}

	state.LanguageOverride = types.StringValue(index.Options.LanguageOverride)
	if index.Options.LanguageOverride == "" {
		state.LanguageOverride = types.StringNull()
	}

	state.TextIndexVersion = types.Int64Value(int64(index.Options.TextIndexVersion))
	if index.Options.TextIndexVersion == 0 {
		state.TextIndexVersion = types.Int64Null()
	}

	return diags
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

	// Use the helper function to set state
	resp.Diagnostics.Append(r.setStateFromIndex(ctx, &state, index)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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

	database := idParts[0]
	collection := idParts[1]
	indexName := idParts[2]

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

	var state IndexResourceModel

	// Use the helper function to set state
	resp.Diagnostics.Append(r.setStateFromIndex(ctx, &state, index)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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
	tflog.Trace(ctx, "Index deleted")
	resp.State.RemoveResource(ctx)
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
