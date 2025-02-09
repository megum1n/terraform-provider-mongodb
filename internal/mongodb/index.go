package mongodb

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	createIndexCmd = "createIndexes"
	getIndexCmd    = "listIndexes"
	dropIndexCmd   = "dropIndexes"
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
			Name:               &index.Name,
			Unique:             index.Unique,
			ExpireAfterSeconds: index.ExpireAfterSeconds,
		},
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
			default:
				keyType = "1"
			}
			index.Keys = append(index.Keys, IndexKey{
				Field: elem.Key,
				Type:  keyType,
			})
		}
	}


	if unique, ok := bsonIndex["unique"].(bool); ok {
		index.Unique = &unique
	}

	if expireAfterSeconds, ok := bsonIndex["expireAfterSeconds"].(int32); ok {
		index.ExpireAfterSeconds = &expireAfterSeconds
	}

	return index, nil
}
