package mongodb

import (
	"context"

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

// Always use version 2 for better compatibility
const DefaultIndexVersion int32 = 2

type IndexKeys []IndexKey

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
}

type Index struct {
	Name       string       `bson:"name"`
	Database   string       `bson:"database"`
	Collection string       `bson:"collection"`
	Keys       IndexKeys    `bson:"keys"`
	Options    IndexOptions `bson:"options"`
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
		case "wildcard":
			value = 1
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
