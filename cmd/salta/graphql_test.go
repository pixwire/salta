package main

import (
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
)

func TestResolver(t *testing.T) {
	// will panic if schema is invalid
	_ = graphql.MustParseSchema(schema, &graphqlResolver{}, graphql.UseFieldResolvers())
}
