package azureaisearch_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/azureaisearch"
)

func getValues(t *testing.T) string {
	t.Helper()

	azureaisearchEndpoint := os.Getenv(azureaisearch.EnvironmentVariable_Endpoint)
	if azureaisearchEndpoint == "" {
		t.Fatalf("Must set %s to run test", azureaisearch.EnvironmentVariable_Endpoint)
	}

	azureaisearchAPIKey := os.Getenv(azureaisearch.EnvironmentVariable_APIKey)
	if azureaisearchAPIKey == "" {
		t.Fatalf("Must set %s to run test", azureaisearch.EnvironmentVariable_APIKey)
	}

	indexName := os.Getenv("COGNITIVE_SEARCH_INDEX")
	if indexName == "" {
		t.Fatal("Must set COGNITIVE_SEARCH_INDEX to run test")
	}

	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey == "" {
		fmt.Printf("openaiKey: %v\n", openaiKey)
		t.Fatal("OPENAI_API_KEY not set")
	}

	return indexName
}

func setIndex(t *testing.T, storer azureaisearch.Store, indexName string) {
	t.Helper()
	if err := storer.CreateIndex(context.TODO(), indexName); err != nil {
		t.Fatalf("error creating index: %v\n", err)
	}
}

func removeIndex(t *testing.T, storer azureaisearch.Store, indexName string) {
	t.Helper()
	if err := storer.DeleteIndex(context.TODO(), indexName); err != nil {
		t.Fatalf("error deleting index: %v\n", err)
	}
}

func setLLM(t *testing.T) *openai.LLM {
	t.Helper()
	openaiOpts := []openai.Option{}

	if openAIBaseURL := os.Getenv("OPENAI_BASE_URL"); openAIBaseURL != "" {
		openaiOpts = append(openaiOpts,
			openai.WithBaseURL(openAIBaseURL),
			openai.WithAPIType(openai.APITypeAzure),
			openai.WithEmbeddingModel("text-embedding-ada-002"),
			openai.WithModel("gpt-4"),
		)
	}

	llm, err := openai.New(openaiOpts...)
	if err != nil {
		t.Fatalf("error setting openAI embedded: %v\n", err)
	}

	return llm
}

func TestCognitivesearchStoreRest(t *testing.T) {
	indexName := getValues(t)
	llm := setLLM(t)
	e, err := embeddings.NewEmbedder(llm)
	require.NoError(t, err)

	storer, err := azureaisearch.New(
		context.Background(),
		azureaisearch.WithEmbedder(e),
	)
	require.NoError(t, err)

	setIndex(t, storer, indexName)
	defer removeIndex(t, storer, indexName)

	err = storer.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "tokyo"},
		{PageContent: "potato"},
	}, vectorstores.WithNameSpace(indexName))
	require.NoError(t, err)

	docs, err := storer.SimilaritySearch(context.Background(), "japan", 1, vectorstores.WithNameSpace(indexName))
	require.NoError(t, err)
	fmt.Printf("docs: %v\n", len(docs))
	require.Len(t, docs, 1)
	require.Equal(t, "tokyo", docs[0].PageContent)
}

func TestCognitivesearchStoreRestWithScoreThreshold(t *testing.T) {
	indexName := getValues(t)

	llm := setLLM(t)
	e, err := embeddings.NewEmbedder(llm)
	require.NoError(t, err)

	storer, err := azureaisearch.New(
		context.Background(),
		azureaisearch.WithEmbedder(e),
	)
	require.NoError(t, err)

	setIndex(t, storer, indexName)
	defer removeIndex(t, storer, indexName)

	err = storer.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "Tokyo"},
		{PageContent: "Yokohama"},
		{PageContent: "Osaka"},
		{PageContent: "Nagoya"},
		{PageContent: "Sapporo"},
		{PageContent: "Fukuoka"},
		{PageContent: "Dublin"},
		{PageContent: "Paris"},
		{PageContent: "London "},
		{PageContent: "New York"},
	}, vectorstores.WithNameSpace(indexName))
	require.NoError(t, err)
	time.Sleep(time.Second)
	// test with a score threshold of 0.84, expected 6 documents
	docs, err := storer.SimilaritySearch(context.Background(),
		"Which of these are cities in Japan", 10,
		vectorstores.WithScoreThreshold(0.84),
		vectorstores.WithNameSpace(indexName))
	require.NoError(t, err)
	require.Len(t, docs, 6)

}

func TestCognitivesearchAsRetriever(t *testing.T) {

	indexName := getValues(t)

	llm := setLLM(t)
	e, err := embeddings.NewEmbedder(llm)
	require.NoError(t, err)

	storer, err := azureaisearch.New(
		context.Background(),
		azureaisearch.WithEmbedder(e),
	)
	require.NoError(t, err)

	setIndex(t, storer, indexName)
	defer removeIndex(t, storer, indexName)

	err = storer.AddDocuments(
		context.Background(),
		[]schema.Document{
			{PageContent: "The color of the house is blue."},
			{PageContent: "The color of the car is red."},
			{PageContent: "The color of the desk is orange."},
		},
		vectorstores.WithNameSpace(indexName),
	)
	require.NoError(t, err)

	time.Sleep(time.Second) // let the time for azure search to digest sent documents, otherwise only "The color of the house is blue." is fetched

	result, err := chains.Run(
		context.TODO(),
		chains.NewRetrievalQAFromLLM(
			llm,
			vectorstores.ToRetriever(storer, 1, vectorstores.WithNameSpace(indexName)),
		),
		"What color is the desk?",
	)
	require.NoError(t, err)
	require.True(t, strings.Contains(result, "orange"), "expected orange in result")
}

func TestCognitivesearchAsRetrieverWithScoreThreshold(t *testing.T) {

	indexName := getValues(t)

	llm := setLLM(t)
	e, err := embeddings.NewEmbedder(llm)
	require.NoError(t, err)

	storer, err := azureaisearch.New(
		context.Background(),
		azureaisearch.WithEmbedder(e),
	)
	require.NoError(t, err)

	setIndex(t, storer, indexName)
	defer removeIndex(t, storer, indexName)

	err = storer.AddDocuments(
		context.Background(),
		[]schema.Document{
			{PageContent: "The color of the house is blue."},
			{PageContent: "The color of the car is red."},
			{PageContent: "The color of the desk is orange."},
			{PageContent: "The color of the lamp beside the desk is black."},
			{PageContent: "The color of the chair beside the desk is beige."},
		},
		vectorstores.WithNameSpace(indexName),
	)
	require.NoError(t, err)
	time.Sleep(time.Second)
	result, err := chains.Run(
		context.TODO(),
		chains.NewRetrievalQAFromLLM(
			llm,
			vectorstores.ToRetriever(storer, 5,
				vectorstores.WithNameSpace(indexName),
				vectorstores.WithScoreThreshold(0.8)),
		),
		"What colors is each piece of furniture next to the desk?",
	)
	require.NoError(t, err)

	require.Contains(t, result, "black", "expected black in result")
	require.Contains(t, result, "beige", "expected beige in result")
}