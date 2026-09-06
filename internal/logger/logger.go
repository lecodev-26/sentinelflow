package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Init inicializa el logger con la configuración
func Init(cfg *struct {
	Level  string
	Format string
	Output string
}) {
	log = logrus.New()

	// Nivel de log
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Formato
	if cfg.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	// Salida
	if cfg.Output == "file" {
		file, err := os.OpenFile("logs/sentinelflow.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
		} else {
			log.SetOutput(os.Stdout)
		}
	} else {
		log.SetOutput(os.Stdout)
	}
}

// Get retorna el logger instanciado
func Get() *logrus.Logger {
	if log == nil {
		log = logrus.New()
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		log.SetLevel(logrus.InfoLevel)
	}
	return log
}

// Funciones helper
func Debug(args ...interface{}) {
	Get().Debug(args...)
}

func Info(args ...interface{}) {
	Get().Info(args...)
}

func Warn(args ...interface{}) {
	Get().Warn(args...)
}

func Error(args ...interface{}) {
	Get().Error(args...)
}

func Fatal(args ...interface{}) {
	Get().Fatal(args...)
}

func Debugf(format string, args ...interface{}) {
	Get().Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	Get().Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	Get().Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	Get().Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	Get().Fatalf(format, args...)
}