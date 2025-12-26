package main

import (
	"fmt"
	"log"

	"github.com/KeraDB/golang-sdk"
)

func main() {
	// Create a new database
	fmt.Println("Creating database...")
	db, err := nosqlite.Create("example.ndb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Insert documents
	fmt.Println("\n--- Inserting documents ---")
	id1, err := db.Insert("users", map[string]interface{}{
		"name":  "Alice",
		"age":   30,
		"email": "alice@example.com",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Inserted Alice with ID: %s\n", id1)

	id2, err := db.Insert("users", map[string]interface{}{
		"name":  "Bob",
		"age":   25,
		"email": "bob@example.com",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Inserted Bob with ID: %s\n", id2)

	// Find by ID
	fmt.Println("\n--- Finding by ID ---")
	alice, err := db.FindByID("users", id1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found Alice: %v\n", alice)

	// Update
	fmt.Println("\n--- Updating ---")
	updated, err := db.Update("users", id1, map[string]interface{}{
		"name":  "Alice",
		"age":   31,
		"email": "alice@example.com",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated Alice: %v\n", updated)

	// Find all
	fmt.Println("\n--- Finding all ---")
	allUsers, err := db.FindAll("users", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("All users: %v\n", allUsers)

	// Count
	fmt.Println("\n--- Counting ---")
	count := db.Count("users")
	fmt.Printf("Total users: %d\n", count)

	// List collections
	fmt.Println("\n--- Listing collections ---")
	collections, err := db.ListCollections()
	if err != nil {
		log.Fatal(err)
	}
	for _, col := range collections {
		fmt.Printf("Collection: %s, Documents: %d\n", col.Name, col.Count)
	}

	// Pagination
	fmt.Println("\n--- Pagination ---")
	limit := 1
	skip := 0
	page1, err := db.FindAll("users", &nosqlite.FindAllOptions{
		Limit: &limit,
		Skip:  &skip,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Page 1: %v\n", page1)

	skip = 1
	page2, err := db.FindAll("users", &nosqlite.FindAllOptions{
		Limit: &limit,
		Skip:  &skip,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Page 2: %v\n", page2)

	// Delete
	fmt.Println("\n--- Deleting ---")
	if err := db.Delete("users", id2); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Deleted Bob")

	fmt.Printf("Remaining users: %d\n", db.Count("users"))

	// Sync
	if err := db.Sync(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nDatabase operations completed successfully!")
}
