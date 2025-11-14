package mongodb

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type GetIndexOptions struct {
	Name       string
	Database   string
	Collection string
}

// setIndexOptions is a workaround to use pointers. As an alternative, we can check each option for nil and then set it.
func setIndexOptions(index *Index) func(*options.IndexOptions) error {
	return func(opts *options.IndexOptions) error {
		opts.Unique = index.Options.Unique
		opts.Sparse = index.Options.Sparse
		opts.Hidden = index.Options.Hidden
		opts.Collation = index.Options.Collation
		opts.ExpireAfterSeconds = index.Options.ExpireAfterSeconds
		opts.SphereVersion = index.Options.SphereVersion
		opts.Bits = index.Options.Bits
		opts.Min = index.Options.Min
		opts.Max = index.Options.Max
		opts.DefaultLanguage = index.Options.DefaultLanguage
		opts.LanguageOverride = index.Options.LanguageOverride
		opts.TextVersion = index.Options.TextIndexVersion

		if len(index.Options.PartialFilterExpression) > 0 {
			opts.PartialFilterExpression = index.Options.PartialFilterExpression
		}

		if len(index.Options.WildcardProjection) > 0 {
			opts.WildcardProjection = index.Options.WildcardProjection
		}

		if len(index.Options.Weights) > 0 {
			opts.Weights = index.Options.Weights
		}

		return nil
	}
}

func (c *Client) CreateIndex(ctx context.Context, index *Index) (*Index, error) {
	tflog.Debug(ctx, "CreateIndex", map[string]any{
		"database":   index.Database,
		"collection": index.Collection,
		"name":       index.Name,
	})

	opts := options.Index().
		SetName(index.Name)

	opts.Opts = append(opts.Opts, setIndexOptions(index))

	indexModel := mongo.IndexModel{
		Keys:    index.Keys.toBson(),
		Options: opts,
	}

	collection := c.mongo.Database(index.Database).Collection(index.Collection)

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return nil, fmt.Errorf("error creating index: %w", err)
	}

	return c.GetIndex(ctx, &GetIndexOptions{
		Name:       index.Name,
		Database:   index.Database,
		Collection: index.Collection,
	})
}

func (c *Client) GetIndex(ctx context.Context, opt *GetIndexOptions) (*Index, error) {
	collection := c.mongo.Database(opt.Database).Collection(opt.Collection)

	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}

	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			tflog.Error(ctx, "error closing cursor", map[string]any{
				"err": err,
			})
		}
	}(cursor, ctx)

	var indexes []Index

	err = cursor.All(ctx, &indexes)
	if err != nil {
		return nil, err
	}

	tflog.Debug(ctx, "Index data from MongoDB", map[string]any{
		"indexes": indexes,
	})

	for i := range indexes {
		if indexes[i].Name == opt.Name {
			indexes[i].Database = opt.Database
			indexes[i].Collection = opt.Collection

			return &indexes[i], nil
		}
	}

	return nil, NotFoundError{
		name: opt.Name,
		t:    "index",
	}
}

func (c *Client) DeleteIndex(ctx context.Context, options *GetIndexOptions) error {
	tflog.Debug(ctx, "DeleteIndex", map[string]any{
		"database":   options.Database,
		"collection": options.Collection,
		"name":       options.Name,
	})

	collection := c.mongo.Database(options.Database).Collection(options.Collection)

	return collection.Indexes().DropOne(ctx, options.Name)
}
