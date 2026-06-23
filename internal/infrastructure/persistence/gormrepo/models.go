package gormrepo

import "time"

type UserModel struct {
	ID           uint64 `gorm:"primaryKey"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	FullName     string
	Role         string `gorm:"type:varchar(20);index;not null;default:customer"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (UserModel) TableName() string { return "users" }

type ProductModel struct {
	ID          uint64 `gorm:"primaryKey"`
	SKU         string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"index;not null"`
	Description string
	PriceAmount int64  `gorm:"not null"`
	Currency    string `gorm:"type:varchar(3);not null;default:USD"`
	Active      bool   `gorm:"index;not null;default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (ProductModel) TableName() string { return "products" }

type InventoryModel struct {
	ID           uint64 `gorm:"primaryKey"`
	ProductID    uint64 `gorm:"uniqueIndex;not null"`
	Available    int    `gorm:"not null;default:0"`
	Reserved     int    `gorm:"not null;default:0"`
	ReorderLevel int    `gorm:"not null;default:0"`
	Version      int    `gorm:"not null;default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (InventoryModel) TableName() string { return "inventories" }

type OrderModel struct {
	ID             uint64 `gorm:"primaryKey"`
	UserID         uint64 `gorm:"not null;index:idx_orders_user_status,priority:1"`
	Status         string `gorm:"type:varchar(20);not null;index;index:idx_orders_user_status,priority:2"`
	TotalAmount    int64  `gorm:"not null"`
	Currency       string `gorm:"type:varchar(3);not null;default:USD"`
	IdempotencyKey string `gorm:"uniqueIndex;not null"`
	FailureReason  string
	Items          []OrderItemModel `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
	CreatedAt      time.Time        `gorm:"index"`
	UpdatedAt      time.Time
}

func (OrderModel) TableName() string { return "orders" }

type OrderItemModel struct {
	ID              uint64 `gorm:"primaryKey"`
	OrderID         uint64 `gorm:"not null;index"`
	ProductID       uint64 `gorm:"not null;index"`
	SKU             string
	Name            string
	UnitPriceAmount int64  `gorm:"not null"`
	Currency        string `gorm:"type:varchar(3);not null;default:USD"`
	Quantity        int    `gorm:"not null"`
}

func (OrderItemModel) TableName() string { return "order_items" }

type PaymentModel struct {
	ID             uint64 `gorm:"primaryKey"`
	OrderID        uint64 `gorm:"uniqueIndex;not null"`
	IdempotencyKey string `gorm:"uniqueIndex;not null"`
	Amount         int64  `gorm:"not null"`
	Currency       string `gorm:"type:varchar(3);not null;default:USD"`
	Status         string `gorm:"type:varchar(20);not null;index"`
	Provider       string
	ProviderRef    string
	Attempts       int `gorm:"not null;default:0"`
	FailureReason  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (PaymentModel) TableName() string { return "payments" }

type NotificationModel struct {
	ID            uint64  `gorm:"primaryKey"`
	UserID        *uint64 `gorm:"index"`
	Type          string  `gorm:"type:varchar(40);not null;index"`
	Channel       string  `gorm:"type:varchar(20);not null"`
	Status        string  `gorm:"type:varchar(20);not null;index"`
	Subject       string
	Payload       string `gorm:"type:text"`
	Attempts      int    `gorm:"not null;default:0"`
	FailureReason string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (NotificationModel) TableName() string { return "notifications" }

type AuditLogModel struct {
	ID        uint64    `gorm:"primaryKey"`
	ActorID   *uint64   `gorm:"index"`
	Action    string    `gorm:"not null;index"`
	Entity    string    `gorm:"not null;index"`
	EntityID  uint64    `gorm:"index"`
	Before    string    `gorm:"type:text"`
	After     string    `gorm:"type:text"`
	Metadata  string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"index"`
}

func (AuditLogModel) TableName() string { return "audit_logs" }

func AllModels() []any {
	return []any{
		&UserModel{},
		&ProductModel{},
		&InventoryModel{},
		&OrderModel{},
		&OrderItemModel{},
		&PaymentModel{},
		&NotificationModel{},
		&AuditLogModel{},
	}
}
