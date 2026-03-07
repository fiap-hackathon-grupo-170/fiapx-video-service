package port

type Claims struct {
	UserID    string
	UserEmail string
}

type TokenValidator interface {
	Validate(tokenString string) (*Claims, error)
}
