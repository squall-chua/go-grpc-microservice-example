package middleware

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockTokenValidator struct {
	tokenInfo *TokenInfo
	err       error
}

func (m *mockTokenValidator) ValidateToken(ctx context.Context, token string) (*TokenInfo, error) {
	return m.tokenInfo, m.err
}

func TestAuthInterceptor(t *testing.T) {
	// For testing AuthInterceptor we can use a dummy UnaryServerInfo.
	// We'll pass a non-existent method to bypass Protobuf descriptor extraction,
	// because the global registry might not be fully initialized in pure unit tests without importing the pb package.
	// Wait, actually let's test the pass-through behavior.
	validator := &mockTokenValidator{
		tokenInfo: &TokenInfo{
			Scopes: []string{"read"},
			Roles:  []string{"user"},
		},
	}

	interceptor := AuthInterceptor(validator)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	// 1. Unknown method (bypasses auth logic)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/UnknownService/UnknownMethod",
	}

	_, err := interceptor(context.Background(), nil, info, handler)
	if err != nil {
		t.Fatalf("expected no error for unknown method, got %v", err)
	}

	// 2. Setup a valid method from the compiled proto (CreateItem)
	// Must anonymously import pb to populate protoregistry.GlobalFiles if not already
	infoValid := &grpc.UnaryServerInfo{
		FullMethod: "/item.v1.ItemService/CreateItem", // requires write:items and admin/user
	}

	// 3. No metadata -> missing token error
	_, err = interceptor(context.Background(), nil, infoValid, handler)
	if err == nil {
		t.Fatal("expected error for missing token")
	}

	// 4. Valid token, sufficient scopes/roles
	md := metadata.Pairs("authorization", "bearer valid-token")
	ctxAuth := metadata.NewIncomingContext(context.Background(), md)

	// CreateItem expects `write:items` and either `admin` or `user`
	validator.tokenInfo.Scopes = []string{"write:items"}
	validator.tokenInfo.Roles = []string{"admin"}
	_, err = interceptor(ctxAuth, nil, infoValid, handler)
	if err != nil {
		t.Fatalf("expected no error for valid token and scopes, got %v", err)
	}

	// 5. Invalid token (validator returns error)
	validator.err = context.DeadlineExceeded
	_, err = interceptor(ctxAuth, nil, infoValid, handler)
	if err == nil {
		t.Fatal("expected error for token validation failure")
	}
	validator.err = nil

	// 6. Missing scope
	validator.tokenInfo.Scopes = []string{"read:items"} // missing write:items
	_, err = interceptor(ctxAuth, nil, infoValid, handler)
	if err == nil {
		t.Fatal("expected error for missing scope")
	}

	// 7. Missing role
	validator.tokenInfo.Scopes = []string{"write:items"}
	validator.tokenInfo.Roles = []string{"guest"} // missing admin or user
	_, err = interceptor(ctxAuth, nil, infoValid, handler)
	if err == nil {
		t.Fatal("expected error for missing role")
	}
}

func TestContextWithTokenInfo(t *testing.T) {
	ctx := context.Background()
	info := &TokenInfo{
		Scopes: []string{"read"},
		Roles:  []string{"admin"},
	}

	ctx = ContextWithTokenInfo(ctx, info)

	retrieved, ok := TokenInfoFromContext(ctx)
	if !ok {
		t.Fatal("expected to find TokenInfo in context")
	}

	if len(retrieved.Scopes) != 1 || retrieved.Scopes[0] != "read" {
		t.Errorf("unexpected scopes: %v", retrieved.Scopes)
	}
}
