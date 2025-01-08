package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	luhnmod10 "github.com/luhnmod10/go"
	uuid "github.com/satori/go.uuid"

	authenticator "github.com/Alena-Kurushkina/gophermart.git/internal/auth"
	"github.com/Alena-Kurushkina/gophermart.git/internal/config"
	"github.com/Alena-Kurushkina/gophermart.git/internal/gophermart"
	"github.com/Alena-Kurushkina/gophermart.git/internal/gopherror"
	"github.com/Alena-Kurushkina/gophermart.git/internal/helpers"
	"github.com/Alena-Kurushkina/gophermart.git/internal/logger"
	"github.com/Alena-Kurushkina/gophermart.git/internal/model"
	"github.com/Alena-Kurushkina/gophermart.git/internal/worker"
)

type Storager interface {
	AddOrder(ctx context.Context, userID uuid.UUID, number string) error
	GetOrderByNumber(ctx context.Context, number string) (*model.OrderFromDB, error)
	AddUser(ctx context.Context, userID uuid.UUID, login, hashedPassword string) error
	CheckUser(ctx context.Context, login string) (uuid.UUID, string, error)
	GetUserOrders(ctx context.Context, userID uuid.UUID) ([]model.OrderFromDB, error)
	GetUserAccruals(ctx context.Context, userID uuid.UUID) (*model.BalanceFromDB, error)
	WithdrawFunds(ctx context.Context, id uuid.UUID, number string, sum uint32) error
	GetUserWithdrawals(ctx context.Context, id uuid.UUID) ([]model.WithdrawalFromDB, error)
}

type AccrualWorker interface {
	Push(*worker.Task)
}

type Gophermart struct {
	storage Storager
	config  *config.Config
	queue   AccrualWorker
}

func NewGophermart(storage Storager, config *config.Config, queue AccrualWorker) gophermart.Handler {
	ghmart := Gophermart{
		storage: storage,
		config:  config,
		queue:   queue,
	}

	return &ghmart
}

func (gh *Gophermart) UserAuthenticate(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")

	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		// `400` — неверный формат запроса
		http.Error(res, "Invalid content type", http.StatusBadRequest)
		return
	}

	var credentials model.Credentials
	err := json.NewDecoder(req.Body).Decode(&credentials)
	if err != nil {
		// `400` — неверный формат запроса
		http.Error(res, "Can't read body", http.StatusBadRequest)
		return
	}

	if len(credentials.Password) == 0 || len(credentials.Login) == 0 {
		// `400` — неверный формат запроса
		http.Error(res, "Empty password or login", http.StatusBadRequest)
		return
	}

	// получаем пользователя
	userID, savedPasswordHash, err := gh.storage.CheckUser(req.Context(), credentials.Login)
	if err != nil {
		logger.Log.Debug("Error while getting user by login", logger.ErrorMark(err))
		if errors.Is(err, sql.ErrNoRows) {
			// `401` — неверный логин
			http.Error(res, "Incorrect login", http.StatusUnauthorized)
			return
		}
		// `500` — внутренняя ошибка сервера
		http.Error(res, "Can't check user credentials", http.StatusInternalServerError)
		return
	}
	// проверяем пароль
	if !helpers.CompareHashPassword(savedPasswordHash, credentials.Password) {
		// `401` — неверный пароль
		http.Error(res, "Incorrect password", http.StatusUnauthorized)
		return
	}

	// создаём токен аутентификации и добавляем в куки
	err = authenticator.SetNewJWTInCookie(res, userID)
	if err != nil {
		// `500` — внутренняя ошибка сервера
		http.Error(res, "Can't create JWT", http.StatusInternalServerError)
		return
	}

	//`200` — пользователь успешно зарегистрирован и аутентифицирован
	res.WriteHeader(http.StatusOK)
}

func (gh *Gophermart) UserRegister(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")

	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		// `400` — неверный формат запроса
		http.Error(res, "Invalid content type", http.StatusBadRequest)
		return
	}

	var credentials model.Credentials
	err := json.NewDecoder(req.Body).Decode(&credentials)
	if err != nil {
		// `400` — неверный формат запроса
		http.Error(res, "Can't read body", http.StatusBadRequest)
		return
	}

	if len(credentials.Password) == 0 || len(credentials.Login) == 0 {
		// `400` — неверный формат запроса
		http.Error(res, "Empty password or login", http.StatusBadRequest)
		return
	}

	// сокрытие пароля
	hash, err := helpers.HashPassword(credentials.Password)
	if err != nil {
		// `500` — внутренняя ошибка сервера
		http.Error(res, "Can't hash password", http.StatusInternalServerError)
		return
	}

	// генерация id пользователя
	userID := uuid.NewV4()

	// добавляем пользователя в базу
	err = gh.storage.AddUser(req.Context(), userID, credentials.Login, hash)
	if err != nil {
		if errors.Is(err, gopherror.ErrLoginAlreadyExists) {
			// `409` — логин уже занят
			http.Error(res, "Login is already used by another user", http.StatusConflict)
			return
		}
	}

	// создаём токен аутентификации и добавляем в куки
	err = authenticator.SetNewJWTInCookie(res, userID)
	if err != nil {
		// `500` — внутренняя ошибка сервера
		http.Error(res, "Can't create JWT", http.StatusInternalServerError)
		return
	}

	//`200` — пользователь успешно зарегистрирован и аутентифицирован
	res.WriteHeader(http.StatusOK)
}

