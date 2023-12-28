package azureaisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type IndexOption func(indexMap *map[string]interface{})

const (
	vectorDimension              = 1536
	hnswParametersM              = 4
	hnswParametersEfConstruction = 400
	hnswParametersEfSearch       = 500
)

func (s *Store) CreateIndex(ctx context.Context, indexName string, opts ...IndexOption) error {
	defaultIndex := map[string]interface{}{
		"name": indexName,
		"fields": []map[string]interface{}{
			{
				"key":        true,
				"name":       "id",
				"type":       FieldTypeString,
				"filterable": true,
			},
			{
				"name":       "content",
				"type":       FieldTypeString,
				"searchable": true,
			},
			{
				"name":       "contentVector",
				"type":       CollectionField(FieldTypeSingle),
				"searchable": true,
				// dimensions is the number of dimensions generated by the embedding model. For text-embedding-ada-002, it's 1536.
				// basically the length of the array returned by the function
				"dimensions":          vectorDimension,
				"vectorSearchProfile": "default",
			},
			{
				"name":       "metadata",
				"type":       FieldTypeString,
				"searchable": true,
			},
		},
		"vectorSearch": map[string]interface{}{
			"algorithms": []map[string]interface{}{
				{
					"name": "default-hnsw",
					"kind": "hnsw",
					"hnswParameters": map[string]interface{}{
						"m":              hnswParametersM,
						"efConstruction": hnswParametersEfConstruction,
						"efSearch":       hnswParametersEfSearch,
						"metric":         "cosine",
					},
				},
			},
			"profiles": []map[string]interface{}{
				{
					"name":      "default",
					"algorithm": "default-hnsw",
				},
			},
		},
	}

	for _, indexOption := range opts {
		indexOption(&defaultIndex)
	}

	if err := s.CreateIndexAPIRequest(ctx, indexName, defaultIndex); err != nil {
		return fmt.Errorf("error creating index: %w", err)
	}

	return nil
}

func (s *Store) CreateIndexAPIRequest(ctx context.Context, indexName string, payload any) error {
	URL := fmt.Sprintf("%s/indexes/%s?api-version=2023-11-01", s.azureAISearchEndpoint, indexName)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("err marshalling json: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, URL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("err setting request for index creating: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	if s.azureAISearchAPIKey != "" {
		req.Header.Add("api-key", s.azureAISearchAPIKey)
	}

	if err := s.HTTPDefaultSend(req, "index creating for azure ai search", nil); err != nil {
		return fmt.Errorf("err request: %w", err)
	}

	return nil
}
