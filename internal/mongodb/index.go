package mongodb

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	mongooptions "go.mongodb.org/mongo-driver/mongo/options"
)

type GetIndexOptions struct {
	Name       string
	Database   string
	Collection string
}

func (c *Client) CreateIndex(ctx context.Context, index *Index) (*Index, error) {

	tflog.Debug(ctx, "CreateIndex", map[string]interface{}{
		"database":   index.Database,
		"collection": index.Collection,
		"name":       index.Name,
	})

	// Determine if it's a wildcard index
	isWildcardIndex := false
	for _, key := range index.Keys {
		if key.Type == "wildcard" {
			isWildcardIndex = true
			break
		}
	}

	// Check if it's a 2d index
	is2dIndex := false
	for _, key := range index.Keys {
		if key.Type == "2d" {
			is2dIndex = true
			break
		}
	}

	// Check if it's a text index
	isTextIndex := false
	for _, key := range index.Keys {
		if key.Type == "text" {
			isTextIndex = true
			break
		}
	}

	version := DefaultIndexVersion

	opts := options.Index().
		SetName(index.Name).
		SetVersion(version)

	// Only set options if they are explicitly specified
	if index.Options.Unique {
		opts.SetUnique(index.Options.Unique)
	}

	if index.Options.Sparse {
		opts.SetSparse(index.Options.Sparse)
	}

	if index.Options.Hidden {
		opts.SetHidden(index.Options.Hidden)
	}

	// Set 2d-specific options
	if is2dIndex {
		if index.Options.Bits > 0 {
			opts.SetBits(index.Options.Bits)
		}
		if index.Options.Min != 0 {
			opts.SetMin(index.Options.Min)
		}
		if index.Options.Max != 0 {
			opts.SetMax(index.Options.Max)
		}
	}

	// In CreateIndex function, after other options:
	if isTextIndex {
		if index.Options.Weights != nil {
			opts.SetWeights(index.Options.Weights)
		}
		if index.Options.DefaultLanguage != "" {
			opts.SetDefaultLanguage(index.Options.DefaultLanguage)
		}
		if index.Options.LanguageOverride != "" {
			opts.SetLanguageOverride(index.Options.LanguageOverride)
		}
		if index.Options.TextIndexVersion > 0 {
			opts.SetTextVersion(index.Options.TextIndexVersion)
		}
	}

	// Only set TTL for non-wildcard indexes
	if index.Options.ExpireAfterSeconds > 0 && !isWildcardIndex {
		opts.SetExpireAfterSeconds(index.Options.ExpireAfterSeconds)
	}

	if index.Options.Collation != nil {
		opts.SetCollation(index.Options.Collation)
	}

	if len(index.Options.PartialFilterExpression) > 0 {
		opts.PartialFilterExpression = index.Options.PartialFilterExpression
	}

	// Only set for wildcard indexes and if not empty
	if isWildcardIndex && len(index.Options.WildcardProjection) > 0 {
		opts.WildcardProjection = index.Options.WildcardProjection
	}

	indexModel := mongo.IndexModel{
		Keys:    index.Keys.toBson(),
		Options: opts,
	}

	collection := c.mongo.Database(index.Database).Collection(index.Collection)
	indexName, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return nil, fmt.Errorf("error creating index: %v", err)
	}

	index.Name = indexName
	index.Options.Version = version
	return c.GetIndex(ctx, &GetIndexOptions{
		Name:       index.Name,
		Database:   index.Database,
		Collection: index.Collection,
	})
}

