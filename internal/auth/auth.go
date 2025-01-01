package authenticator

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	gopherror "github.com/Alena-Kurushkina/gophermart.git/internal/errors"
	"github.com/Alena-Kurushkina/gophermart.git/internal/logger"
	"github.com/golang-jwt/jwt/v4"
	"github.com/satori/go.uuid"
)

// Claims — структура утверждений, которая включает стандартные утверждения и
// одно пользовательское UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID
}

const tokenExp = time.Hour * 3

// TODO: перенести в env
const secretKey = "secretkey"

// buildJWTString создаёт токен и возвращает его в виде строки.
func buildJWTString(id uuid.UUID) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		// собственное утверждение
		UserID: id,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

func getUserIDFromJWT(tokenString string) (uuid.UUID, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secretKey), nil
		})
	if err != nil {
		return uuid.Nil, err
	}
	if !token.Valid {
		return uuid.Nil, gopherror.ErrTokenInvalid
	}
	// if claims.UserID == uuid.Nil {
	// 	return uuid.Nil, sherr.ErrNoUserIDInToken
	// }
	logger.Log.Info("User token is valid")
	return claims.UserID, nil
}

func SetNewJWTInCookie(w http.ResponseWriter, userID uuid.UUID) error {
	jwt, err := buildJWTString(userID)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: "token", Value: jwt, MaxAge: 0})
	return nil
}

// AuthMiddleware realises middleware for user authentication
func AuthMiddleware(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			switch {
			case errors.Is(err, http.ErrNoCookie):
				logger.Log.Info("No cookie in request")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		userID, err := getUserIDFromJWT(cookie.Value)
		if err != nil {
			if errors.Is(err, gopherror.ErrTokenInvalid){
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		q := r.URL.Query()
		q.Add("userUUID", userID.String())
		r.URL.RawQuery = q.Encode()

		logger.Log.Info("Got user id from token", 
			logger.StringMark("User ID", userID.String()),
		)

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(logFn)
}
