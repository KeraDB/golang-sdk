package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"

	"github.com/keradb/golang-sdk"
)

// generateRandomEmbedding creates a random normalized vector
func generateRandomEmbedding(dimensions int) keradb.Embedding {
	embedding := make(keradb.Embedding, dimensions)
	var sumSquares float32 = 0

	for i := 0; i < dimensions; i++ {
		val := rand.Float32()*2 - 1 // Random value between -1 and 1
		embedding[i] = val
		sumSquares += val * val
	}

	// Normalize the vector
	norm := float32(math.Sqrt(float64(sumSquares)))
	for i := 0; i < dimensions; i++ {
		embedding[i] /= norm
	}

	return embedding
}

// addNoise adds small random perturbations to create similar vectors
func addNoise(embedding keradb.Embedding, noiseLevel float32) keradb.Embedding {
	result := make(keradb.Embedding, len(embedding))
	var sumSquares float32 = 0

	for i := 0; i < len(embedding); i++ {
		noise := (rand.Float32()*2 - 1) * noiseLevel
		result[i] = embedding[i] + noise
		sumSquares += result[i] * result[i]
	}

	// Re-normalize
	norm := float32(math.Sqrt(float64(sumSquares)))
	for i := 0; i < len(result); i++ {
		result[i] /= norm
	}

	return result
}

func main() {
	fmt.Println("=== KeraDB Vector Search Example ===\n")

	// 1. Connect to database
	fmt.Println("1. Connecting to database...")
	client, err := keradb.Connect("vector_demo.ndb")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	fmt.Println("   ✓ Connected\n")

	// 2. Create a vector collection
	fmt.Println("2. Creating vector collection...")
	dimensions := 128
	config := keradb.NewVectorConfig(dimensions).
		WithDistance(keradb.Cosine).
		WithM(16).
		WithEfConstruction(200).
		WithEfSearch(50).
		WithDeltaCompression()

	err = client.CreateVectorCollection("articles", config)
	if err != nil {
		// Collection might already exist, which is fine
		fmt.Printf("   Note: %v\n", err)
	} else {
		fmt.Println("   ✓ Collection created with delta compression")
	}
	fmt.Println()

	// 3. Insert sample vectors with metadata
	fmt.Println("3. Inserting sample article vectors...")
	articles := []struct {
		title    string
		category string
		baseVec  keradb.Embedding
	}{
		{"Introduction to Machine Learning", "tech", nil},
		{"Advanced Neural Networks", "tech", nil},
		{"Deep Learning Fundamentals", "tech", nil},
		{"Cooking Italian Pasta", "food", nil},
		{"Mediterranean Diet Guide", "food", nil},
		{"Travel Guide to Tokyo", "travel", nil},
		{"European Travel Tips", "travel", nil},
		{"AI Ethics and Society", "tech", nil},
	}

	// Generate base vectors for each category
	techBase := generateRandomEmbedding(dimensions)
	foodBase := generateRandomEmbedding(dimensions)
	travelBase := generateRandomEmbedding(dimensions)

	for i := range articles {
		var embedding keradb.Embedding
		switch articles[i].category {
		case "tech":
			articles[i].baseVec = techBase
			embedding = addNoise(techBase, 0.1)
		case "food":
			articles[i].baseVec = foodBase
			embedding = addNoise(foodBase, 0.1)
		case "travel":
			articles[i].baseVec = travelBase
			embedding = addNoise(travelBase, 0.1)
		}

		metadata := keradb.M{
			"title":    articles[i].title,
			"category": articles[i].category,
			"index":    i,
		}

		id, err := client.InsertVector("articles", embedding, metadata)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   ✓ Inserted: %s (ID: %d)\n", articles[i].title, id)
	}
	fmt.Println()

	// 4. Perform similarity search
	fmt.Println("4. Searching for articles similar to 'tech' category...")
	queryVector := addNoise(techBase, 0.05) // Similar to tech articles
	results, err := client.VectorSearch("articles", queryVector, 5)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Top 5 similar articles:")
	for _, result := range results {
		title := result.Document.Metadata["title"]
		category := result.Document.Metadata["category"]
		fmt.Printf("   %d. [%.4f] %s (%s)\n",
			result.Rank, result.Score, title, category)
	}
	fmt.Println()

	// 5. Filtered search
	fmt.Println("5. Searching within 'food' category only...")
	filter := keradb.MetadataFilter{
		Field:     "category",
		Condition: "eq",
		Value:     "food",
	}

	foodQuery := addNoise(foodBase, 0.05)
	filteredResults, err := client.VectorSearchFiltered("articles", foodQuery, 3, filter)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Food articles:")
	for _, result := range filteredResults {
		title := result.Document.Metadata["title"]
		fmt.Printf("   %d. [%.4f] %s\n",
			result.Rank, result.Score, title)
	}
	fmt.Println()

	// 6. Get vector by ID
	fmt.Println("6. Retrieving vector by ID...")
	doc, err := client.GetVector("articles", results[0].Document.ID)
	if err != nil {
		log.Fatal(err)
	}
	if doc != nil {
		fmt.Printf("   ✓ Retrieved: %s\n", doc.Metadata["title"])
		fmt.Printf("     ID: %d\n", doc.ID)
		fmt.Printf("     Has embedding: %v\n", doc.Embedding != nil)
		fmt.Printf("     Metadata fields: %d\n", len(doc.Metadata))
	}
	fmt.Println()

	// 7. Collection statistics
	fmt.Println("7. Vector collection statistics...")
	stats, err := client.VectorStats("articles")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Vectors: %d\n", stats.VectorCount)
	fmt.Printf("   Dimensions: %d\n", stats.Dimensions)
	fmt.Printf("   Distance metric: %s\n", stats.Distance)
	fmt.Printf("   HNSW layers: %d\n", stats.LayerCount)
	fmt.Printf("   Memory usage: %d bytes\n", stats.MemoryUsage)
	if stats.Compression != nil {
		fmt.Printf("   Compression: %s\n", *stats.Compression)
		if stats.AnchorCount != nil && stats.DeltaCount != nil {
			total := *stats.AnchorCount + *stats.DeltaCount
			compressionRatio := float64(*stats.DeltaCount) / float64(total) * 100
			fmt.Printf("   Anchors: %d, Deltas: %d (%.1f%% compressed)\n",
				*stats.AnchorCount, *stats.DeltaCount, compressionRatio)
		}
	}
	fmt.Println()

	// 8. List all vector collections
	fmt.Println("8. Listing vector collections...")
	collections, err := client.ListVectorCollections()
	if err != nil {
		log.Fatal(err)
	}

	for _, coll := range collections {
		fmt.Printf("   - %s: %d vectors\n", coll.Name, coll.Count)
	}
	fmt.Println()

	// 9. Delete a vector
	fmt.Println("9. Deleting a vector...")
	deleted, err := client.DeleteVector("articles", results[len(results)-1].Document.ID)
	if err != nil {
		log.Fatal(err)
	}
	if deleted {
		fmt.Println("   ✓ Vector deleted")
	}
	fmt.Println()

	// 10. Sync to disk
	fmt.Println("10. Syncing database to disk...")
	if err := client.Sync(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("    ✓ Synced\n")

	fmt.Println("=== Example Complete ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("  ✓ Vector collection creation with HNSW index")
	fmt.Println("  ✓ Delta compression for storage efficiency")
	fmt.Println("  ✓ Similarity search with cosine distance")
	fmt.Println("  ✓ Metadata filtering for hybrid search")
	fmt.Println("  ✓ Vector CRUD operations")
	fmt.Println("  ✓ Collection statistics and monitoring")
}
