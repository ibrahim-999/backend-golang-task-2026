package payment

type PaymentInitiated struct {
	PaymentID uint64
	OrderID   uint64
}

func (e PaymentInitiated) EventName() string   { return "payment.initiated" }
func (e PaymentInitiated) AggregateID() uint64 { return e.PaymentID }

type PaymentSucceeded struct {
	PaymentID uint64
	OrderID   uint64
}

func (e PaymentSucceeded) EventName() string   { return "payment.succeeded" }
func (e PaymentSucceeded) AggregateID() uint64 { return e.PaymentID }

type PaymentFailed struct {
	PaymentID uint64
	OrderID   uint64
	Reason    string
}

func (e PaymentFailed) EventName() string   { return "payment.failed" }
func (e PaymentFailed) AggregateID() uint64 { return e.PaymentID }

type PaymentRefunded struct {
	PaymentID uint64
	OrderID   uint64
}

func (e PaymentRefunded) EventName() string   { return "payment.refunded" }
func (e PaymentRefunded) AggregateID() uint64 { return e.PaymentID }
