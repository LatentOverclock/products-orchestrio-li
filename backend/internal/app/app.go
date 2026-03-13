package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"products-orchestio-li/backend/internal/model"
	"products-orchestio-li/backend/internal/store"

	"github.com/graphql-go/graphql"
)

type App struct {
	store  store.Store
	schema graphql.Schema
}

func New(s store.Store) (*App, error) {
	schema, err := buildSchema(s)
	if err != nil {
		return nil, err
	}
	return &App{store: s, schema: schema}, nil
}

func (a *App) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc("/graphql", a.handleGraphQL)
	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := a.store.Health(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (a *App) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req graphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	result := graphql.Do(graphql.Params{
		Schema:         a.schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		Context:        r.Context(),
	})
	writeJSON(w, result)
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func ctxOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

var allowedStatuses = map[string]bool{
	"mafo":         true,
	"write-manual": true,
	"all-done":     true,
}

func normalizeLink(v interface{}) *string {
	raw, ok := v.(string)
	if !ok {
		return nil
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeInput(args map[string]any) (model.ProductInput, error) {
	name := strings.TrimSpace(fmt.Sprintf("%v", args["name"]))
	description := strings.TrimSpace(fmt.Sprintf("%v", args["description"]))
	status := strings.TrimSpace(fmt.Sprintf("%v", args["status"]))

	if name == "" {
		return model.ProductInput{}, errors.New("name is required")
	}
	if !allowedStatuses[status] {
		return model.ProductInput{}, errors.New("status must be one of: mafo, write-manual, all-done")
	}

	return model.ProductInput{
		Name:           name,
		PurchaseLink:   normalizeLink(args["purchaseLink"]),
		ShopLink:       normalizeLink(args["shopLink"]),
		BooqableLink:   normalizeLink(args["booqableLink"]),
		ManualLink:     normalizeLink(args["manualLink"]),
		InspectionLink: normalizeLink(args["inspectionLink"]),
		Description:    description,
		Status:         status,
	}, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func formatProduct(p *model.Product) map[string]any {
	return map[string]any{
		"id":             fmt.Sprintf("%d", p.ID),
		"name":           p.Name,
		"purchaseLink":   p.PurchaseLink,
		"shopLink":       p.ShopLink,
		"booqableLink":   p.BooqableLink,
		"manualLink":     p.ManualLink,
		"inspectionLink": p.InspectionLink,
		"description":    p.Description,
		"status":         p.Status,
		"createdAt":      formatTime(p.CreatedAt),
		"updatedAt":      formatTime(p.UpdatedAt),
	}
}

func buildSchema(s store.Store) (graphql.Schema, error) {
	productType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Product",
		Fields: graphql.Fields{
			"id":             &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"purchaseLink":   &graphql.Field{Type: graphql.String},
			"shopLink":       &graphql.Field{Type: graphql.String},
			"booqableLink":   &graphql.Field{Type: graphql.String},
			"manualLink":     &graphql.Field{Type: graphql.String},
			"inspectionLink": &graphql.Field{Type: graphql.String},
			"description":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"status":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"createdAt":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"updatedAt":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	productInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "ProductInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"name":           &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"purchaseLink":   &graphql.InputObjectFieldConfig{Type: graphql.String},
			"shopLink":       &graphql.InputObjectFieldConfig{Type: graphql.String},
			"booqableLink":   &graphql.InputObjectFieldConfig{Type: graphql.String},
			"manualLink":     &graphql.InputObjectFieldConfig{Type: graphql.String},
			"inspectionLink": &graphql.InputObjectFieldConfig{Type: graphql.String},
			"description":    &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"status":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"products": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(productType))),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					products, err := s.ListProducts(ctxOrBackground(p.Context))
					if err != nil {
						return nil, err
					}
					mapped := make([]map[string]any, 0, len(products))
					for i := range products {
						prod := products[i]
						mapped = append(mapped, formatProduct(&prod))
					}
					return mapped, nil
				},
			},
			"product": &graphql.Field{
				Type: productType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					idRaw := fmt.Sprintf("%v", p.Args["id"])
					id, err := strconv.ParseInt(idRaw, 10, 64)
					if err != nil {
						return nil, errors.New("invalid id")
					}
					product, err := s.GetProduct(ctxOrBackground(p.Context), id)
					if err != nil {
						if errors.Is(err, store.ErrNotFound) {
							return nil, nil
						}
						return nil, err
					}
					return formatProduct(product), nil
				},
			},
		},
	})

	mutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createProduct": &graphql.Field{
				Type: graphql.NewNonNull(productType),
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(productInput)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					input, err := normalizeInput(p.Args["input"].(map[string]any))
					if err != nil {
						return nil, err
					}
					created, err := s.CreateProduct(ctxOrBackground(p.Context), input)
					if err != nil {
						return nil, err
					}
					return formatProduct(created), nil
				},
			},
			"updateProduct": &graphql.Field{
				Type: graphql.NewNonNull(productType),
				Args: graphql.FieldConfigArgument{
					"id":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(productInput)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					idRaw := fmt.Sprintf("%v", p.Args["id"])
					id, err := strconv.ParseInt(idRaw, 10, 64)
					if err != nil {
						return nil, errors.New("invalid id")
					}
					input, err := normalizeInput(p.Args["input"].(map[string]any))
					if err != nil {
						return nil, err
					}
					updated, err := s.UpdateProduct(ctxOrBackground(p.Context), id, input)
					if err != nil {
						return nil, err
					}
					return formatProduct(updated), nil
				},
			},
			"deleteProduct": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					idRaw := fmt.Sprintf("%v", p.Args["id"])
					id, err := strconv.ParseInt(idRaw, 10, 64)
					if err != nil {
						return nil, errors.New("invalid id")
					}
					return s.DeleteProduct(ctxOrBackground(p.Context), id)
				},
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{Query: query, Mutation: mutation})
}
