package utils

func Recover(panicPrint func(e interface{})) {
	if e := recover(); e != nil {
		if panicPrint != nil {
			panicPrint(e)
		}
	}
}

func RecoverWithFunc(panicPrint func(e interface{}), f func()) {
	if f == nil {
		return
	}

	defer Recover(panicPrint)
	f()
}
