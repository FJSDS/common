package utils

func SafeGO(errF func(interface{}), f func()) {
	go func() {
		defer Recover(errF)

		f()
	}()
}
