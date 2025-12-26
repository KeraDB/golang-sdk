package main

import (
	"fmt"
	"log"

	"github.com/KeraDB/golang-sdk"
)

func main() {
	// Connect to database (MongoDB-compatible API)
	fmt.Println("Connecting to database...")
	client, err := keradb.Connect("example.ndb")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Get database and collection
	db := client.Database()
	users := db.Collection("users")

	// Insert documents
	fmt.Println("\n--- Inserting documents ---")
	result1, err := users.InsertOne(keradb.M{
		"name":  "Alice",
		"age":   30,
		"email": "alice@example.com",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Inserted Alice with ID: %s\n", result1.InsertedID)

	result2, err := users.InsertOne(keradb.M{
		"name":  "Bob",
		"age":   25,
		"email": "bob@example.com",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Inserted Bob with ID: %s\n", result2.InsertedID)

	// Find by ID
	fmt.Println("\n--- Finding by ID ---")
	var alice keradb.Document
	err = users.FindOne(keradb.M{"_id": result1.InsertedID}).Decode(&alice)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found Alice: %v\n", alice)

	// Update
	fmt.Println("\n--- Updating ---")
	updateResult, err := users.UpdateOne(
		keradb.M{"_id": result1.InsertedID},
		keradb.M{"$set": keradb.M{"age": 31}},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated %d document(s)\n", updateResult.ModifiedCount)

	// Find all
	fmt.Println("\n--- Finding all ---")
	cursor := users.Find(keradb.M{})
	allUsers, err := cursor.All()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("All users: %v\n", allUsers)

	// Count
	fmt.Println("\n--- Counting ---")
	count, err := users.CountDocuments(keradb.M{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total users: %d\n", count)

	// List collections
	fmt.Println("\n--- Listing collections ---")
	collections, err := db.ListCollectionNames()
	if err != nil {
		log.Fatal(err)
	}
	for _, name := range collections {
		fmt.Printf("Collection: %s\n", name)
	}

	// Pagination
	fmt.Println("\n--- Pagination ---")
	page1, err := users.Find(keradb.M{}).Limit(1).Skip(0).All()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Page 1: %v\n", page1)

	page2, err := users.Find(keradb.M{}).Limit(1).Skip(1).All()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Page 2: %v\n", page2)

	// Delete
	fmt.Println("\n--- Deleting ---")
	deleteResult, err := users.DeleteOne(keradb.M{"_id": result2.InsertedID})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Deleted %d document(s)\n", deleteResult.DeletedCount)

	remaining, _ := users.CountDocuments(keradb.M{})
	fmt.Printf("Remaining users: %d\n", remaining)

	// Sync
	if err := client.Sync(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nDatabase operations completed successfully!")
}
