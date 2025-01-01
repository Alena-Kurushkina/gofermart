package main

import (
	"context"

	"github.com/Alena-Kurushkina/gophermart.git/internal/api"
	"github.com/Alena-Kurushkina/gophermart.git/internal/config"
	"github.com/Alena-Kurushkina/gophermart.git/internal/gophermart"
	"github.com/Alena-Kurushkina/gophermart.git/internal/logger"
	"github.com/Alena-Kurushkina/gophermart.git/internal/storage"
	"github.com/Alena-Kurushkina/gophermart.git/internal/worker"
)

func main(){
	config:=config.InitConfig()

	logger.CreateLogger()
	defer logger.Log.Sync()

	logger.Log.Debug("Config parameters: ",
		logger.StringMark("Database URI", config.DatabaseURI),
		logger.StringMark("Server address", config.ServerAddress),
		logger.StringMark("Accrual server address", config.AccrualAddress),
	)

	storage, err:=storage.NewDBStorage(config.DatabaseURI)
	if err!=nil{
		logger.Log.Panic("Error initializing DB",
		logger.ErrorMark(err),
		)
	}

	queue:=worker.RunWorkers(context.TODO(), storage, config.AccrualAddress)

	ghmart:=api.NewGophermart(storage, config, queue)
	server:= gophermart.NewServer(ghmart, config)

	server.Run()
}