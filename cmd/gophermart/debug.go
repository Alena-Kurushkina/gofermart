package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type AccrualResponse struct {
	Order string `json:"order"`
	Status string `json:"status"`
	Accrual float32 `json:"accrual,omitempty"`
}

var Client *http.Client 
var Log *zap.SugaredLogger = zap.NewNop().Sugar()


// func GenerateOrderNumber(){
// 	ds := random.DigitString(5, 15)
// 	cd, err := luhnCheckDigit(ds)
// 	if err != nil {
// 		return "", fmt.Errorf("cannot calculate check digit: %s", err)
// 	}
// 	return ds + strconv.Itoa(cd), nil
// }

// func luhnCheckDigit(s string) (int, error) {
// 	number, err := strconv.Atoi(s)
// 	if err != nil {
// 		return 0, err
// 	}

// 	checkNumber := luhnChecksum(number)

// 	if checkNumber == 0 {
// 		return 0, nil
// 	}
// 	return 10 - checkNumber, nil
// }

// func luhnChecksum(number int) int {
// 	var luhn int

// 	for i := 0; number > 0; i++ {
// 		cur := number % 10

// 		if i%2 == 0 { // even
// 			cur = cur * 2
// 			if cur > 9 {
// 				cur = cur%10 + cur/10
// 			}
// 		}

// 		luhn += cur
// 		number = number / 10
// 	}
// 	return luhn % 10
// }

func GetAccrual(number string){
	for {
		req, err:=http.NewRequest(http.MethodGet, "http://localhost:8080/api/orders/"+number, nil)
		if err != nil {
			fmt.Errorf(err.Error())
		}
		resp, err:=Client.Do(req)
		if err != nil {
			fmt.Errorf(err.Error())
		}
		//fmt.Println("Статус-код ", resp.Status)
		defer resp.Body.Close()
		body,err:=io.ReadAll(resp.Body)
		if err != nil {
			fmt.Errorf(err.Error())
		}

		rr:=AccrualResponse{}
		err = json.Unmarshal(body, &rr)
		if err != nil {
			fmt.Errorf(err.Error())
		}
		//fmt.Println(rr)
		Log.Info(rr)
		if rr.Status=="PROCESSED"{
			break
		}
	}
}

func PostOrder(order string){
	o:=[]byte(`
		{
			"order": "`+order+`", 
			"goods": [
				{
					"description": "smth",
					"price": 45477.00
				},
				{
					"description": "smth ER13",
					"price": 6476.76
				}
			]
		}
	`)
	request, err := http.NewRequest(http.MethodPost, "http://localhost:8080/api/orders",bytes.NewBuffer(o))
	if err != nil {
		fmt.Errorf(err.Error())
	}

	request.Header.Add("Content-Type", "application/json")

	// отправляем запрос
	response, err := Client.Do(request)
	if err != nil {
		fmt.Errorf(err.Error())
	}

	// ответ
	fmt.Println("Статус-код ", response.Status)
	defer response.Body.Close()
}

func main_d() {
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{
		"/Users/alena/log/accrual_get.log",
	}
	zl, err := cfg.Build()
	if err!=nil{
		panic(err)
	}
	Log = zl.Sugar()

	// добавляем HTTP-клиент
	Client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
	}}


	// m := []byte(`
	// 		{
	// 			"match": "` + "ER13" + `",
	// 			"reward": 5,
	// 			"reward_type": "%"
	// 		}
	// 	`)
	// request, err := http.NewRequest(http.MethodPost, "http://localhost:8080/api/goods",bytes.NewBuffer(m))
	// if err != nil {
	// 	fmt.Errorf(err.Error())
	// }

	// request.Header.Add("Content-Type", "application/json")

	// // отправляем запрос
	// response, err := Client.Do(request)
	// if err != nil {
	// 	fmt.Errorf(err.Error())
	// }

	// // ответ
	// fmt.Println("Статус-код ", response.Status)
	// defer response.Body.Close()

	accruals:=[]string{"1234561239","5580473372024733","123455"}

	for _,ac:=range accruals{
		PostOrder(ac)
		GetAccrual(ac)
	}
}
