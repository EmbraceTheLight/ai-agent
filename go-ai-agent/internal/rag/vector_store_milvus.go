package rag

import (
	"context"
)

// MilvusOperation milvus 操作接口
type MilvusOperation interface {
	InitCollections(ctx context.Context) error
}

type MilvusUsecase struct {
	repo   MilvusOperation
	loader DocumentLoader
}

func NewMilvusUsecase(repo MilvusOperation) *MilvusUsecase {
	return &MilvusUsecase{
		repo: repo,
	}
}

func (usecase *MilvusUsecase) InitMilvusCollections(ctx context.Context) error {
	return usecase.repo.InitCollections(ctx)
}
