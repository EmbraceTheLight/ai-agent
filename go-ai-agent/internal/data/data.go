package data

import (
	"context"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

type Data struct {
	client *milvusclient.Client
}

func NewData(milvusClient *milvusclient.Client) (*Data, func(), error) {
	data := &Data{
		client: milvusClient,
	}
	return data,
		func() {
			data.client.Close(context.Background())
		}, nil
}