func (c *Client) GetIndex(ctx context.Context, options *GetIndexOptions) (*Index, error) {

	collection := c.mongo.Database(options.Database).Collection(options.Collection)
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rawIndexes []bson.M
	if err = cursor.All(ctx, &rawIndexes); err != nil {
		return nil, err
	}

	tflog.Debug(ctx, "Index data from MongoDB", map[string]interface{}{
		"indexes": rawIndexes,
	})

	for _, rawIndex := range rawIndexes {
		if rawIndex["name"] == options.Name {
			tflog.Debug(ctx, "Found matching index", map[string]interface{}{
				"index": rawIndex,
			})

			var index Index
			index.Name = options.Name
			index.Database = options.Database
			index.Collection = options.Collection

			// Decode keys
			if keyValue, ok := rawIndex["key"].(bson.M); ok {
				index.Keys = make(IndexKeys, 0)

				// Check if this is a text index
				isTextIndex := false
				if value, hasFts := keyValue["_fts"]; hasFts {
					if valueStr, ok := value.(string); ok && valueStr == "text" {
						isTextIndex = true
					}
				}

				if isTextIndex {
					// Handle text index keys
					//if weights, hasWeights := rawIndex["weights"].(bson.M); hasWeights {
					if weights, hasWeights := rawIndex["weights"].(bson.M); hasWeights && len(weights) > 0 {
						for field := range weights {
							index.Keys = append(index.Keys, IndexKey{
								Field: field,
								Type:  "text",
							})
						}
					} else {
						// Fallback if weights not found
						index.Keys = append(index.Keys, IndexKey{
							Field: "_fts",
							Type:  "text",
						})
					}
				} else {
					// Handle standard indexes
					for field, value := range keyValue {
						fieldType := fmt.Sprintf("%v", value)

						// Special case for wildcard indexes
						if field == "$**" && value == int32(1) {
							fieldType = "wildcard"
						} else {
							// Standard index type mapping
							switch value {
							case int32(1):
								fieldType = "1"
							case int32(-1):
								fieldType = "-1"
							}
						}

						index.Keys = append(index.Keys, IndexKey{
							Field: field,
							Type:  fieldType,
						})
					}
				}
			}

			// Process text index specific options
			if textVersion, exists := rawIndex["textIndexVersion"].(int32); exists {
				index.Options.TextIndexVersion = textVersion
			}

			if defaultLang, exists := rawIndex["default_language"].(string); exists {
				index.Options.DefaultLanguage = defaultLang
			}

			if langOverride, exists := rawIndex["language_override"].(string); exists {
				index.Options.LanguageOverride = langOverride
			}

			// Process weights for text indexes
			if weights, exists := rawIndex["weights"].(bson.M); exists && len(weights) > 0 {
				index.Options.Weights = make(map[string]int32)
				for field, value := range weights {
					if weight, ok := value.(int32); ok {
						index.Options.Weights[field] = weight
					}
				}
			}

			// In GetIndex function, add handling for 2d index options:
			if bits, exists := rawIndex["bits"].(int32); exists {
				index.Options.Bits = bits
			}
			if min, exists := rawIndex["min"].(float64); exists {
				index.Options.Min = min
			}
			if max, exists := rawIndex["max"].(float64); exists {
				index.Options.Max = max
			}

			// Process wildcard projection
			if proj, exists := rawIndex["wildcardProjection"].(bson.M); exists && proj != nil {
				index.Options.WildcardProjection = make(map[string]int32)
				for key, value := range proj {
					if v, ok := value.(int32); ok {
						index.Options.WildcardProjection[key] = v
					} else if v, ok := value.(int); ok {
						index.Options.WildcardProjection[key] = int32(v)
					}
				}
			}

			if collation, exists := rawIndex["collation"].(bson.M); exists && collation != nil {
				// Map specific fields we care about
				userCollation := mongooptions.Collation{
					Locale:          collation["locale"].(string),
					Strength:        int(collation["strength"].(int32)),
					CaseLevel:       collation["caseLevel"].(bool),
					Alternate:       collation["alternate"].(string),
					Backwards:       collation["backwards"].(bool),
					CaseFirst:       collation["caseFirst"].(string),
					MaxVariable:     collation["maxVariable"].(string),
					NumericOrdering: collation["numericOrdering"].(bool),
				}
				index.Options.Collation = &userCollation
			}

			// Process partial filter expression
			if pfe, exists := rawIndex["partialFilterExpression"].(bson.M); exists && pfe != nil {
				index.Options.PartialFilterExpression = make(map[string]interface{})
				for key, value := range pfe {
					switch v := value.(type) {
					case int32:
						index.Options.PartialFilterExpression[key] = fmt.Sprintf("%d", v)
					default:
						index.Options.PartialFilterExpression[key] = fmt.Sprintf("%v", v)
					}
				}
			}

			// Process common index options
			if v, ok := rawIndex["unique"].(bool); ok {
				index.Options.Unique = v
			}

			if v, ok := rawIndex["sparse"].(bool); ok {
				index.Options.Sparse = v
			}

			if v, ok := rawIndex["v"].(int32); ok {
				index.Options.Version = v
			}

			return &index, nil
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
