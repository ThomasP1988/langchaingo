package selfquery

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

type FromLLMArgs struct {
	LLM               llms.Model
	Store             StoreWithQueryTranslator
	DocumentContents  string
	MetadataFieldInfo []schema.AttributeInfo
	EnableLimit       *bool
	DefaultLimit      *int
}

// create retriever with LLM.
func FromLLM(args FromLLMArgs) *Retriever {
	retriever := Retriever{
		Store:             args.Store,
		LLM:               args.LLM,
		DocumentContents:  args.DocumentContents,
		MetadataFieldInfo: args.MetadataFieldInfo,
		EnableLimit:       args.EnableLimit,
	}

	if args.DefaultLimit != nil {
		retriever.DefaultLimit = *args.DefaultLimit
	} else {
		retriever.DefaultLimit = 10
	}

	return &retriever
}