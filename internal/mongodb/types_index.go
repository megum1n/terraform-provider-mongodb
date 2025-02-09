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
	Type  string `bson:"type"  tfsdk:"type"` // 1 for ascending, -1 for descending
}

type IndexKeys []IndexKey

type Index struct {
	Name               string    `bson:"name"`
	Database           string    `bson:"database"`
	Collection         string    `bson:"collection"`
	Keys               IndexKeys `bson:"keys"`
	Unique             *bool     `bson:"unique,omitempty"`
	ExpireAfterSeconds *int32    `bson:"expireAfterSeconds,omitempty"`
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
		case "asc", "1":
			value = 1
		case "desc", "-1":
			value = -1
		default:
			value = 1 // default to ascending
		}
		out = append(out, bson.E{Key: key.Field, Value: value})
	}
	return out
}

var IndexKeyAttributeTypes = map[string]attr.Type{
	"field": types.StringType,
	"type":  types.StringType,
}
