package model

import (
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/Alena-Kurushkina/gophermart.git/internal/helpers"
)

type Credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Balance struct {
	CurrentBalance float32 `json:"current"`
	Withdrawals float32 `json:"withdrawn"`
}

type BalanceFromDB struct {
	Balance
	Accruals uint32
	Withdrawals uint32
}

// ConvertOutput преобразует структуру BalanceFromDB в Balance,
// а именно вычисляет баланс как разность начислений и списаний, 
// переводить внутренние типы сервиса в пользовательские типы
func (c *BalanceFromDB) ConvertOutput() *Balance {
	BalanceToOutput := Balance{
		CurrentBalance: helpers.BaseToAccrual(c.Accruals-c.Withdrawals),
		Withdrawals: helpers.BaseToAccrual(c.Withdrawals),
	}
	return &BalanceToOutput
}

type Order struct {
	Number        string    `json:"number"`
	UploadedAt    time.Time `json:"uploaded_at"`
	Status        string    `json:"status"`
	Accrual float32   `json:"accrual,omitempty"`
}

type OrderFromDB struct {
	Order
	UserID        uuid.UUID    
	Accrual       uint32    
}

// ConvertOutput преобразует структуру OrderFromDB в Order,
// а именно переводить внутренние типы сервиса в пользовательские типы
func (c *OrderFromDB) ConvertOutput() *Order {
	OrderToOutput := Order{
		Number: c.Number,
		UploadedAt: c.UploadedAt,
		Status: c.Status,
		Accrual: helpers.BaseToAccrual(c.Accrual),
	}
	return &OrderToOutput
}

type Withdrawal struct {
	OrderNumber string `json:"order"`
	Sum float32 `json:"sum"`
	ProcessedAt time.Time `json:"processed_at,omitempty"`
}

type WithdrawalFromDB struct {
	Withdrawal
	Sum uint32
}

// ConvertOutput преобразует структуру OrderFromDB в Order,
// а именно переводить внутренние типы сервиса в пользовательские типы
func (c *WithdrawalFromDB) ConvertOutput() *Withdrawal {
	WithdrawalToOutput := Withdrawal{
		OrderNumber: c.OrderNumber,
		ProcessedAt: c.ProcessedAt,
		Sum: helpers.BaseToAccrual(c.Sum),
	}
	return &WithdrawalToOutput
}

// ConvertOutput преобразует структуру Withdrawal в WithdrawalFromDB,
// а именно переводит пользовательские типы во внутренние типы сервиса
func (c *Withdrawal) ConvertInput() *WithdrawalFromDB {
	WithdrawalFromDB:= WithdrawalFromDB{
		Withdrawal: *c,
		Sum: helpers.AccrualToBase(c.Sum),
	}
	return &WithdrawalFromDB
}