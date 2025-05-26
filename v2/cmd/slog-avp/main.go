package main

import slogavp "github.com/Anatoly8853/slog-avp/v2"

func main() {
	// Включаем логирование в БД
	//slogavp.SetLogToDB(true)   // Логирование в SQLite (по умолчанию: выключено)
	//Отключаем запись в консоль и если ведем запись в бд отключается запись в файл
	//slogavp.SetLogConsole(true) // Логирование в консоль (по умолчанию: включено)
	//slogavp.SetLogConsole(false)
	//slogavp.DBPath = "log/logs.db" путь и файл по умолчанию

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
