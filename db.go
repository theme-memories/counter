package main

import (
	"context"
	"log"
	"maps"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CounterDB struct {
	client     *mongo.Client
	collection *mongo.Collection
	cache      map[string]int64
	mu         sync.Mutex
}

func NewCounterDB(uri string) *CounterDB {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))

	if err != nil {
		log.Fatal("MongoDB Connection Error:", err)
	}

	err = client.Ping(ctx, nil)

	if err != nil {
		log.Fatal("MongoDB Ping Error:", err)
	}

	col := client.Database("counter").Collection("values")

	return &CounterDB{
		client:     client,
		collection: col,
		cache:      make(map[string]int64),
	}
}

func (db *CounterDB) GetAndIncrement(name string) int64 {
	db.mu.Lock()
	defer db.mu.Unlock()

	if val, ok := db.cache[name]; ok {
		db.cache[name] = val + 1
		return db.cache[name]
	}

	var result struct {
		Num int64 `bson:"num"`
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.collection.FindOne(ctx, bson.M{"name": name}).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			result.Num = 0
		} else {
			log.Printf("DB Error fetching %s: %v", name, err)
		}
	}

	newVal := result.Num + 1
	db.cache[name] = newVal
	return newVal
}

func (db *CounterDB) Sync() {
	db.mu.Lock()

	if len(db.cache) == 0 {
		db.mu.Unlock()
		return
	}

	toSync := make(map[string]int64)
	maps.Copy(toSync, db.cache)
	db.cache = make(map[string]int64)
	db.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for name, num := range toSync {
		_, err := db.collection.UpdateOne(
			ctx,
			bson.M{"name": name},
			bson.M{"$set": bson.M{"num": num}},
			options.Update().SetUpsert(true),
		)

		if err != nil {
			log.Printf("DB Error syncing %s: %v", name, err)
		}
	}

	log.Printf("Synced %d counters to MongoDB", len(toSync))
}

func (db *CounterDB) StartSyncTicker(interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for range ticker.C {
			db.Sync()
		}
	}()
}
