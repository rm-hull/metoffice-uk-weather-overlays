package metoffice

import "time"

type Order struct {
	OrderId            string   `json:"orderId"`
	Name               string   `json:"name"`
	Description        *string  `json:"description,omitempty"`
	ModelId            string   `json:"modelId"`
	RequiredLatestRuns []string `json:"requiredLatestRuns,omitempty"`
	Regions            []Region `json:"regions,omitempty"`
	Timesteps          []string `json:"timesteps,omitempty"`
	Format             string   `json:"format"`
	ProductType        *string   `json:"productType,omitempty"`
}

type Region struct {
	Name   string          `json:"name"`
	Extent map[string]Axis `json:"extent,omitempty"`
}

type Axis struct {
	Label      string `json:"label"`
	LowerBound string `json:"lowerBound"`
	UpperBound string `json:"upperBound"`
	UomLabel   string `json:"uomLabel"`
}

type File struct {
	FileId      string    `json:"fileId"`
	SurfaceId   *string   `json:"surfaceId,omitempty"`
	Levels      []string  `json:"levels,omitempty"`
	RunDateTime time.Time `json:"runDateTime"`
	Run         string    `json:"run"`
	Region      *Region   `json:"region,omitempty"`
}

type LatestOrderResponse struct {
	OrderDetails struct {
		Order Order  `json:"order"`
		Files []File `json:"files"`
	} `json:"orderDetails"`
}

type AllOrdersResponse struct {
	Orders []Order `json:"orders"`
}
