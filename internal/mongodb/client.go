package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"

	"go.mongodb.org/mongo-driver/v2/mongo"
	mongooptions "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ClientOptions struct {
	Hosts              []string
	Username           string
	Password           string
	AuthSource         string
	AuthMechanism      string
	ReplicaSet         string
	TLS                bool
	InsecureSkipVerify bool
	Certificate        string
	DirectConnection   bool
}

type Client struct {
	mongo *mongo.Client

	ClientOptions
}

func New(ctx context.Context, options *ClientOptions) (*Client, error) {
	opt := mongooptions.Client().
		SetHosts(options.Hosts).
		SetAuth(mongooptions.Credential{
			Username:      options.Username,
			Password:      options.Password,
			AuthSource:    options.AuthSource,
			AuthMechanism: options.AuthMechanism,
		}).
		SetReplicaSet(options.ReplicaSet).
		SetDirect(options.DirectConnection)

	if options.TLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: options.InsecureSkipVerify,
		}

		if options.Certificate != "" {
			certPool := x509.NewCertPool()

			ok := certPool.AppendCertsFromPEM([]byte(options.Certificate))
			if !ok {
				return nil, errors.New("failed to parse certificate")
			}

			tlsConfig.RootCAs = certPool
		}

		opt.SetTLSConfig(tlsConfig)
	}

	mongoClient, err := mongo.Connect(opt)
	if err != nil {
		return nil, err
	}

	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	client := &Client{
		mongo:         mongoClient,
		ClientOptions: *options,
	}

	return client, nil
}
