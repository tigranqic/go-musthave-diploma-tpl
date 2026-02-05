package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"go.uber.org/zap"
)

func setupStorage(t *testing.T) (*repository.PostgresStorage, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	storage := repository.NewPostgresStorage(db, zap.NewNop())
	return storage, mock, func() { _ = db.Close() }
}

func TestCreateUser_Success(t *testing.T) {
	storage, mock, closeDB := setupStorage(t)
	defer closeDB()

	login := "testuser"
	pass := []byte("hash")

	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery("INSERT INTO users").WithArgs(login, pass).WillReturnRows(rows)

	id, err := storage.CreateUser(context.Background(), login, pass)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), id)
}

func TestCreateUser_LoginTaken(t *testing.T) {
	storage, mock, closeDB := setupStorage(t)
	defer closeDB()

	login := "testuser"
	pass := []byte("hash")

	mock.ExpectQuery("INSERT INTO users").WillReturnError(errors.New("login already taken"))

	_, err := storage.CreateUser(context.Background(), login, pass)
	assert.Equal(t, repository.ErrLoginTaken, err)
}

func TestGetUserByLogin_Success(t *testing.T) {
	storage, mock, closeDB := setupStorage(t)
	defer closeDB()

	user := &model.User{ID: 1, Login: "u", PasswordHash: []byte("p")}
	rows := sqlmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
		AddRow(user.ID, user.Login, user.PasswordHash, time.Now())
	mock.ExpectQuery("SELECT id, login").WithArgs("u").WillReturnRows(rows)

	got, err := storage.GetUserByLogin(context.Background(), "u")
	assert.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
}

func TestGetUserByLogin_NotFound(t *testing.T) {
	storage, mock, closeDB := setupStorage(t)
	defer closeDB()

	mock.ExpectQuery("SELECT id, login").WithArgs("unknown").WillReturnError(sql.ErrNoRows)

	user, err := storage.GetUserByLogin(context.Background(), "unknown")
	assert.Nil(t, user)
	assert.Equal(t, repository.ErrUserNotFound, err)
}

func TestCreateOrder_Success(t *testing.T) {
	storage, mock, closeDB := setupStorage(t)
	defer closeDB()

	order := &model.Order{Number: "123", UserID: 1, Status: "NEW"}

	mock.ExpectExec("INSERT INTO orders").WithArgs(order.Number, order.UserID, order.Status).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := storage.CreateOrder(context.Background(), order)
	assert.NoError(t, err)
}

func TestCreateOrder_ExistsSameUser(t *testing.T) {
	storage, mock, closeDB := setupStorage(t)
	defer closeDB()

	order := &model.Order{Number: "123", UserID: 1, Status: "NEW"}

	mock.ExpectExec("INSERT INTO orders").WillReturnError(errors.New("order exists"))

	rows := sqlmock.NewRows([]string{"user_id"}).AddRow(1)
	mock.ExpectQuery("SELECT user_id FROM orders").WithArgs(order.Number).WillReturnRows(rows)

	err := storage.CreateOrder(context.Background(), order)
	assert.Equal(t, repository.ErrOrderExists, err)
}

func TestCreateOrder_ExistsOtherUser(t *testing.T) {
	storage, mock, closeDB := setupStorage(t)
	defer closeDB()

	order := &model.Order{Number: "123", UserID: 1, Status: "NEW"}

	mock.ExpectExec("INSERT INTO orders").WillReturnError(errors.New("order owned by another user"))

	rows := sqlmock.NewRows([]string{"user_id"}).AddRow(2)
	mock.ExpectQuery("SELECT user_id FROM orders").WithArgs(order.Number).WillReturnRows(rows)

	err := storage.CreateOrder(context.Background(), order)
	assert.Equal(t, repository.ErrOrderOwnedByOther, err)
}

func TestGetOrder_Success(t *testing.T) {
	storage, mock, closeDB := setupStorage(t)
	defer closeDB()

	o := &model.Order{Number: "123", UserID: 1, Status: "NEW", Accrual: nil}
	rows := sqlmock.NewRows([]string{"number", "user_id", "status", "accrual", "uploaded_at"}).
		AddRow(o.Number, o.UserID, o.Status, o.Accrual, time.Now())
	mock.ExpectQuery("SELECT number, user_id").WithArgs("123").WillReturnRows(rows)

	got, err := storage.GetOrder(context.Background(), "123")
	assert.NoError(t, err)
	assert.Equal(t, o.Number, got.Number)
}

func TestGetOrder_NotFound(t *testing.T) {
	storage, mock, closeDB := setupStorage(t)
	defer closeDB()

	mock.ExpectQuery("SELECT number, user_id").WithArgs("123").WillReturnError(sql.ErrNoRows)

	got, err := storage.GetOrder(context.Background(), "123")
	assert.Nil(t, got)
	assert.Equal(t, repository.ErrOrderNotFound, err)
}

func TestUpdateOrderStatus_Success(t *testing.T) {
	storage, mock, closeDB := setupStorage(t)
	defer closeDB()

	order := &model.Order{Number: "123", Status: "PROCESSED", Accrual: nil}
	mock.ExpectExec("UPDATE orders").WithArgs(order.Status, order.Accrual, order.Number).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := storage.UpdateOrderStatus(context.Background(), order)
	assert.NoError(t, err)
}

func TestUpdateOrderStatus_NotFound(t *testing.T) {
	storage, mock, closeDB := setupStorage(t)
	defer closeDB()

	order := &model.Order{Number: "123", Status: "PROCESSED", Accrual: nil}
	mock.ExpectExec("UPDATE orders").WithArgs(order.Status, order.Accrual, order.Number).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := storage.UpdateOrderStatus(context.Background(), order)
	assert.Equal(t, repository.ErrOrderNotFound, err)
}
