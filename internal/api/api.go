package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	luhnmod10 "github.com/luhnmod10/go"
	uuid "github.com/satori/go.uuid"

	"github.com/Alena-Kurushkina/gophermart.git/internal/config"
	gopherror "github.com/Alena-Kurushkina/gophermart.git/internal/errors"
	"github.com/Alena-Kurushkina/gophermart.git/internal/gophermart"
	"github.com/Alena-Kurushkina/gophermart.git/internal/logger"
	"github.com/Alena-Kurushkina/gophermart.git/internal/worker"
)

type Storager interface {
	AddOrder(ctx context.Context, user_id uuid.UUID, number string) error
	GetOrderByNumber(ctx context.Context, number string) (*Order,error)
	//UpdateOrderStatus(ctx context.Context, number, status string, accrual uint32) error
}

type AccrualWorker interface {
	Push(*worker.Task)
}

// TODO: delete
var UserID=uuid.FromStringOrNil("00008acd-6bb7-4d27-a224-233c4b22fc02")

type Order struct {
	Number string
	UserID uuid.UUID
	UploadedAt time.Time
	Status string
	Accrual uint32
}

type Gophermart struct {
	storage Storager
	config *config.Config
	queue AccrualWorker
}

func NewGophermart(storage Storager, config *config.Config, queue AccrualWorker) gophermart.Handler {
	ghmart:=Gophermart{
		storage: storage,
		config: config,
		queue: queue,
	}

	return &ghmart
}

func (gh *Gophermart) UserRegister(res http.ResponseWriter, req *http.Request) {

}

func (gh *Gophermart) UserAuthenticate(res http.ResponseWriter, req *http.Request) {
	
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

	if valid:=luhnmod10.Valid(number); !valid {
		//`422` — неверный формат номера заказа
		http.Error(res, "Incorrect order number", http.StatusUnprocessableEntity)
	}

	// q := req.URL.Query()
	// id, err := uuid.FromString(q.Get("userUUID"))
	// if err != nil {
	// 	http.Error(res, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	id:=UserID

	err = gh.storage.AddOrder(req.Context(), id, number)
	if err!=nil{
		// если такой номер заказа уже есть в БД
		if errors.Is(err, gopherror.ErrRecordAlreadyExists){
			// получаем информацию о заказе по номеру
			order,err:=gh.storage.GetOrderByNumber(req.Context(), number)
			if err!=nil{
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			if order.UserID!=id{
				//`409` — номер заказа уже был загружен другим пользователем
				res.WriteHeader(http.StatusConflict)
				return
			}
			//`200` — номер заказа уже был загружен этим пользователем;
			res.WriteHeader(http.StatusOK)
			return
		}else{
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
	
}

func (gh *Gophermart) GetBalance(res http.ResponseWriter, req *http.Request) {
	
}

func (gh *Gophermart) WithdrawFunds(res http.ResponseWriter, req *http.Request) {
	
}

func (gh *Gophermart) GetWithdrawals(res http.ResponseWriter, req *http.Request) {
	
}