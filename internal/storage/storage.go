package storage

import (
	"context"
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	uuid "github.com/satori/go.uuid"

	"github.com/Alena-Kurushkina/gophermart.git/internal/gopherror"
	"github.com/Alena-Kurushkina/gophermart.git/internal/logger"
	"github.com/Alena-Kurushkina/gophermart.git/internal/model"
)

const UniqueConstraintViolationErrorCode string= "23505"

type DBStorage struct {
	database *sql.DB
}

func NewDBStorage (connectionStr string) (*DBStorage, error){
	db, err:=sql.Open("pgx", connectionStr)
	if err!=nil{
		return nil, err
	}
	logger.Log.Debug("DB connection opened")

	//TODO: use makefile
	
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err!=nil{
		return nil, err
	}
    m, err := migrate.NewWithDatabaseInstance(
        "file:internal/storage/migrations",
        "gophermart", driver,
	)
	if err!=nil{
		return nil, err
	}
    err=m.Up()
	if err!=nil{
		return nil, err
	}

	return &DBStorage{database: db}, nil
}

func (d DBStorage) AddOrder(ctx context.Context, userID uuid.UUID, number string) error {
	result, err:= d.database.ExecContext(ctx,
		`INSERT INTO orders (user_id, number, status_processing) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (number) 
			DO NOTHING;`,
		userID,
		number,
		"NEW",
	)
	if err!=nil{
		return err
	}
	affected, err:=result.RowsAffected()
	if err!=nil{
		return err
	}
	if affected==0 {
		logger.Log.Info("Attempt to add order with number that already exists")
		return gopherror.ErrRecordAlreadyExists
	}

	return nil
}

func (d DBStorage) GetOrderByNumber(ctx context.Context, number string) (*model.OrderFromDB, error){
	row:=d.database.QueryRowContext(ctx, 
		`SELECT number, user_id, uploaded_at, status_processing, accrual 
		FROM orders
		WHERE number = $1`,
		number,
	)
	var order model.OrderFromDB
	err:=row.Scan(&order.Number, &order.UserID, &order.UploadedAt, &order.Status, &order.Accrual)
	if err != nil {
		return nil,err
	}
	return &order, nil
}

func (d DBStorage) UpdateOrderStatus(ctx context.Context, number, status string) error {
	_, err:= d.database.ExecContext(ctx,
		`UPDATE orders 
		SET status_processing = $1 
		WHERE number = $2;`,
		status,
		number,
	)
	if err!=nil{
		return err
	}

	return nil
}

func (d DBStorage) UpdateOrderStatusAndAccrual(ctx context.Context, number, status string, accrual uint32) error {
	_, err:= d.database.ExecContext(ctx,
		`UPDATE orders 
		SET status_processing = $1, accrual = $2 
		WHERE number = $3;`,
		status,
		accrual,
		number,
	)
	if err!=nil{
		return err
	}

	return nil 
}

func (d DBStorage) AddUser(ctx context.Context, userID uuid.UUID, login, hashedPassword string) error {
	_, err:= d.database.ExecContext(ctx,
		`INSERT INTO users (id, login, password) 
		VALUES ($1, $2, $3);`,
		userID,
		login,
		hashedPassword,
	)
	if err!=nil{
		if pqerr, ok:=err.(*pgconn.PgError); ok{
			if pqerr.Code==UniqueConstraintViolationErrorCode{
				return gopherror.ErrLoginAlreadyExists
			}
		}
		return err
	}
	return nil
}

func (d DBStorage) CheckUser(ctx context.Context, login string) (uuid.UUID, string, error) {
	row:= d.database.QueryRowContext(ctx,
		`SELECT id, password FROM users
		WHERE login = $1;`,
		login,
	)
	var id uuid.UUID
	var password string
	err:=row.Scan(&id, &password)
	if err != nil {
		return uuid.Nil, "", err
	}
	return id, password, nil
}

func (d DBStorage) GetUserOrders(ctx context.Context, id uuid.UUID) ([]model.OrderFromDB, error){
	rows, err := d.database.QueryContext(ctx,
		`SELECT number, user_id, uploaded_at, status_processing, accrual
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC`,
		id.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]model.OrderFromDB, 0, 10)

	// пробегаем по всем записям
	for rows.Next() {
		var v model.OrderFromDB
		err = rows.Scan(&v.Number, &v.UserID, &v.UploadedAt, &v.Status, &v.Accrual)
		if err != nil {
			return nil, err
		}

		records = append(records, v)
	}

	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (d DBStorage) GetUserAccruals(ctx context.Context, id uuid.UUID) (*model.BalanceFromDB, error){
	rows, err := d.database.QueryContext(ctx,
		`SELECT coalesce(sum(accrual), 0)  from orders where user_id=$1
		UNION ALL
		SELECT coalesce(sum(sum), 0) from withdraws where user_id=$1
		`,
		id.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balance model.BalanceFromDB

	// считываем значение суммы начислений
	if rows.Next(){
		err=rows.Scan(&balance.Accruals)
		if err != nil {
			return nil, err
		}
	}else{
		return nil, rows.Err()
	}

	// считываем значение суммы списаний
	if rows.Next(){
		err = rows.Scan(&balance.Withdrawals)
		if err != nil {
			return nil, err
		}
	}else{
		return nil, rows.Err()
	}

	return &balance, nil
}

func (d DBStorage) WithdrawFunds(ctx context.Context, id uuid.UUID, number string, sum uint32) error {
	_, err:= d.database.ExecContext(ctx,
		`INSERT INTO withdraws (withdraw_number, sum, user_id) 
		VALUES ($1, $2, $3);`,
		number,
		sum,
		id,
	)
	if err!=nil{
		return err
	}

	return nil
}

func (d DBStorage) GetUserWithdrawals(ctx context.Context, id uuid.UUID) ([]model.WithdrawalFromDB, error){
	rows, err := d.database.QueryContext(ctx,
		`SELECT withdraw_number, sum, processed_at
		FROM withdraws
		WHERE user_id = $1
		ORDER BY processed_at DESC`,
		id.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]model.WithdrawalFromDB, 0, 10)

	// пробегаем по всем записям
	for rows.Next() {
		var v model.WithdrawalFromDB
		err = rows.Scan(&v.OrderNumber, &v.Sum, &v.ProcessedAt)
		if err != nil {
			return nil, err
		}

		records = append(records, v)
	}

	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return records, nil
}