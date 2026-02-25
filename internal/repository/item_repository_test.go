package repository

import (
	"context"
	"testing"

	"github.com/tryvium-travels/memongo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	pb "github.com/squall-chua/go-grpc-microservice-example/api/v1"
)

var mongoServer *memongo.Server
var testDb *mongo.Database

func TestMain(m *testing.M) {
	var err error
	mongoServer, err = memongo.Start("8.2.5")
	if err != nil {
		panic(err)
	}
	defer mongoServer.Stop()

	clientOpts := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		panic(err)
	}

	testDb = client.Database("testdb")

	m.Run()
}

func TestItemRepository_CreateAndGet(t *testing.T) {
	repo := NewItemRepository(testDb)

	// Test Create
	item := &ItemModel{
		Name:  "Test Item",
		Price: 10.5,
	}
	created, err := repo.Create(context.Background(), item)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}
	if created.ID.IsZero() {
		t.Fatal("expected ID to be populated")
	}

	// Test Get
	fetched, err := repo.Get(context.Background(), created.ID.Hex())
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if fetched.Name != item.Name {
		t.Errorf("expected %v, got %v", item.Name, fetched.Name)
	}

	// Test List
	items, count, err := repo.List(context.Background(), &pb.PageRequest{PageSize: 10, PageNumber: 1}, "", 0, 0)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	// Depending on previous test interactions, count could be >= 1. We just want to ensure it works.
	if count < 1 {
		t.Errorf("expected count >= 1, got %v", count)
	}
	if len(items) < 1 {
		t.Fatalf("expected >= 1 item, got %v", len(items))
	}

	// Test Update
	req := &pb.UpdateItemRequest{
		Id:    created.ID.Hex(),
		Name:  "Updated Item",
		Price: 20.0,
	}
	updated, err := repo.Update(context.Background(), created.ID.Hex(), req)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}
	if updated.Name != req.Name {
		t.Errorf("expected %v, got %v", req.Name, updated.Name)
	}

	// Test Delete
	err = repo.Delete(context.Background(), created.ID.Hex())
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Verify Delete
	_, err = repo.Get(context.Background(), created.ID.Hex())
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestItemModel_ToProto(t *testing.T) {
	model := &ItemModel{
		ID:          bson.NewObjectID(),
		Name:        "Test",
		Description: "Desc",
		Price:       1.0,
	}
	proto := model.ToProto()
	if proto.Id != model.ID.Hex() || proto.Name != "Test" {
		t.Fatal("ToProto failed")
	}
}

func TestModelFromProto(t *testing.T) {
	req := &pb.CreateItemRequest{
		Name:        "Test",
		Description: "Desc",
		Price:       1.0,
	}
	model := ModelFromProto(req)
	if model.Name != "Test" {
		t.Fatal("ModelFromProto failed")
	}
}

func TestItemRepository_Errors(t *testing.T) {
	repo := NewItemRepository(testDb)
	ctx := context.Background()

	// invalid ID Get
	_, err := repo.Get(ctx, "invalid-hex")
	if err == nil {
		t.Fatal("expected err for invalid hex on Get")
	}

	// invalid ID Update
	_, err = repo.Update(ctx, "invalid-hex", &pb.UpdateItemRequest{})
	if err == nil {
		t.Fatal("expected err for invalid hex on Update")
	}

	// invalid ID Delete
	err = repo.Delete(ctx, "invalid-hex")
	if err == nil {
		t.Fatal("expected err for invalid hex on Delete")
	}
}
