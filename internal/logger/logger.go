package logger

import (
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger = zap.NewNop()

// func Initialize() error {
// 	cfg := zap.NewProductionConfig()
// 	cfg.OutputPaths = []string{
// 		"/Users/alena/log/gophermart.log",
// 		"stdout",
// 	}
// 	zl, err := cfg.Build()

// 	if err != nil {
// 		return err
// 	}

// 	sugar := zl.Sugar()
// 	Log = sugar

// 	return nil
// }

func CreateLogger(){
	stdout := zapcore.AddSync(os.Stdout)

    file := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "/Users/alena/log/gophermart.log",
		MaxSize:    1, //MB
		MaxBackups: 30,
		MaxAge:     90, //days
		Compress:   false,
	})

    level := zap.NewAtomicLevelAt(zap.DebugLevel)

    productionCfg := zap.NewProductionEncoderConfig()
    // productionCfg.TimeKey = "timestamp"
    // productionCfg.EncodeTime = zapcore.ISO8601TimeEncoder

    developmentCfg := zap.NewDevelopmentEncoderConfig()
    developmentCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

    consoleEncoder := zapcore.NewConsoleEncoder(developmentCfg)
    fileEncoder := zapcore.NewJSONEncoder(productionCfg)

    core := zapcore.NewTee(
        zapcore.NewCore(consoleEncoder, stdout, level),
        zapcore.NewCore(fileEncoder, file, level),
    )

    Log= zap.New(core)
}

type (
	responseData struct {
		code int
		size int
		body string
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	r.responseData.body=string(b)
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.code = statusCode
}

func logResponse(code, size int, url string, duration time.Duration) {
	Log.Info("Response",
		zap.String("url", url),
		zap.Int("status code", code),
		zap.Int("size", size),
		zap.Duration("duration", duration),
	)
}

func logRequest(uri, method string) {
	Log.Info("Request",
		zap.String("method", method),
		zap.String("uri", uri),
	)
}

// LogMiddleware realises middleware for logging requests and responses
func LogMiddleware(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		uri := r.RequestURI
		method := r.Method

		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData: &responseData{
				code: 0,
				size: 0,
				body: "",
			},
		}
		logRequest(uri, method)

		h.ServeHTTP(&lw, r)

		duration := time.Since(start)	
		logResponse(lw.responseData.code, lw.responseData.size, uri, duration)
		if lw.responseData.code >=400{
			Log.Error("Response with error", zap.String("Error", lw.responseData.body))
		}
		Log.Debug("------------------------------------------------")
	}

	return http.HandlerFunc(logFn)
}

func StringMark(key, value string) zap.Field {
	return zap.String(key, value)
}

func IntMark(key string, value int) zap.Field {
	return zap.Int(key, value)
}

func Uint32Mark(key string, value uint32) zap.Field {
	return zap.Uint32(key, value)
}

func Float32Mark(key string, value float32) zap.Field {
	return zap.Float32(key, value)
}

func ErrorMark(err error) zap.Field {
	return zap.Error(err)
}