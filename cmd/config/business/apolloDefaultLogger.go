package business

import (
	"fmt"
	"mango/pkg/log"
)

type DefaultLogger struct {
}

func (d *DefaultLogger) Debugf(format string, params ...interface{}) {
	//log.Debug("agollo", format, params...)
}

func (d *DefaultLogger) Infof(format string, params ...interface{}) {
	log.Info("agollo", format, params...)
}

func (d *DefaultLogger) Warnf(format string, params ...interface{}) {
	log.Warning("agollo", format, params...)
}

func (d *DefaultLogger) Errorf(format string, params ...interface{}) {
	log.Error("agollo", format, params...)
}

func (d *DefaultLogger) Debug(v ...interface{}) {
	//log.Debug("agollo", "%v", fmt.Sprint(v...))
}
func (d *DefaultLogger) Info(v ...interface{}) {
	log.Info("agollo", "%v", fmt.Sprint(v...))
}

func (d *DefaultLogger) Warn(v ...interface{}) {
	log.Warning("agollo", "%v", fmt.Sprint(v...))
}

func (d *DefaultLogger) Error(v ...interface{}) {
	log.Error("agollo", "%v", fmt.Sprint(v...))
}
