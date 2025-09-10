module kuzu_ollama_example

go 1.24.0

replace github.com/soundprediction/go-graphiti => ../..

require github.com/soundprediction/go-graphiti v0.0.0-00010101000000-000000000000

require (
	github.com/neo4j/neo4j-go-driver/v5 v5.28.0 // indirect
	github.com/sashabaranov/go-openai v1.38.0 // indirect
)
