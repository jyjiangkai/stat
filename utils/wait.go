package utils

import "time"

// WaitFor if times > 0, with retry at most xx times, else will retry util success
func WaitFor(times, testInterval int, testF func() error) error {
	it := 0
	var err error
	for {
		it++
		if times > 0 && it > times {
			return err
		}

		err = testF()

		if err == nil {
			return nil
		}
		time.Sleep(time.Second * time.Duration(testInterval))
	}
}
