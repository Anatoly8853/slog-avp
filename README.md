# slog-avp
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
Установка
go get github.com/yourusername/slogavp
Зависимости
github.com/gookit/slog - базовый функционал логирования
github.com/mattn/go-sqlite3 - драйвер SQLite
Базовое использование
package main

import "github.com/yourusername/slogavp"

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
Структура логов в БД
Логи сохраняются в таблице следующей структуры:

CREATE TABLE logs (
id INTEGER PRIMARY KEY AUTOINCREMENT,
timestamp DATETIME,
level VARCHAR(10),
message TEXT,
file_name TEXT,
line_number INTEGER,
function_name TEXT
);
Конфигурация
Библиотека предоставляет следующие функции для настройки:

SetLogConsole(bool) - включение/выключение вывода в консоль
SetLogToDB(bool) - включение/выключение записи в БД
SetIsDebugMode(bool) - управление режимом отладки
SetIsInfoMode(bool) - управление информационным режимом
SetIsWarnMode(bool) - управление режимом предупреждений