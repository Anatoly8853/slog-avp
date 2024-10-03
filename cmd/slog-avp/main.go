package main

import (
	slogavp "github.com/Anatoly8853/slog-avp"
	"github.com/gookit/slog"
)

type Application struct {
	Log *slog.Logger
}

func SetupApplication() *Application {
	// Настройка логгера перед его инициализацией
	slogavp.SetLogConsole(false) // Логи будут записываться в файл
	slogavp.SetIsDebugMode(true)
	slogavp.SetIsInfoMode(true)
	slogavp.SetIsWarnMode(true)
	// Настраиваем логгер
	logger := slogavp.SetupLogger()
	// Создаем экземпляр Application с настроенным логгером
	return &Application{Log: logger}
}

func main() {
	// Настраиваем логгер
	app := SetupApplication()
	app.Log.Println("Ура работает")
}
