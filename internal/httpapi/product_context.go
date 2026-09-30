package httpapi

import "context"

type productContextKey struct{}

func withProduct(ctx context.Context, product string) context.Context {
	return context.WithValue(ctx, productContextKey{}, product)
}

func productFromContext(ctx context.Context) (string, bool) {
	product, ok := ctx.Value(productContextKey{}).(string)
	return product, ok && product != ""
}
