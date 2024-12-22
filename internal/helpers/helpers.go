package helpers

func AccrualToBase(accrual float32) uint32 {
	return uint32(accrual*100)
}

func BaseToAccrual(base uint32) float32 {
	return float32(base)/100
}