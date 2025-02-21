package mongodb

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	// Set text index options
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

	var indexes []*Index
	if err = cursor.All(ctx, &indexes); err != nil {
		return nil, err
	}

	tflog.Debug(ctx, "Index data from MongoDB", map[string]interface{}{
		"indexes": indexes,
	})

	for _, idx := range indexes {
		if idx.Name == options.Name {
			tflog.Debug(ctx, "Found matching index", map[string]interface{}{
				"index": idx,
			})
			idx.Database = options.Database
			idx.Collection = options.Collection

			return idx, nil
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
