package main

import slogavp "github.com/Anatoly8853/slog-avp"

func main() {
	// Настройка параметров логирования
	//slogavp.DBPath = "logs/logs"
	slogavp.SetLogConsole(false) // Включаем вывод в консоль
	//slogavp.SetLogToDB(true)     // Включаем логирование в БД
	//slogavp.SetIsDebugMode(true) // Включаем режим отладки
	//slogavp.SetIsInfoMode(true)  // Включаем информационный режим
	//slogavp.SetIsWarnMode(true)  // Включаем режим предупреждений

	// Создаем экземпляр приложения с настроенным логгером
	logger := slogavp.SetupLogger()

	// Теперь все логи будут автоматически записываться в БД
	logger.Debug("Отладочное сообщение")
	logger.Info("Информационное сообщение")
	logger.Warn("Предупреждение")
	logger.Error("Сообщение об ошибке")

	loggers := slogavp.SetupApplication()
	loggers.Log.Error()
}
