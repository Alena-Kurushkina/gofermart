package worker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"slices"

	"github.com/Alena-Kurushkina/gophermart.git/internal/helpers"
	"github.com/Alena-Kurushkina/gophermart.git/internal/logger"
)

type Status string
const (
	StatusNew="NEW"
	StatusRegistered = "REGISTERED"
	StatusInvalid = "INVALID"
	StatusProcessing = "PROCESSING" 
	StatusProcessed = "PROCESSED"
)

type Task struct {
    Number string
	Status Status
}

type Queue chan *Task

func newQueue() Queue {
	q:=make(chan *Task, 20)	
    return q
}

func (q Queue) Push(t *Task) {
    // добавляем задачу в очередь
    q <- t
}

func (q Queue) Pop() *Task {
    // получаем задачу
    return <-q
}

type AccrualStorager interface {
	UpdateOrderStatus(ctx context.Context, number, status string) error
	UpdateOrderStatusAndAccrual(ctx context.Context, number, status string, accrual uint32) error
}

type updater struct {
	db AccrualStorager
	client *http.Client
	accrualAddress string
}

func newUpdater(db AccrualStorager, address string) *updater {
	cl:= http.Client{}
    return &updater{
       db: db,
	   client: &cl,
	   accrualAddress: address,
    }
}

type AccrualResponse struct {
	Order string `json:"order"`
	Status string `json:"status"`
	Accrual float32 `json:"accrual,omitempty"`
}

func (u *updater) updateStatus(number string) (Status, uint32, error) {
	logger.Log.Debug("Sending request to accrual service to update order status")
	req,err:=http.NewRequest(http.MethodGet, u.accrualAddress+"/api/orders/"+number, nil)
	if err!=nil{
		return StatusNew, 0, err
	}
	resp, err:=u.client.Do(req)
	if err!=nil{
		return StatusNew, 0, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode{
	case 500:
		logger.Log.Error("Accrual service has returned Internal server error")
	case 429:
		logger.Log.Error("The number of requests to the accrual service has been exceeded")
		// TODO: как уменьшить количество запросов
	case 204:
		logger.Log.Info("Order "+number+" are not registered in calculation system")
	case 200:
		logger.Log.Debug("Accrual service returned successfull response")
		body,err:=io.ReadAll(resp.Body)
		if err!=nil{
			return StatusNew, 0, err
		}
		rr:=AccrualResponse{}
		err = json.Unmarshal(body, &rr)
		if err != nil {
			return StatusNew, 0, err
		}
		var accrual uint32
		logger.Log.Info("Accrual service returned accrual", logger.Float32Mark("amount", rr.Accrual))
		if rr.Accrual!=0{
			accrual=helpers.AccrualToBase(rr.Accrual)
		}
		return Status(rr.Status), accrual, nil		
	}

	return StatusNew, 0, nil
}

func (u *updater) saveStatus(ctx context.Context, number string, status Status) error {
	logger.Log.Debug("Updating order status in DB")

	err:=u.db.UpdateOrderStatus(ctx, number, string(status))
	if err!=nil{
		return err
	}
    return nil
}

func (u *updater) saveStatusAndAccrual(ctx context.Context, number string, status Status, accrual uint32) error {
	logger.Log.Debug("Updating order status and accrual in DB")

	err:=u.db.UpdateOrderStatusAndAccrual(ctx, number, string(status), accrual)
	if err!=nil{
		return err
	}
    return nil
}

type Worker struct {
	doneCtx context.Context
    id      int
    queue   Queue
    updater *updater
}

func newWorker(ctx context.Context, id int, queue Queue, updater *updater) *Worker {
    w := Worker{
		doneCtx: ctx,
        id:      id,
        queue:   queue,
        updater: updater,
    }
    return &w
}

func RunWorkers(ctx context.Context, db AccrualStorager, address string) Queue {
	queue := newQueue()

    for i := 0; i < runtime.NumCPU(); i++ {
        w := newWorker(ctx, i, queue, newUpdater(db, address))
        go w.loop()
    }

	return queue
}

func (w *Worker) loop() {
	tempStatuses := []string{StatusRegistered, StatusProcessing, StatusNew}
    for {
		// TODO: как и когда закрывать канал с заказами

		// берём заказ из очереди
		t := w.queue.Pop()
		// запрашиваем статус из службы accrual
		accrualStatus, accrual, err := w.updater.updateStatus(t.Number)
		if err != nil {
			logger.Log.Error("Cant update order status from accrual service",
				logger.ErrorMark(err),
			)
			// кладём обратно в очередь
			w.queue.Push(t)
			continue
		}
		
		// если статус обновился, обновляем данные в БД
		if string(accrualStatus)!=string(t.Status){
			logger.Log.Debug("Worker has updated the status of order",
				logger.IntMark("Worker id", w.id),
				logger.StringMark("Order number", t.Number),
				logger.StringMark("New status", string(accrualStatus)),
				logger.StringMark("Old status", string(t.Status)),
				logger.Uint32Mark("Accrual", accrual),
			)
			if accrualStatus == StatusProcessed {
				w.updater.saveStatusAndAccrual(w.doneCtx, t.Number, accrualStatus, accrual)
			} else {
				w.updater.saveStatus(w.doneCtx,t.Number, accrualStatus)
			}
		}
		// если статус не окончательный, кладём обратно в очередь
		if slices.Contains(tempStatuses, string(accrualStatus)){
			w.queue.Push(t)
		}    
    }
}