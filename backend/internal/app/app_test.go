package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"products-orchestio-li/backend/internal/model"
	"products-orchestio-li/backend/internal/store"
)

func login(t *testing.T, h http.Handler, email, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login failed with status %d: %s", rr.Code, rr.Body.String())
	}

	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	token, ok := out["token"].(string)
	if !ok || token == "" {
		t.Fatalf("missing login token: %v", out)
	}
	return token
}

func gql(t *testing.T, h http.Handler, token, query string, vars map[string]any) map[string]any {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"query": query, "variables": vars})
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	var out map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	out["__status"] = rr.Code
	return out
}

func TestAuthAndUserRequirementsFlow(t *testing.T) {
	s := store.NewMemoryStore()
	_, err := s.EnsureUser(context.Background(), model.CreateUserInput{Email: "admin@example.com", Password: "secret123"})
	if err != nil {
		t.Fatal(err)
	}

	a, err := New(s, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	h := a.Router()

	// R1 auth: GraphQL access requires login token.
	unauthorized := gql(t, h, "", `query { products { id } }`, nil)
	if unauthorized["__status"].(int) != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth token, got %v", unauthorized["__status"])
	}

	token := login(t, h, "admin@example.com", "secret123")

	// Users CRUD requirement: list/edit/delete users.
	createUser := gql(t, h, token, `mutation($input:CreateUserInput!){createUser(input:$input){id email}}`, map[string]any{
		"input": map[string]any{"email": "editor@example.com", "password": "editor123"},
	})
	if createUser["errors"] != nil {
		t.Fatalf("createUser failed: %v", createUser["errors"])
	}
	userID := createUser["data"].(map[string]any)["createUser"].(map[string]any)["id"].(string)

	listUsers := gql(t, h, token, `query { users { id email } }`, nil)
	if listUsers["errors"] != nil {
		t.Fatalf("users query failed: %v", listUsers["errors"])
	}
	users := listUsers["data"].(map[string]any)["users"].([]any)
	if len(users) < 2 {
		t.Fatalf("expected at least 2 users, got %d", len(users))
	}

	updateUser := gql(t, h, token, `mutation($id:ID!, $input:UpdateUserInput!){updateUser(id:$id, input:$input){id email}}`, map[string]any{
		"id":    userID,
		"input": map[string]any{"email": "updated@example.com"},
	})
	if updateUser["errors"] != nil {
		t.Fatalf("updateUser failed: %v", updateUser["errors"])
	}

	deleteUser := gql(t, h, token, `mutation($id:ID!){deleteUser(id:$id)}`, map[string]any{"id": userID})
	if deleteUser["errors"] != nil {
		t.Fatalf("deleteUser failed: %v", deleteUser["errors"])
	}
	if !deleteUser["data"].(map[string]any)["deleteUser"].(bool) {
		t.Fatalf("expected deleteUser true")
	}
}

func TestProductRequirementsFlow(t *testing.T) {
	s := store.NewMemoryStore()
	_, err := s.EnsureUser(context.Background(), model.CreateUserInput{Email: "admin@example.com", Password: "secret123"})
	if err != nil {
		t.Fatal(err)
	}

	a, err := New(s, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	h := a.Router()
	token := login(t, h, "admin@example.com", "secret123")

	createMutation := `mutation($input:ProductInput!){
		createProduct(input:$input){
			id name purchaseLink shopLink booqableLink manualLink inspectionLink description status
		}
	}`

	create := gql(t, h, token, createMutation, map[string]any{"input": map[string]any{
		"name":           "MacBook Pro 14",
		"purchaseLink":   "https://example.com/purchase",
		"shopLink":       "https://example.com/shop",
		"booqableLink":   "https://example.com/booqable",
		"manualLink":     "https://example.com/manual",
		"inspectionLink": "https://example.com/inspection",
		"description":    "Main product description",
		"status":         "mafo",
	}})
	if create["errors"] != nil {
		t.Fatalf("createProduct failed: %v", create["errors"])
	}
	created := create["data"].(map[string]any)["createProduct"].(map[string]any)
	if created["status"] != "mafo" {
		t.Fatalf("unexpected initial status: %v", created["status"])
	}
	id := created["id"].(string)

	updateMutation := `mutation($id:ID!, $input:ProductInput!){
		updateProduct(id:$id, input:$input){ id name status description }
	}`
	update := gql(t, h, token, updateMutation, map[string]any{"id": id, "input": map[string]any{
		"name":           "MacBook Pro 14 (updated)",
		"purchaseLink":   "https://example.com/purchase2",
		"shopLink":       "https://example.com/shop2",
		"booqableLink":   "https://example.com/booqable2",
		"manualLink":     "https://example.com/manual2",
		"inspectionLink": "https://example.com/inspection2",
		"description":    "Updated description",
		"status":         "write-manual",
	}})
	if update["errors"] != nil {
		t.Fatalf("updateProduct failed: %v", update["errors"])
	}

	queryList := gql(t, h, token, `query { products { id name status description } }`, nil)
	if queryList["errors"] != nil {
		t.Fatalf("products query failed: %v", queryList["errors"])
	}
	products := queryList["data"].(map[string]any)["products"].([]any)
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}

	queryDetail := gql(t, h, token, `query($id:ID!){ product(id:$id){ id name purchaseLink shopLink booqableLink manualLink inspectionLink description status } }`, map[string]any{"id": id})
	if queryDetail["errors"] != nil {
		t.Fatalf("product(id) query failed: %v", queryDetail["errors"])
	}
	detail := queryDetail["data"].(map[string]any)["product"].(map[string]any)
	if detail["name"] != "MacBook Pro 14 (updated)" {
		t.Fatalf("detail view returned wrong product: %v", detail)
	}

	updateDone := gql(t, h, token, updateMutation, map[string]any{"id": id, "input": map[string]any{
		"name":           "MacBook Pro 14 (done)",
		"purchaseLink":   "https://example.com/purchase3",
		"shopLink":       "https://example.com/shop3",
		"booqableLink":   "https://example.com/booqable3",
		"manualLink":     "https://example.com/manual3",
		"inspectionLink": "https://example.com/inspection3",
		"description":    "Done description",
		"status":         "all-done",
	}})
	if updateDone["errors"] != nil {
		t.Fatalf("updateProduct all-done failed: %v", updateDone["errors"])
	}

	invalidStatus := gql(t, h, token, updateMutation, map[string]any{"id": id, "input": map[string]any{
		"name":           "Invalid",
		"purchaseLink":   "https://example.com/purchase",
		"shopLink":       "https://example.com/shop",
		"booqableLink":   "https://example.com/booqable",
		"manualLink":     "https://example.com/manual",
		"inspectionLink": "https://example.com/inspection",
		"description":    "Invalid status",
		"status":         "unknown",
	}})
	if invalidStatus["errors"] == nil {
		t.Fatalf("expected invalid status error")
	}

	deleteProduct := gql(t, h, token, `mutation($id:ID!){ deleteProduct(id:$id) }`, map[string]any{"id": id})
	if deleteProduct["errors"] != nil {
		t.Fatalf("deleteProduct failed: %v", deleteProduct["errors"])
	}
	if !deleteProduct["data"].(map[string]any)["deleteProduct"].(bool) {
		t.Fatalf("expected deleteProduct to return true")
	}
}
