package controllers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/healtronlabs/gofasta/app/dtos"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
	"github.com/healtronlabs/gofasta/pkg/httputil"
	"github.com/healtronlabs/gofasta/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupController() (*controllers.UserController, *mocks.MockUserService) {
	mockSvc := new(mocks.MockUserService)
	uc := controllers.NewUserControllerInstance(mockSvc)
	return uc, mockSvc
}

func TestCreateUser_Handler_Success(t *testing.T) {
	uc, mockSvc := setupController()

	testUser := &dtos.User{
		ID:        uuid.New(),
		FirstName: "John",
		Email:     "john@example.com",
	}
	mockSvc.On("CreateUser", mock.Anything, mock.AnythingOfType("dtos.TCreateUserDto")).
		Return(&dtos.TUserResponseDto{Data: testUser}, nil)

	body, _ := json.Marshal(dtos.TCreateUserDto{
		FirstName:   "John",
		OtherNames:  "Doe",
		Email:       "john@example.com",
		PhoneNumber: "1234567890",
	})

	req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler := httputil.Handle(uc.CreateUser)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	mockSvc.AssertExpectations(t)
}

func TestCreateUser_Handler_BadPayload(t *testing.T) {
	uc, _ := setupController()

	req := httptest.NewRequest("POST", "/users", bytes.NewReader([]byte("invalid json")))
	rec := httptest.NewRecorder()

	handler := httputil.Handle(uc.CreateUser)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetUser_Handler_Success(t *testing.T) {
	uc, mockSvc := setupController()

	testID := uuid.New()
	testUser := &dtos.User{
		ID:        testID,
		FirstName: "Jane",
		Email:     "jane@example.com",
	}
	mockSvc.On("FindUserByID", mock.Anything, mock.AnythingOfType("dtos.TFindUserByIDDto")).
		Return(&dtos.TUserResponseDto{Data: testUser}, nil)

	req := httptest.NewRequest("GET", "/users/"+testID.String(), nil)
	rec := httptest.NewRecorder()

	// Need to set mux vars
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})

	handler := httputil.Handle(uc.GetUser)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockSvc.AssertExpectations(t)
}

func TestGetUser_Handler_InvalidUUID(t *testing.T) {
	uc, _ := setupController()

	req := httptest.NewRequest("GET", "/users/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	req = mux.SetURLVars(req, map[string]string{"id": "not-a-uuid"})

	handler := httputil.Handle(uc.GetUser)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListUsers_Handler_Success(t *testing.T) {
	uc, mockSvc := setupController()

	totalRecords := 0
	page := 1
	limit := 10
	totalPages := 0
	mockSvc.On("FindUsersWithFilters", mock.Anything, mock.AnythingOfType("dtos.UserFiltersDto")).
		Return(&dtos.TUsersResponseDto{
			Data: []*dtos.User{},
			Pagination: &dtos.TPaginationObjectDto{
				TotalRecords:   &totalRecords,
				CurrentPage:    &page,
				RecordsPerPage: &limit,
				TotalPages:     &totalPages,
			},
		}, nil)

	req := httptest.NewRequest("GET", "/users?sortByField=id", nil)
	rec := httptest.NewRecorder()

	handler := httputil.Handle(uc.ListUsers)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockSvc.AssertExpectations(t)
}
