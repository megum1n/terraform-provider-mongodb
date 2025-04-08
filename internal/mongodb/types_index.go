package mongodb

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type IndexKeys map[string]interface{}

type IndexOptions struct {
	Unique                  *bool                  `bson:"unique,omitempty"`
	Sparse                  *bool                  `bson:"sparse,omitempty"`
	Hidden                  *bool                  `bson:"hidden,omitempty"`
	PartialFilterExpression map[string]interface{} `bson:"partialFilterExpression,omitempty"`
	WildcardProjection      map[string]int32       `bson:"wildcardProjection,omitempty"`
	Collation               *options.Collation     `bson:"collation,omitempty"`
	ExpireAfterSeconds      *int32                 `bson:"expireAfterSeconds,omitempty"`
	SphereVersion           *int32                 `bson:"2dSphereVersion,omitempty"`
	Bits                    *int32                 `bson:"bits,omitempty"`
	Min                     *float64               `bson:"min,omitempty"`
	Max                     *float64               `bson:"max,omitempty"`
	Weights                 map[string]int32       `bson:"weights,omitempty"`
	DefaultLanguage         *string                `bson:"default_language,omitempty"`
	LanguageOverride        *string                `bson:"language_override,omitempty"`
	TextIndexVersion        *int32                 `bson:"textIndexVersion,omitempty"`
	Background              *bool                  `bson:"background,omitempty"` 
}

type Index struct {
	Name       string       `bson:"name"`
	Database   string       `bson:"-"` // Not in MongoDB response
	Collection string       `bson:"-"` // Not in MongoDB response
	Keys       IndexKeys    `bson:"key"`
	Options    IndexOptions `bson:"inline"` // Inline embedding
}

func (k IndexKeys) toBson() bson.D {
	out := bson.D{}

	for field, value := range k {
		out = append(out, bson.E{Key: field, Value: value})
	}

	return out
}

func (k IndexKeys) ToStringMap() map[string]string {
	out := map[string]string{}

	for field, value := range k {
		var ok bool

		out[field], ok = value.(string)
		if !ok {
			out[field] = fmt.Sprintf("%v", value)
		}
	}

	return out
}

func ConvertMap(k map[string]string, indexKeys bool) map[string]interface{} {
	out := map[string]interface{}{}

	for field, value := range k {
		if indexKeys {
			switch value {
			case "1":
				out[field] = 1
			case "-1":
				out[field] = -1
			default:
				out[field] = value
			}
		} else {
			out[field] = value
		}
	}

	return out
}
