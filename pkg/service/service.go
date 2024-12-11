package service

import (
	"encoding/json"
	"main/pkg/model"
	"net/http"
)

func CreateUser(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req []model.CreateUserStruct
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data, err := s.CreateUserPointerSliceToPointerResponse(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(data)
	}
}

func CreateUserPointerSliceToPointerResponse(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func NotWork_CreateUserSliceToPointerResponse(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func CreateUserStructToPointerResponse(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func CreateUserPointerSliceToSliceResponse(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func CreateUserPointerSliceToNonPointerSliceResponse(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func CreateUserPointerSliceToNonPointerResponse(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func CreateUserSliceToSliceResponse(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func CreateUserStructToSliceResponse(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func CreateUserStructToNonPointerResponse(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func DeleteUser(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.DeleteUser(1)
		user := model.UserResponse{}
		json.NewEncoder(w).Encode(user)
	}
}

func UpdateUser(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.UpdateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.UpdateUser(req)
		user := model.UserResponse{Info: []model.Info{{ID: 1, Username: req.Username, Email: req.Email}}}
		json.NewEncoder(w).Encode(user)
	}
}
