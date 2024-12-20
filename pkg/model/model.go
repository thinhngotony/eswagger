package model

import "time"

type UserInterface interface {
	// NOTE If API not appears in here, it cannot generate model, it just can show endpoint

	CreateUserPointerSliceToPointerResponse(*CreateUserStruct) (*UserResponse, error)
	NotWork_CreateUserSliceToPointerResponse([]CreateUserStruct) (*UserResponse, error)
	CreateUserStructToPointerResponse(CreateUserStruct) (*UserResponse, error)
	CreateUserPointerSliceToSliceResponse(*[]CreateUserStruct) (*[]UserResponse, error)
	CreateUserPointerSliceToNonPointerSliceResponse(*[]CreateUserStruct) ([]UserResponse, error)
	CreateUserPointerSliceToNonPointerResponse(*[]CreateUserStruct) (UserResponse, error)
	CreateUserSliceToSliceResponse([]CreateUserStruct) ([]UserResponse, error)
	CreateUserStructToSliceResponse(CreateUserStruct) ([]UserResponse, error)
	CreateUserStructToNonPointerResponse(CreateUserStruct) (UserResponse, error)

	// For another type
	UpdateUser(input UpdateUserRequest) (UserResponse, error)
	DeleteUser(id int) error
}

type CreateUserStruct struct {
	// ID        int    `json:"id" validate:"required" doc:"Unique identifier for the user" example:"1"`
	// Username  string `json:"username" doc:"Username for login" example:"john_doe"`
	// Email     string `json:"email" doc:"User's email address" example:"john@example.com"`
	// FirstName string `json:"first_name" doc:"First name of the user" example:"John"`
	// LastName  string `json:"last_name" doc:"Last name of the user" example:"Doe"`
	UpdateUserRequest
	//TODO: These case not work, same for response
	//UpdateUserRequest []UpdateUserRequest
	// UpdateUserRequest UpdateUserRequest
}

type UpdateUserRequest struct {
	Username string `json:"update_username,omitempty" doc:"Update the username of the user" example:"johnny_bravo"`
	Email    string `json:"update_email,omitempty" doc:"Update the email of the user" example:"johnny@example.com"`
}

type UserResponse struct {
	Info
}
type Info struct {
	ID         int       `json:"id" doc:"Unique identifier for the user" example:"1"`
	Username   string    `json:"username" doc:"Username for login" example:"john_doe"`
	Email      string    `json:"email" doc:"User's email address" example:"john@example.com"`
	IsResponse bool      `json:"is_response" doc:"bool value to indicate if this is a response" example:"false"`
	CreatedAt  time.Time `json:"created_at" doc:"Timestamp of user creation" example:"2024-01-01T00:00:00Z"`
}

type CreateUserRequest struct {
	Username string `json:"username" example:"john_doe" doc:"description=Desired username for new account"`
	Email    string `json:"email" example:"john@example.com" doc:"description=Email address for notifications"`
}
