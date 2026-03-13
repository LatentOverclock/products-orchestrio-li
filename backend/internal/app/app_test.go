package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"products-orchestio-li/backend/internal/store"
)

func gql(t *testing.T, h http.Handler, query string, vars map[string]any) map[string]any {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"query": query, "variables": vars})
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	var out map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	return out
}

func TestProductRequirementsFlow(t *testing.T) {
	s := store.NewMemoryStore()
	a, err := New(s)
	if err != nil {
		t.Fatal(err)
	}
	h := a.Router()

	createMutation := `mutation($input:ProductInput!){
		createProduct(input:$input){
			id name purchaseLink shopLink booqableLink manualLink inspectionLink description status
		}
	}`

	// R1 + R2 + R3: create a product with required properties and valid status.
	create := gql(t, h, createMutation, map[string]any{"input": map[string]any{
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

	// R3: edit product and move status through workflow.
	updateMutation := `mutation($id:ID!, $input:ProductInput!){
		updateProduct(id:$id, input:$input){ id name status description }
	}`
	update := gql(t, h, updateMutation, map[string]any{"id": id, "input": map[string]any{
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
	updated := update["data"].(map[string]any)["updateProduct"].(map[string]any)
	if updated["status"] != "write-manual" {
		t.Fatalf("expected status write-manual, got %v", updated["status"])
	}

	// R2 + R5 + R6: list products and fetch detail view by id.
	queryList := gql(t, h, `query { products { id name status description } }`, nil)
	if queryList["errors"] != nil {
		t.Fatalf("products query failed: %v", queryList["errors"])
	}
	products := queryList["data"].(map[string]any)["products"].([]any)
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}

	queryDetail := gql(t, h, `query($id:ID!){ product(id:$id){ id name purchaseLink shopLink booqableLink manualLink inspectionLink description status } }`, map[string]any{"id": id})
	if queryDetail["errors"] != nil {
		t.Fatalf("product(id) query failed: %v", queryDetail["errors"])
	}
	detail := queryDetail["data"].(map[string]any)["product"].(map[string]any)
	if detail["name"] != "MacBook Pro 14 (updated)" {
		t.Fatalf("detail view returned wrong product: %v", detail)
	}

	// R2: allowed status all-done should be accepted.
	updateDone := gql(t, h, updateMutation, map[string]any{"id": id, "input": map[string]any{
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

	// R2: invalid status should be rejected.
	invalidStatus := gql(t, h, updateMutation, map[string]any{"id": id, "input": map[string]any{
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

	// R3: delete product.
	delete := gql(t, h, `mutation($id:ID!){ deleteProduct(id:$id) }`, map[string]any{"id": id})
	if delete["errors"] != nil {
		t.Fatalf("deleteProduct failed: %v", delete["errors"])
	}
	if !delete["data"].(map[string]any)["deleteProduct"].(bool) {
		t.Fatalf("expected deleteProduct to return true")
	}

	listAfterDelete := gql(t, h, `query { products { id } }`, nil)
	remaining := listAfterDelete["data"].(map[string]any)["products"].([]any)
	if len(remaining) != 0 {
		t.Fatalf("expected 0 products after delete, got %d", len(remaining))
	}
}
