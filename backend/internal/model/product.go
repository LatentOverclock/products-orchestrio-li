package model

import "time"

type Product struct {
	ID             int64
	Name           string
	PurchaseLink   *string
	ShopLink       *string
	BooqableLink   *string
	ManualLink     *string
	InspectionLink *string
	Description    string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ProductInput struct {
	Name           string
	PurchaseLink   *string
	ShopLink       *string
	BooqableLink   *string
	ManualLink     *string
	InspectionLink *string
	Description    string
	Status         string
}
