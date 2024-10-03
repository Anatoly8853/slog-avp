package main

import (
	"github.com/Anatoly8853/slog-avp"
)

func SetupApplication() *slogavp.Application {
	// Настройка логгера перед его инициализацией
	slogavp.SetLogConsole(false) // Логи будут записываться в файл
	slogavp.SetIsDebugMode(true)
	slogavp.SetIsInfoMode(true)
	slogavp.SetIsWarnMode(true)
	// Настраиваем логгер
	logger := slogavp.SetupLogger()
	// Создаем экземпляр Application с настроенным логгером
	return &slogavp.Application{Log: logger}
}

func main() {
	// Настраиваем логгер
	app := SetupApplication()
	app.Log.Println("Ура работает")
}
