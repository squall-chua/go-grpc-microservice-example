package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/squall-chua/go-grpc-microservice-example/api/v1"
	"github.com/squall-chua/go-grpc-microservice-example/internal/repository"
)

type ItemService struct {
	pb.UnimplementedItemServiceServer
	repo repository.ItemRepository
}

func NewItemService(repo repository.ItemRepository) *ItemService {
	return &ItemService{
		repo: repo,
	}
}

func (s *ItemService) CreateItem(ctx context.Context, req *pb.CreateItemRequest) (*pb.Item, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	model := repository.ModelFromProto(req)
	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create item: %v", err)
	}

	itemsCreatedTotal.Inc()

	return created.ToProto(), nil
}

func (s *ItemService) GetItem(ctx context.Context, req *pb.GetItemRequest) (*pb.Item, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	item, err := s.repo.Get(ctx, req.Id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "item not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get item: %v", err)
	}
	return item.ToProto(), nil
}

func (s *ItemService) UpdateItem(ctx context.Context, req *pb.UpdateItemRequest) (*pb.Item, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	item, err := s.repo.Update(ctx, req.Id, req)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "item not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update item: %v", err)
	}

	itemsUpdatedTotal.Inc()

	return item.ToProto(), nil
}

func (s *ItemService) DeleteItem(ctx context.Context, req *pb.DeleteItemRequest) (*pb.DeleteItemResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	err := s.repo.Delete(ctx, req.Id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "item not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete item: %v", err)
	}

	itemsDeletedTotal.Inc()

	return &pb.DeleteItemResponse{Success: true}, nil
}

func (s *ItemService) ListItems(ctx context.Context, req *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	items, count, err := s.repo.List(ctx, req.PageRequest, req.NameContains, req.MinPrice, req.MaxPrice)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list items: %v", err)
	}

	var pbItems []*pb.Item
	for _, item := range items {
		pbItems = append(pbItems, item.ToProto())
	}

	pageSize := int32(10)
	pageNumber := int32(1)
	if req.PageRequest != nil {
		if req.PageRequest.PageSize > 0 {
			pageSize = req.PageRequest.PageSize
		}
		if req.PageRequest.PageNumber > 0 {
			pageNumber = req.PageRequest.PageNumber
		}
	}

	totalPages := count / int64(pageSize)
	if count%int64(pageSize) > 0 {
		totalPages++
	}

	pageInfo := &pb.PageInfo{
		PageNumber: pageNumber,
		PageSize:   pageSize,
		TotalPages: int32(totalPages),
		TotalItems: int32(count),
	}

	return &pb.ListItemsResponse{
		Items:    pbItems,
		PageInfo: pageInfo,
	}, nil
}
