package product

type ProductCreated struct {
	ProductID uint64
	SKU       string
}

func (e ProductCreated) EventName() string   { return "product.created" }
func (e ProductCreated) AggregateID() uint64 { return e.ProductID }

type ProductRepriced struct {
	ProductID uint64
}

func (e ProductRepriced) EventName() string   { return "product.repriced" }
func (e ProductRepriced) AggregateID() uint64 { return e.ProductID }
