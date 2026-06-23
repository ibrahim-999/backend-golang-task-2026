package inventory

type InventoryReserved struct {
	ProductID uint64
	Quantity  int
}

func (e InventoryReserved) EventName() string { return "inventory.reserved" }

func (e InventoryReserved) AggregateID() uint64 { return e.ProductID }

type InventoryReleased struct {
	ProductID uint64
	Quantity  int
}

func (e InventoryReleased) EventName() string { return "inventory.released" }

func (e InventoryReleased) AggregateID() uint64 { return e.ProductID }

type InventoryCommitted struct {
	ProductID uint64
	Quantity  int
}

func (e InventoryCommitted) EventName() string { return "inventory.committed" }

func (e InventoryCommitted) AggregateID() uint64 { return e.ProductID }

type InventoryRestocked struct {
	ProductID uint64
	Quantity  int
}

func (e InventoryRestocked) EventName() string { return "inventory.restocked" }

func (e InventoryRestocked) AggregateID() uint64 { return e.ProductID }
