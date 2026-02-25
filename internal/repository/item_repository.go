package repository

import (
	"context"
	"errors"

	"github.com/squall-chua/gmqb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	pb "github.com/squall-chua/go-grpc-microservice-example/api/v1"
)

var ErrNotFound = errors.New("item not found")

type ItemModel struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	Name        string        `bson:"name"`
	Description string        `bson:"description"`
	Price       float64       `bson:"price"`
}

func (m *ItemModel) ToProto() *pb.Item {
	return &pb.Item{
		Id:          m.ID.Hex(),
		Name:        m.Name,
		Description: m.Description,
		Price:       m.Price,
	}
}

func ModelFromProto(req *pb.CreateItemRequest) *ItemModel {
	return &ItemModel{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
	}
}

type ItemRepository interface {
	Create(ctx context.Context, item *ItemModel) (*ItemModel, error)
	Get(ctx context.Context, id string) (*ItemModel, error)
	Update(ctx context.Context, id string, item *pb.UpdateItemRequest) (*ItemModel, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, pageReq *pb.PageRequest, nameContains string, minPrice, maxPrice float64) ([]*ItemModel, int64, error)
}

type itemRepository struct {
	coll *gmqb.Collection[ItemModel]
}

func NewItemRepository(db *mongo.Database) ItemRepository {
	return &itemRepository{
		coll: gmqb.Wrap[ItemModel](db.Collection("items")),
	}
}

func (r *itemRepository) Create(ctx context.Context, item *ItemModel) (*ItemModel, error) {
	result, err := r.coll.InsertOne(ctx, item)
	if err != nil {
		return nil, err
	}
	item.ID = result.InsertedID.(bson.ObjectID)
	return item, nil
}

func (r *itemRepository) Get(ctx context.Context, id string) (*ItemModel, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	f := gmqb.Field[ItemModel]
	filter := gmqb.Eq(f("ID"), objID)
	result, err := r.coll.FindOne(ctx, filter)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return result, nil
}

func (r *itemRepository) Update(ctx context.Context, id string, req *pb.UpdateItemRequest) (*ItemModel, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	f := gmqb.Field[ItemModel]
	filter := gmqb.Eq(f("ID"), objID)
	update := gmqb.NewUpdate().
		Set(f("Name"), req.Name).
		Set(f("Description"), req.Description).
		Set(f("Price"), req.Price)

	result, err := r.coll.FindOneAndUpdate(ctx, filter, update, gmqb.WithReturnDocument(1)) // 1 is ReturnDocument.After
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return result, nil
}

func (r *itemRepository) Delete(ctx context.Context, id string) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	f := gmqb.Field[ItemModel]
	filter := gmqb.Eq(f("ID"), objID)
	result, err := r.coll.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *itemRepository) List(ctx context.Context, pageReq *pb.PageRequest, nameContains string, minPrice, maxPrice float64) ([]*ItemModel, int64, error) {
	// Build match filter
	f := gmqb.Field[ItemModel]
	filters := []gmqb.Filter{}
	if nameContains != "" {
		filters = append(filters, gmqb.Regex(f("Name"), nameContains, "i"))
	}
	if minPrice > 0 {
		filters = append(filters, gmqb.Gte(f("Price"), minPrice))
	}
	if maxPrice > 0 {
		filters = append(filters, gmqb.Lte(f("Price"), maxPrice))
	}

	var matchFilter gmqb.Filter
	if len(filters) > 0 {
		matchFilter = gmqb.And(filters...)
	} else {
		matchFilter = gmqb.Raw(bson.D{})
	}

	var effectiveLimit int64 = 10
	var offset int64 = 0

	itemsPipeline := gmqb.NewPipeline()

	if pageReq != nil {
		if pageReq.PageSize > 0 {
			effectiveLimit = int64(pageReq.PageSize)
		}
		if pageReq.PageNumber > 0 {
			offset = int64(pageReq.PageNumber-1) * effectiveLimit
		}
	}

	itemsPipeline = itemsPipeline.Skip(offset).Limit(effectiveLimit)

	// Create a facet pipeline to fetch paginated items and the total count in one go
	pipeline := gmqb.NewPipeline().
		Match(matchFilter).
		Facet(map[string]gmqb.Pipeline{
			"items": itemsPipeline,
			"total": gmqb.NewPipeline().
				Count("count"),
		})

	type countResult struct {
		Count int64 `bson:"count"`
	}

	type facetResult struct {
		Items []ItemModel   `bson:"items"`
		Total []countResult `bson:"total"`
	}

	results, err := gmqb.Aggregate[facetResult](r.coll, ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}

	if len(results) == 0 {
		return []*ItemModel{}, 0, nil
	}

	res := results[0]
	var count int64 = 0
	if len(res.Total) > 0 {
		count = res.Total[0].Count
	}

	var items []*ItemModel
	for _, it := range res.Items {
		val := it
		items = append(items, &val)
	}

	return items, count, nil
}
