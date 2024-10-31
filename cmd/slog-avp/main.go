package main

import slogavp "github.com/Anatoly8853/slog-avp"

func main() {
	// Включаем логирование в БД
	slogavp.SetLogToDB(true)
	slogavp.SetLogConsole(false)

	// Создаем экземпляр приложения с настроенным логгером
	app := slogavp.SetupApplication()

	// Используем логгер через структуру приложения
	app.Log.Info("Приложение запущено")
	app.Log.Debug("Отладочное сообщение")
	app.Log.Error("Произошла ошибка")

	// В других функциях или методах
	doSomething(app)
}

func doSomething(app *slogavp.Application) {
	// Используем логгер
	app.Log.Info("Выполняется doSomething")
}
