package mongodb

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"go.mongodb.org/mongo-driver/bson"
)

type IndexKey struct {
	Field string `bson:"field" tfsdk:"field"`
	Type  string `bson:"type"  tfsdk:"type"`
}

type IndexKeys []IndexKey

type IndexOptions struct {
	Unique                  *bool       `bson:"unique,omitempty" tfsdk:"unique"`
	Sparse                  *bool       `bson:"sparse,omitempty" tfsdk:"sparse"`
	ExpireAfterSeconds      *int32      `bson:"expireAfterSeconds,omitempty" tfsdk:"expire_after_seconds"`
	Hidden                  *bool       `bson:"hidden,omitempty" tfsdk:"hidden"`
	StorageEngine           interface{} `bson:"storageEngine,omitempty" tfsdk:"storage_engine"`
	Collation               interface{} `bson:"collation,omitempty" tfsdk:"collation"`
	PartialFilterExpression interface{} `bson:"partialFilterExpression,omitempty" tfsdk:"partial_filter_expression"`
	Weights                 interface{} `bson:"weights,omitempty" tfsdk:"weights"`
	DefaultLanguage         *string     `bson:"default_language,omitempty" tfsdk:"default_language"`
	LanguageOverride        *string     `bson:"language_override,omitempty" tfsdk:"language_override"`
	TextIndexVersion        *int32      `bson:"textIndexVersion,omitempty" tfsdk:"text_index_version"`
	SphereIndexVersion      *int32      `bson:"2dsphereIndexVersion,omitempty" tfsdk:"sphere_index_version"`
	Bits                    *int32      `bson:"bits,omitempty" tfsdk:"bits"`
	Min                     *float64    `bson:"min,omitempty" tfsdk:"min"`
	Max                     *float64    `bson:"max,omitempty" tfsdk:"max"`
	WildcardProjection      interface{} `bson:"wildcardProjection,omitempty" tfsdk:"wildcard_projection"`
}

type Index struct {
	Name       string       `bson:"name" tfsdk:"name"`
	Database   string       `bson:"database" tfsdk:"database"`
	Collection string       `bson:"collection" tfsdk:"collection"`
	Keys       IndexKeys    `bson:"keys" tfsdk:"keys"`
	Options    IndexOptions `bson:"options" tfsdk:"options"`
}

func (k *IndexKeys) ToTerraformSet(ctx context.Context) (*types.Set, diag.Diagnostics) {
	var keys []basetypes.ObjectValue
	keyType := types.ObjectType{
		AttrTypes: IndexKeyAttributeTypes,
	}

	for _, key := range *k {
		keyObj, d := types.ObjectValueFrom(ctx, IndexKeyAttributeTypes, key)
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

func (k *IndexKeys) toBson() bson.D {
	out := bson.D{}
	for _, key := range *k {
		var value interface{}
		switch key.Type {
		case "1", "asc":
			value = 1
		case "-1", "desc":
			value = -1
		case "2dsphere":
			value = "2dsphere"
		case "text":
			value = "text"
		default:
			value = 1
		}
		out = append(out, bson.E{Key: key.Field, Value: value})
	}
	return out
}

var IndexKeyAttributeTypes = map[string]attr.Type{
	"field": types.StringType,
	"type":  types.StringType,
}
