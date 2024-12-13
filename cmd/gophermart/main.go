package main

import (
	"context"

	"github.com/Alena-Kurushkina/gophermart.git/internal/api"
	"github.com/Alena-Kurushkina/gophermart.git/internal/config"
	"github.com/Alena-Kurushkina/gophermart.git/internal/gophermart"
	"github.com/Alena-Kurushkina/gophermart.git/internal/logger"
	"github.com/Alena-Kurushkina/gophermart.git/internal/storage"
	"go.uber.org/zap"
)

func main(){
	ctx:=context.Background()

	config:=config.InitConfig()

	logger.CreateLogger()
	defer logger.Log.Sync()

	logger.Log.Debug("Config parameters: ",
		zap.String("Database URI", config.DatabaseURI),
		zap.String("Server address", config.ServerAddress),
		zap.String("Accrual server address", config.AccrualAddress),
	)

	storage, err:=storage.NewDBStorage(config.DatabaseURI)
	if err!=nil{
		logger.Log.Panic("Error initializing DB",
			zap.Error(err),
		)
	}

	ghmart:=api.NewGophermart(ctx,storage, config)
	server:= gophermart.NewServer(ghmart, config)

	server.Run()
}