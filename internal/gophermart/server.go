package gophermart

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Alena-Kurushkina/gophermart.git/internal/config"
	"github.com/Alena-Kurushkina/gophermart.git/internal/logger"
)

type Handler interface {
	UserRegister(res http.ResponseWriter, req *http.Request)
	UserAuthenticate(res http.ResponseWriter, req *http.Request)
	AddOrder(res http.ResponseWriter, req *http.Request)
	GetOrders(res http.ResponseWriter, req *http.Request)
	GetBalance(res http.ResponseWriter, req *http.Request)
	WithdrawFunds(res http.ResponseWriter, req *http.Request)
	GetWithdrawals(res http.ResponseWriter, req *http.Request)
}

type Server struct {
	Handler chi.Router
	Config *config.Config
}

func NewServer(hdl Handler, cfg *config.Config) *Server {
	return &Server{
		Handler: newRouter(hdl),
		Config: cfg,
	}
}

func (s *Server) Run() {
	logger.Log.Info("Server is listening on "+s.Config.ServerAddress)
	err:=http.ListenAndServe(s.Config.ServerAddress, s.Handler)
	if err!=nil{
		panic(err)
	}
}

func newRouter(hdl Handler) chi.Router{
	r:=chi.NewRouter()

	r.Use(logger.LogMiddleware)
	r.Post("/api/user/register", hdl.UserRegister)
	r.Post("/api/user/login", hdl.UserAuthenticate)

	r.Group(func(r chi.Router){
		r.Post("/api/user/orders", hdl.AddOrder)
		r.Get("/api/user/orders", hdl.GetOrders)
		r.Get("/api/user/balance", hdl.GetBalance)
		r.Post("/api/user/balance/withdraw", hdl.WithdrawFunds)
		r.Get("/api/user/withdrawals", hdl.GetWithdrawals)
	})

	return r
}