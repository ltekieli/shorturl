package db

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

type Database interface {
	FetchByShort(link string) ([]string, error)
	FetchByLong(link string) ([]string, error)
	Insert(long string, short string) error

	Disconnect() error
}

type MongoDatabase struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

type LinkPair struct {
	ID    primitive.ObjectID `bson:"_id,omitempty"`
	Long  string             `bson:"long,omitempty"`
	Short string             `bson:"short,omitempty"`
}

func Connect(uri string, database string, collection string) (*MongoDatabase, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	mongoDatabase := client.Database(database)
	mongoCollection := mongoDatabase.Collection(collection)

	return &MongoDatabase{client: client, database: mongoDatabase, collection: mongoCollection}, nil
}

func (mdb *MongoDatabase) FetchByShort(link string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := mdb.collection.Find(ctx, bson.M{"short": bson.D{{"$eq", link}}})
	if err != nil {
		return make([]string, 0), err
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var linkPairs []LinkPair
	if err = cursor.All(ctx, &linkPairs); err != nil {
		return make([]string, 0), err
	}

	links := make([]string, len(linkPairs))
	for i := range links {
		links[i] = linkPairs[i].Long
	}

	return links, nil
}

func (mdb *MongoDatabase) FetchByLong(link string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := mdb.collection.Find(ctx, bson.M{"long": bson.D{{"$eq", link}}})
	if err != nil {
		return make([]string, 0), err
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var linkPairs []LinkPair
	if err = cursor.All(ctx, &linkPairs); err != nil {
		return make([]string, 0), err
	}

	links := make([]string, len(linkPairs))
	for i := range links {
		links[i] = linkPairs[i].Short
	}

	return links, nil
}

func (mdb *MongoDatabase) Insert(long string, short string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := mdb.collection.InsertOne(ctx, LinkPair{Long: long, Short: short})
	return err
}

func (mdb *MongoDatabase) Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return mdb.client.Disconnect(ctx)
}
