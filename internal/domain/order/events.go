package order

type OrderPlaced struct {
	OrderID     uint64
	UserID      uint64
	TotalAmount int64
	Currency    string
}

func (e OrderPlaced) EventName() string  { return "order.placed" }
func (e OrderPlaced) AggregateID() uint64 { return e.OrderID }

type OrderReserved struct {
	OrderID uint64
}

func (e OrderReserved) EventName() string   { return "order.reserved" }
func (e OrderReserved) AggregateID() uint64  { return e.OrderID }

type OrderPaid struct {
	OrderID uint64
}

func (e OrderPaid) EventName() string   { return "order.paid" }
func (e OrderPaid) AggregateID() uint64 { return e.OrderID }

type OrderFulfilled struct {
	OrderID uint64
}

func (e OrderFulfilled) EventName() string   { return "order.fulfilled" }
func (e OrderFulfilled) AggregateID() uint64 { return e.OrderID }

type OrderCancelled struct {
	OrderID uint64
	Reason  string
}

func (e OrderCancelled) EventName() string   { return "order.cancelled" }
func (e OrderCancelled) AggregateID() uint64 { return e.OrderID }

type OrderFailed struct {
	OrderID uint64
	Reason  string
}

func (e OrderFailed) EventName() string   { return "order.failed" }
func (e OrderFailed) AggregateID() uint64 { return e.OrderID }
