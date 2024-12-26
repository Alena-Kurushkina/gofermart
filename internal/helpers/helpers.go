package helpers

import "golang.org/x/crypto/bcrypt"

func AccrualToBase(accrual float32) uint32 {
	return uint32(accrual*100)
}

func BaseToAccrual(base uint32) float32 {
	return float32(base)/100
}

func HashPassword(password string) (string, error) {
	hash, err:=bcrypt.GenerateFromPassword([]byte(password),bcrypt.MinCost)
	if err!=nil{
		return "", err
	}

	return string(hash), nil
}

func CompareHashPassword(hash, actualPassword string) bool {
	err:=bcrypt.CompareHashAndPassword([]byte(hash),[]byte(actualPassword))
	
	return err==nil
}