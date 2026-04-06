package providers

import (
	"github.com/google/wire"
	"github.com/gofastadev/gofasta/app/graphql/resolvers"
)

// GraphQLSet provides the GraphQL resolver.
var GraphQLSet = wire.NewSet(
	resolvers.NewResolver,
)
