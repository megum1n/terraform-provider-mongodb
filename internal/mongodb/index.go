package mongodb

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (c *Client) CreateIndex(ctx context.Context, index *Index) (*Index, error) {
	tflog.Debug(ctx, "CreateIndex", map[string]interface{}{
		"database":   index.Database,
		"collection": index.Collection,
		"name":       index.Name,
	})

	indexModel := mongo.IndexModel{
		Keys: index.Keys.toBson(),
		Options: &options.IndexOptions{
			Name:                    &index.Name,
			Unique:                  index.Options.Unique,
			Sparse:                  index.Options.Sparse,
			ExpireAfterSeconds:      index.Options.ExpireAfterSeconds,
			Hidden:                  index.Options.Hidden,
			StorageEngine:           index.Options.StorageEngine,
			PartialFilterExpression: index.Options.PartialFilterExpression,
			Weights:                 index.Options.Weights,
			DefaultLanguage:         index.Options.DefaultLanguage,
			LanguageOverride:        index.Options.LanguageOverride,
			Version:                 index.Options.SphereIndexVersion,
			Bits:                    index.Options.Bits,
			Min:                     index.Options.Min,
			Max:                     index.Options.Max,
			WildcardProjection:      index.Options.WildcardProjection,
		},
	}

	if index.Options.Collation != nil {
		if collationMap, ok := index.Options.Collation.(map[string]interface{}); ok {
			collation := options.Collation{}
			if locale, ok := collationMap["locale"].(string); ok {
				collation.Locale = locale
			}
			if caseLevel, ok := collationMap["caseLevel"].(bool); ok {
				collation.CaseLevel = caseLevel
			}
			if caseFirst, ok := collationMap["caseFirst"].(string); ok {
				collation.CaseFirst = caseFirst
			}
			if strength, ok := collationMap["strength"].(float64); ok {
				strengthInt := int(strength)
				collation.Strength = strengthInt
			}
			if numericOrdering, ok := collationMap["numericOrdering"].(bool); ok {
				collation.NumericOrdering = numericOrdering
			}
			if alternate, ok := collationMap["alternate"].(string); ok {
				collation.Alternate = alternate
			}
			if maxVariable, ok := collationMap["maxVariable"].(string); ok {
				collation.MaxVariable = maxVariable
			}
			if backwards, ok := collationMap["backwards"].(bool); ok {
				collation.Backwards = backwards
			}
			indexModel.Options.Collation = &collation
		}
	}

	collection := c.mongo.Database(index.Database).Collection(index.Collection)
	indexName, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return nil, fmt.Errorf("error creating index: %v", err)
	}

	index.Name = indexName
	return c.GetIndex(ctx, &GetIndexOptions{
		Name:       index.Name,
		Database:   index.Database,
		Collection: index.Collection,
	})
}

type GetIndexOptions struct {
	Name       string
	Database   string
	Collection string
}

func (c *Client) GetIndex(ctx context.Context, options *GetIndexOptions) (*Index, error) {
	tflog.Debug(ctx, "GetIndex", map[string]interface{}{
		"database":   options.Database,
		"collection": options.Collection,
		"name":       options.Name,
	})

	collection := c.mongo.Database(options.Database).Collection(options.Collection)
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var indexes []bson.M
	if err = cursor.All(ctx, &indexes); err != nil {
		return nil, err
	}

	for _, idx := range indexes {
		if idx["name"] == options.Name {
			return convertBsonToIndex(idx, options.Database, options.Collection)
		}
	}

	return nil, NotFoundError{
		name: options.Name,
		t:    "index",
	}
}

func (c *Client) DeleteIndex(ctx context.Context, options *GetIndexOptions) error {
	tflog.Debug(ctx, "DeleteIndex", map[string]interface{}{
		"database":   options.Database,
		"collection": options.Collection,
		"name":       options.Name,
	})

	collection := c.mongo.Database(options.Database).Collection(options.Collection)
	_, err := collection.Indexes().DropOne(ctx, options.Name)
	return err
}

func convertBsonToIndex(bsonIndex bson.M, database, collection string) (*Index, error) {
	index := &Index{
		Name:       bsonIndex["name"].(string),
		Database:   database,
		Collection: collection,
	}

	if key, ok := bsonIndex["key"].(bson.D); ok {
		for _, elem := range key {
			var keyType string
			switch elem.Value {
			case int32(1), int64(1), float64(1):
				keyType = "1"
			case int32(-1), int64(-1), float64(-1):
				keyType = "-1"
			case "2dsphere":
				keyType = "2dsphere"
			case "text":
				keyType = "text"
			default:
				keyType = "1"
			}
			index.Keys = append(index.Keys, IndexKey{
				Field: elem.Key,
				Type:  keyType,
			})
		}
	}

	// Convert options
	if unique, ok := bsonIndex["unique"].(bool); ok {
		index.Options.Unique = &unique
	}
	if sparse, ok := bsonIndex["sparse"].(bool); ok {
		index.Options.Sparse = &sparse
	}
	if hidden, ok := bsonIndex["hidden"].(bool); ok {
		index.Options.Hidden = &hidden
	}
	if eas, ok := bsonIndex["expireAfterSeconds"].(int32); ok {
		index.Options.ExpireAfterSeconds = &eas
	}
	if pfe, ok := bsonIndex["partialFilterExpression"]; ok {
		index.Options.PartialFilterExpression = pfe
	}
	if weights, ok := bsonIndex["weights"]; ok {
		index.Options.Weights = weights
	}
	if collation, ok := bsonIndex["collation"]; ok {
		if collationOpts, ok := collation.(*options.Collation); ok {
			index.Options.Collation = collationOpts
		}
	}

	return index, nil
}
