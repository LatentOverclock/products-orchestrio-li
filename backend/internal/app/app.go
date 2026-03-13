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

	"github.com/golang-jwt/jwt/v5"
	"github.com/graphql-go/graphql"
)

type App struct {
	store     store.Store
	schema    graphql.Schema
	jwtSecret []byte
}

func New(s store.Store, jwtSecret string) (*App, error) {
	if strings.TrimSpace(jwtSecret) == "" {
		return nil, errors.New("jwt secret is required")
	}

	schema, err := buildSchema(s)
	if err != nil {
		return nil, err
	}
	return &App{store: s, schema: schema, jwtSecret: []byte(jwtSecret)}, nil
}

func (a *App) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc("/login", a.handleLogin)
	mux.HandleFunc("/graphql", a.handleGraphQL)
	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authClaims struct {
	UserID int64  `json:"uid"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (a *App) issueToken(user *model.User) (string, error) {
	now := time.Now().UTC()
	claims := authClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

func (a *App) parseTokenFromRequest(r *http.Request) (*authClaims, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return nil, errors.New("missing authorization header")
	}
	prefix := "Bearer "
	if !strings.HasPrefix(strings.ToLower(authHeader), strings.ToLower(prefix)) {
		return nil, errors.New("invalid authorization header")
	}
	rawToken := strings.TrimSpace(authHeader[len(prefix):])
	if rawToken == "" {
		return nil, errors.New("missing bearer token")
	}

	parsed, err := jwt.ParseWithClaims(rawToken, &authClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.jwtSecret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := parsed.Claims.(*authClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	user, err := a.store.AuthenticateUser(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, store.ErrInvalidCredentials) {
			writeGraphQLError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeGraphQLError(w, http.StatusInternalServerError, err.Error())
		return
	}

	token, err := a.issueToken(user)
	if err != nil {
		writeGraphQLError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, map[string]any{
		"token": token,
		"user":  formatUser(user),
	})
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

	claims, err := a.parseTokenFromRequest(r)
	if err != nil {
		writeGraphQLError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if _, err := a.store.GetUser(r.Context(), claims.UserID); err != nil {
		writeGraphQLError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req graphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	ctx := context.WithValue(r.Context(), ctxUserIDKey{}, claims.UserID)
	result := graphql.Do(graphql.Params{
		Schema:         a.schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		Context:        ctx,
	})
	writeJSON(w, result)
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func writeGraphQLError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	writeJSON(w, map[string]any{
		"errors": []map[string]string{{"message": message}},
	})
}

type ctxUserIDKey struct{}

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

func normalizeProductInput(args map[string]any) (model.ProductInput, error) {
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

func normalizeCreateUserInput(args map[string]any) (model.CreateUserInput, error) {
	email := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", args["email"])))
	password := strings.TrimSpace(fmt.Sprintf("%v", args["password"]))

	if email == "" {
		return model.CreateUserInput{}, errors.New("email is required")
	}
	if password == "" {
		return model.CreateUserInput{}, errors.New("password is required")
	}
	if len(password) < 6 {
		return model.CreateUserInput{}, errors.New("password must be at least 6 characters")
	}

	return model.CreateUserInput{Email: email, Password: password}, nil
}

func normalizeUpdateUserInput(args map[string]any) (model.UpdateUserInput, error) {
	var input model.UpdateUserInput

	if rawEmail, exists := args["email"]; exists {
		email := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", rawEmail)))
		if email == "" {
			return model.UpdateUserInput{}, errors.New("email cannot be empty")
		}
		input.Email = &email
	}

	if rawPassword, exists := args["password"]; exists {
		password := strings.TrimSpace(fmt.Sprintf("%v", rawPassword))
		if password != "" && len(password) < 6 {
			return model.UpdateUserInput{}, errors.New("password must be at least 6 characters")
		}
		input.Password = &password
	}

	if input.Email == nil && input.Password == nil {
		return model.UpdateUserInput{}, errors.New("at least one field (email or password) must be provided")
	}

	return input, nil
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

func formatUser(u *model.User) map[string]any {
	return map[string]any{
		"id":        fmt.Sprintf("%d", u.ID),
		"email":     u.Email,
		"createdAt": formatTime(u.CreatedAt),
		"updatedAt": formatTime(u.UpdatedAt),
	}
}

func parseID(raw any) (int64, error) {
	idRaw := fmt.Sprintf("%v", raw)
	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil {
		return 0, errors.New("invalid id")
	}
	return id, nil
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

	userType := graphql.NewObject(graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id":        &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"email":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"createdAt": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"updatedAt": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
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

	createUserInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "CreateUserInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"email":    &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"password": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	updateUserInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "UpdateUserInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"email":    &graphql.InputObjectFieldConfig{Type: graphql.String},
			"password": &graphql.InputObjectFieldConfig{Type: graphql.String},
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
					id, err := parseID(p.Args["id"])
					if err != nil {
						return nil, err
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
			"users": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(userType))),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					users, err := s.ListUsers(ctxOrBackground(p.Context))
					if err != nil {
						return nil, err
					}
					mapped := make([]map[string]any, 0, len(users))
					for i := range users {
						u := users[i]
						mapped = append(mapped, formatUser(&u))
					}
					return mapped, nil
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
					input, err := normalizeProductInput(p.Args["input"].(map[string]any))
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
					id, err := parseID(p.Args["id"])
					if err != nil {
						return nil, err
					}
					input, err := normalizeProductInput(p.Args["input"].(map[string]any))
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
					id, err := parseID(p.Args["id"])
					if err != nil {
						return nil, err
					}
					return s.DeleteProduct(ctxOrBackground(p.Context), id)
				},
			},
			"createUser": &graphql.Field{
				Type: graphql.NewNonNull(userType),
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createUserInput)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					input, err := normalizeCreateUserInput(p.Args["input"].(map[string]any))
					if err != nil {
						return nil, err
					}
					created, err := s.CreateUser(ctxOrBackground(p.Context), input)
					if err != nil {
						if errors.Is(err, store.ErrEmailAlreadyExists) {
							return nil, errors.New("email already exists")
						}
						return nil, err
					}
					return formatUser(created), nil
				},
			},
			"updateUser": &graphql.Field{
				Type: graphql.NewNonNull(userType),
				Args: graphql.FieldConfigArgument{
					"id":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateUserInput)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id, err := parseID(p.Args["id"])
					if err != nil {
						return nil, err
					}
					input, err := normalizeUpdateUserInput(p.Args["input"].(map[string]any))
					if err != nil {
						return nil, err
					}
					updated, err := s.UpdateUser(ctxOrBackground(p.Context), id, input)
					if err != nil {
						if errors.Is(err, store.ErrEmailAlreadyExists) {
							return nil, errors.New("email already exists")
						}
						return nil, err
					}
					return formatUser(updated), nil
				},
			},
			"deleteUser": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id, err := parseID(p.Args["id"])
					if err != nil {
						return nil, err
					}
					return s.DeleteUser(ctxOrBackground(p.Context), id)
				},
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{Query: query, Mutation: mutation})
}
