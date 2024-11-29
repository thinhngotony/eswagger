package model

type RequestStruct struct {
	ID        int    `json:"id" validate:"required" doc:"Unique identifier for the user" example:"1"`
	Username  string `json:"username" doc:"Username for login" example:"john_doe"`
	Email     string `json:"email" doc:"User's email address" example:"john@example.com"`
	FirstName string `json:"first_name" doc:"First name of the user" example:"John"`
	LastName  string `json:"last_name" doc:"Last name of the user" example:"Doe"`
}

type UpdateUserRequest struct {
	Username string `json:"update_username,omitempty" doc:"Update the username of the user"`
	Email    string `json:"update_email,omitempty" doc:"Update the email of the user"`
}
