package mongodb

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IndexKey struct {
	Field string `bson:"field" tfsdk:"field"`
	Type  string `bson:"type"  tfsdk:"type"`
}

const DefaultIndexVersion int32 = 2

type IndexKeys map[string]interface{}

type IndexOptions struct {
	Unique                  bool                   `bson:"unique,omitempty"`
	Sparse                  bool                   `bson:"sparse,omitempty"`
	Hidden                  bool                   `bson:"hidden,omitempty"`
	PartialFilterExpression map[string]interface{} `bson:"partialFilterExpression,omitempty"`
	WildcardProjection      map[string]int32       `bson:"wildcardProjection,omitempty"`
	Collation               *options.Collation     `bson:"collation,omitempty"`
	ExpireAfterSeconds      int32                  `bson:"expireAfterSeconds,omitempty"`
	Version                 int32                  `bson:"v,omitempty"`
	SphereVersion           int32                  `bson:"2dSphereVersion,omitempty"`
	Bits                    int32                  `bson:"bits,omitempty"`
	Min                     float64                `bson:"min,omitempty"`
	Max                     float64                `bson:"max,omitempty"`
	Weights                 map[string]int32       `bson:"weights,omitempty"`
	DefaultLanguage         string                 `bson:"default_language,omitempty"`
	LanguageOverride        string                 `bson:"language_override,omitempty"`
	TextIndexVersion        int32                  `bson:"textIndexVersion,omitempty"`
}

type Index struct {
	Name       string       `bson:"name"`
	Database   string       `bson:"-"` // Not in MongoDB response
	Collection string       `bson:"-"` // Not in MongoDB response
	Keys       IndexKeys    `bson:"key"`
	Options    IndexOptions `bson:"inline"` // Inline embedding
}

func (k IndexKeys) ToTerraformSet(ctx context.Context) (*types.Set, diag.Diagnostics) {
	var keys []basetypes.ObjectValue

	keyType := types.ObjectType{
		AttrTypes: IndexKeyAttributeTypes,
	}

	for field, typeValue := range k {
		key := map[string]attr.Value{
			"field": types.StringValue(field),
			"type":  types.StringValue(fmt.Sprintf("%v", typeValue)),
		}

		keyObj, d := types.ObjectValue(IndexKeyAttributeTypes, key)
		if d.HasError() {
			return nil, d
		}

		keys = append(keys, keyObj)
	}

	keysList, d := types.SetValueFrom(ctx, keyType, keys)
	if d.HasError() {
		return nil, d
	}

	return &keysList, nil
}

func (k IndexKeys) toBson() bson.D {
	out := bson.D{}
	for field, value := range k {
		out = append(out, bson.E{Key: field, Value: value})
	}

	return out
}

var IndexKeyAttributeTypes = map[string]attr.Type{
	"field": types.StringType,
	"type":  types.StringType,
}
