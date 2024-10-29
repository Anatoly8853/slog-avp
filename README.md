# SLOGAVP - Расширенный логгер с поддержкой SQLite
Описание
# SLOGAVP - это мощная библиотека логирования для Go, которая предоставляет расширенные возможности ведения журналов с поддержкой записи в SQLite базу данных и консоль. Библиотека построена на основе пакета slog и предлагает гибкую настройку форматирования и хранения логов.

Основные возможности
Гибкая настройка уровней логирования (Debug, Info, Warn, Error, Fatal)
Параллельная запись логов в консоль и SQLite базу данных
Автоматическое создание структуры базы данных и директорий
Подробная информация о месте вызова (имя файла, номер строки, имя функции)
Настраиваемый формат вывода логов
Возможность включения/отключения различных режимов логирования

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