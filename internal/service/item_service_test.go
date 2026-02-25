package service

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/squall-chua/go-grpc-microservice-example/api/v1"
	"github.com/squall-chua/go-grpc-microservice-example/internal/repository"
)

type mockRepo struct {
	repository.ItemRepository
}

func (m *mockRepo) Create(ctx context.Context, item *repository.ItemModel) (*repository.ItemModel, error) {
	mockID, _ := bson.ObjectIDFromHex("60f7b1b3b3b3b3b3b3b3b3b3")
	item.ID = mockID
	return item, nil
}

func (m *mockRepo) Get(ctx context.Context, id string) (*repository.ItemModel, error) {
	if id == "not-found" {
		return nil, repository.ErrNotFound
	}
	if id == "error" {
		return nil, context.DeadlineExceeded
	}
	mockID, _ := bson.ObjectIDFromHex("60f7b1b3b3b3b3b3b3b3b3b3")
	return &repository.ItemModel{ID: mockID, Name: "Test"}, nil
}

func (m *mockRepo) Update(ctx context.Context, id string, req *pb.UpdateItemRequest) (*repository.ItemModel, error) {
	if id == "not-found" {
		return nil, repository.ErrNotFound
	}
	if id == "error" {
		return nil, context.DeadlineExceeded
	}
	mockID, _ := bson.ObjectIDFromHex("60f7b1b3b3b3b3b3b3b3b3b3")
	return &repository.ItemModel{ID: mockID, Name: req.Name}, nil
}

func (m *mockRepo) List(ctx context.Context, pageReq *pb.PageRequest, nameContains string, minPrice, maxPrice float64) ([]*repository.ItemModel, int64, error) {
	if nameContains == "error" {
		return nil, 0, context.DeadlineExceeded
	}
	mockID, _ := bson.ObjectIDFromHex("60f7b1b3b3b3b3b3b3b3b3b3")
	return []*repository.ItemModel{{ID: mockID, Name: "Test"}}, 1, nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	if id == "not-found" {
		return repository.ErrNotFound
	}
	if id == "error" {
		return context.DeadlineExceeded
	}
	return nil
}

func TestItemService_CreateItem(t *testing.T) {
	svc := NewItemService(&mockRepo{})

	req := &pb.CreateItemRequest{
		Name:  "Test",
		Price: 100,
	}

	resp, err := svc.CreateItem(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no err, got %v", err)
	}
	if resp.Id != "60f7b1b3b3b3b3b3b3b3b3b3" {
		t.Errorf("expected 60f7b1b3b3b3b3b3b3b3b3b3, got %v", resp.Id)
	}
}

func TestItemService_DeleteItem(t *testing.T) {
	svc := NewItemService(&mockRepo{})

	req := &pb.DeleteItemRequest{
		Id: "mock-id",
	}

	// 1. Success
	resp, err := svc.DeleteItem(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no err, got %v", err)
	}
	if !resp.Success {
		t.Errorf("expected true, got %v", resp.Success)
	}

	// 2. Missing ID
	_, err = svc.DeleteItem(context.Background(), &pb.DeleteItemRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}

	// 3. Not Found
	_, err = svc.DeleteItem(context.Background(), &pb.DeleteItemRequest{Id: "not-found"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound, got %v", err)
	}

	// 4. Internal Error
	_, err = svc.DeleteItem(context.Background(), &pb.DeleteItemRequest{Id: "error"})
	if status.Code(err) != codes.Internal {
		t.Errorf("expected Internal, got %v", err)
	}
}

func TestItemService_GetItem(t *testing.T) {
	svc := NewItemService(&mockRepo{})

	// 1. Success
	resp, err := svc.GetItem(context.Background(), &pb.GetItemRequest{Id: "mock-id"})
	if err != nil {
		t.Fatalf("expected no err, got %v", err)
	}
	if resp.Id != "60f7b1b3b3b3b3b3b3b3b3b3" {
		t.Errorf("expected 60f7b1b3b3b3b3b3b3b3b3b3, got %v", resp.Id)
	}

	// 2. Missing ID
	_, err = svc.GetItem(context.Background(), &pb.GetItemRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}

	// 3. Not Found
	_, err = svc.GetItem(context.Background(), &pb.GetItemRequest{Id: "not-found"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound, got %v", err)
	}

	// 4. Internal Error
	_, err = svc.GetItem(context.Background(), &pb.GetItemRequest{Id: "error"})
	if status.Code(err) != codes.Internal {
		t.Errorf("expected Internal, got %v", err)
	}
}

func TestItemService_UpdateItem(t *testing.T) {
	svc := NewItemService(&mockRepo{})

	// 1. Success
	resp, err := svc.UpdateItem(context.Background(), &pb.UpdateItemRequest{Id: "mock-id", Name: "Updated"})
	if err != nil {
		t.Fatalf("expected no err, got %v", err)
	}
	if resp.Name != "Updated" {
		t.Errorf("expected Updated, got %v", resp.Name)
	}

	// 2. Missing ID
	_, err = svc.UpdateItem(context.Background(), &pb.UpdateItemRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}

	// 3. Not Found
	_, err = svc.UpdateItem(context.Background(), &pb.UpdateItemRequest{Id: "not-found"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound, got %v", err)
	}

	// 4. Internal Error
	_, err = svc.UpdateItem(context.Background(), &pb.UpdateItemRequest{Id: "error"})
	if status.Code(err) != codes.Internal {
		t.Errorf("expected Internal, got %v", err)
	}
}

func TestItemService_ListItems(t *testing.T) {
	svc := NewItemService(&mockRepo{})

	// 1. Success with pagination logic
	resp, err := svc.ListItems(context.Background(), &pb.ListItemsRequest{
		PageRequest: &pb.PageRequest{PageNumber: 2, PageSize: 5},
	})
	if err != nil {
		t.Fatalf("expected no err, got %v", err)
	}
	if resp.PageInfo.TotalItems != 1 {
		t.Errorf("expected 1 total item, got %v", resp.PageInfo.TotalItems)
	}

	// 2. Success with defaults
	resp, err = svc.ListItems(context.Background(), &pb.ListItemsRequest{})
	if err != nil {
		t.Fatalf("expected no err, got %v", err)
	}
	if resp.PageInfo.PageSize != 10 {
		t.Errorf("expected default page size 10, got %v", resp.PageInfo.PageSize)
	}

	// 3. Internal Error
	_, err = svc.ListItems(context.Background(), &pb.ListItemsRequest{NameContains: "error"})
	if status.Code(err) != codes.Internal {
		t.Errorf("expected Internal, got %v", err)
	}
}
