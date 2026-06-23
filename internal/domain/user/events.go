package user

type UserRegistered struct {
	UserID uint64
	Email  string
}

func (e UserRegistered) EventName() string {
	return "user.registered"
}

func (e UserRegistered) AggregateID() uint64 {
	return e.UserID
}
