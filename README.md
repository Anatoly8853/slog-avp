# SLOGAVP - Расширенный логгер с поддержкой SQLite
<h2>Описание</h2>
  <p>SLOGAVP - это мощная библиотека логирования для Go, 
     которая предоставляет расширенные возможности ведения 
     журналов с поддержкой записи в SQLite базу данных и консоль. 
     Библиотека построена на основе пакета slog и предлагает гибкую настройку
     форматирования и хранения логов.</p>
<h2>Основные возможности</h2>
  <ul>
    <li>Гибкая настройка уровней логирования (Debug, Info, Warn, Error, Fatal)</li>
    <li>Параллельная запись логов в консоль и SQLite базу данных</li>
    <li>Автоматическое создание структуры базы данных и директорий</li>
    <li>Подробная информация о месте вызова (имя файла, номер строки, имя функции)</li>
    <li>Настраиваемый формат вывода логов</li>
    <li>Возможность включения/отключения различных режимов логирования</li>
  </ul>

## Установка

```shell
go get github.com/Anatoly8853/slog-avp
```

Зависимости
github.com/gookit/slog - базовый функционал логирования
github.com/mattn/go-sqlite3 - драйвер SQLite
Базовое использование

```go
package main

import "github.com/Anatoly8853/slog-avp"

func main() {
// Настройка параметров логирования
slogavp.SetLogConsole(true)      // Включаем вывод в консоль
slogavp.SetLogToDB(true)         // Включаем логирование в БД
slogavp.SetIsDebugMode(true)     // Включаем режим отладки

    // Создаем логгер
    logger := slogavp.SetupLogger()
    
    // Примеры использования
    logger.Debug("Отладочное сообщение")
    logger.Info("Информационное сообщение")
    logger.Warn("Предупреждение")
    logger.Error("Сообщение об ошибке")
}
```

Пример использования структуры Application:
```go
package main

import "github.com/Anatoly8853/slog-avp"

func main() {
	// Включаем логирование в БД
	slogavp.SetLogToDB(true)
	//Отключаем запись в консоль и если ведем запись в бд отключается запись в файл
	slogavp.SetLogConsole(false)
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
```


Структура логов в БД
Логи сохраняются в таблице следующей структуры:

```SQLite
CREATE TABLE logs (
id INTEGER PRIMARY KEY AUTOINCREMENT,
timestamp DATETIME,
level VARCHAR(10),
message TEXT,
file_name TEXT,
line_number INTEGER,
function_name TEXT
);
```

<h2>Конфигурация</h2>
  <p>Библиотека предоставляет следующие функции для настройки:</p>
  <ul>
    <li>SetLogConsole(bool) - включение/выключение вывода в консоль</li>
    <li>SetLogToDB(bool) - включение/выключение записи в БД</li>
    <li>SetIsDebugMode(bool) - управление режимом отладки</li>
    <li>SetIsInfoMode(bool) - управление информационным режимом</li>
    <li>SetIsWarnMode(bool) - управление режимом предупреждений</li>
  </ul>