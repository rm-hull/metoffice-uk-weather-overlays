package metoffice

import "time"

type Order struct {
	OrderId     string `json:"orderId"`
	Name        string `json:"name"`
	ModelId     string `json:"modelId"`
	Format      string `json:"format"`
	ProductType string `json:"productType"`
}

type File struct {
	FileId      string    `json:"fileId"`
	RunDateTime time.Time `json:"runDateTime"`
	Run         string    `json:"run"`
}

type OrderDetails struct {
	Order Order  `json:"order"`
	Files []File `json:"files"`
}

type Response struct {
	OrderDetails OrderDetails `json:"orderDetails"`
}
