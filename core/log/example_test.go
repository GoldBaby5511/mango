package log_test

import (
	"xlddz/core/log"
)

func Example() {
	name := "Leaf"

	log.Debug("log", "My name is %v", name)
	log.Info("log", "My name is %v", name)
	log.Error("log", "My name is %v", name)
	// log.Fatal("My name is %v", name)

	logger, err := log.New()
	if err != nil {
		return
	}
	defer logger.Close()

	//logger.Debug("will not print")
	//logger.Release("My name is %v", name)

	log.Export(logger)

	log.Debug("log", "will not print")
	log.Info("log", "My name is %v", name)
}