func (gh *Gophermart) AddOrder(res http.ResponseWriter, req *http.Request) {
	// set response content type
	res.Header().Set("Content-Type", "text/plain")

	// parse request body
	contentType := req.Header.Get("Content-Type")
	if contentType != "text/plain" {
		http.Error(res, "Invalid content type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "Can't read body", http.StatusBadRequest)
		return
	}
	number := string(body)

	logger.Log.Debug("Request", logger.StringMark("body", number))

	if len(number) == 0 {
		http.Error(res, "Body is empty", http.StatusBadRequest)
		return
	}

	if valid := luhnmod10.Valid(number); !valid {
		//`422` — неверный формат номера заказа
		http.Error(res, "Incorrect order number", http.StatusUnprocessableEntity)
	}

	q := req.URL.Query()
	id, err := uuid.FromString(q.Get("userUUID"))
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = gh.storage.AddOrder(req.Context(), id, number)
	if err != nil {
		// если такой номер заказа уже есть в БД
		if errors.Is(err, gopherror.ErrRecordAlreadyExists) {
			// получаем информацию о заказе по номеру
			order, err := gh.storage.GetOrderByNumber(req.Context(), number)
			if err != nil {
				// `500` — внутренняя ошибка сервера
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			if order.UserID != id {
				//`409` — номер заказа уже был загружен другим пользователем
				res.WriteHeader(http.StatusConflict)
				return
			}
			//`200` — номер заказа уже был загружен этим пользователем;
			res.WriteHeader(http.StatusOK)
			return
		} else {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// добавлен новый заказ, отправляем на обработку
	gh.queue.Push(&worker.Task{Number: number, Status: worker.StatusNew})

	// `202` — новый номер заказа принят в обработку
	res.WriteHeader(http.StatusAccepted)
}

func (gh *Gophermart) GetOrders(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	q := req.URL.Query()
	id, err := uuid.FromString(q.Get("userUUID"))
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	orders, err := gh.storage.GetUserOrders(req.Context(), id)
	if err != nil {
		logger.Log.Error("Error while getting user's orders", logger.ErrorMark(err))
		// `500` — внутренняя ошибка сервера
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		// `204` — нет данных для ответа
		res.WriteHeader(http.StatusNoContent)
		return
	}

	ordersToOutput:=make([]*model.Order, len(orders))
	for k, v := range orders {
		ordersToOutput[k]=v.ConvertOutput()
	}

	response, err := json.Marshal(ordersToOutput)
	if err != nil {
		// `500` — внутренняя ошибка сервера
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.Write(response)

	//`200` — успешная обработка запроса
	res.WriteHeader(http.StatusOK)
}

func (gh *Gophermart) GetBalance(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	q := req.URL.Query()
	id, err := uuid.FromString(q.Get("userUUID"))
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	balance, err := gh.storage.GetUserAccruals(req.Context(), id)
	if err != nil {
		// `500` — внутренняя ошибка сервера
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// преобразуем структуру для вывода значений пользователю
	balanceToOutput:=balance.ConvertOutput()

	response, err := json.Marshal(balanceToOutput)
	if err != nil {
		// `500` — внутренняя ошибка сервера
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.Write(response)

	//`200` — успешная обработка запроса
	res.WriteHeader(http.StatusOK)
}

func (gh *Gophermart) WithdrawFunds(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")

	// parse request body
	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(res, "Invalid content type", http.StatusBadRequest)
		return
	}

	var withdrawal model.Withdrawal
	if err:=json.NewDecoder(req.Body).Decode(&withdrawal); err!=nil {
		http.Error(res, "Can't read body", http.StatusBadRequest)
		return
	}

	if valid := luhnmod10.Valid(withdrawal.OrderNumber); !valid {
		//`422` — неверный номер заказа
		http.Error(res, "Incorrect order number", http.StatusUnprocessableEntity)
	}

	withdrawalInner:=withdrawal.ConvertInput()

	q := req.URL.Query()
	id, err := uuid.FromString(q.Get("userUUID"))
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	balance, err := gh.storage.GetUserAccruals(req.Context(), id)
	if err != nil {
		// `500` — внутренняя ошибка сервера
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if balance.Accruals - balance.Withdrawals < withdrawalInner.Sum{
		// `402` — на счету недостаточно средств
		http.Error(res, "Not enough funds", http.StatusPaymentRequired)
		return
	}

	err=gh.storage.WithdrawFunds(req.Context(), id, withdrawalInner.OrderNumber, withdrawalInner.Sum)
	if err!=nil{
		// `500` — внутренняя ошибка сервера
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//`200` — успешная обработка запроса
	res.WriteHeader(http.StatusOK)
}

func (gh *Gophermart) GetWithdrawals(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	q := req.URL.Query()
	id, err := uuid.FromString(q.Get("userUUID"))
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	withdrawals, err := gh.storage.GetUserWithdrawals(req.Context(), id)
	if err != nil {
		// `500` — внутренняя ошибка сервера
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		// `204` — нет ни одного списания
		res.WriteHeader(http.StatusNoContent)
		return
	}

	withdrawalsOutput:=make([]model.Withdrawal, len(withdrawals))
	for k, v := range withdrawals {
		withdrawalsOutput[k] = *v.ConvertOutput()
	}

	response, err := json.Marshal(withdrawalsOutput)
	if err != nil {
		// `500` — внутренняя ошибка сервера
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.Write(response)

	//`200` — успешная обработка запроса
	res.WriteHeader(http.StatusOK)
}
