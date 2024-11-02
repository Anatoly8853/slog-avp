package v2

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gookit/slog"
	"github.com/gookit/slog/handler"
	_ "github.com/mattn/go-sqlite3"
)

// Application - структура приложения, содержащая логгер
type Application struct {
	Log *slog.Logger
}

// Глобальные переменные для управления режимами логирования
var (
	logConsole  = true  // Логирование в консоль (по умолчанию: включено)
	logToDB     = false // Логирование в SQLite (по умолчанию: выключено)
	IsDebugMode = true  // Режим отладки (по умолчанию: включен)
	IsInfoMode  = true  // Информационный режим (по умолчанию: включен)
	IsWarnMode  = true  // Режим предупреждений (по умолчанию: включен)
	db          *sql.DB // Подключение к SQLite
	DBPath      string  // Путь к базе данных SQLite
)

// SetLogConsole устанавливает флаг логирования в консоль
func SetLogConsole(value bool) {
	logConsole = value
}

// SetLogToDB устанавливает флаг логирования в базу данных
func SetLogToDB(value bool) {
	logToDB = value
}

// SetIsDebugMode устанавливает флаг режима отладки
func SetIsDebugMode(value bool) {
	IsDebugMode = value
}

// SetIsInfoMode устанавливает флаг информационного режима
func SetIsInfoMode(value bool) {
	IsInfoMode = value
}

// SetIsWarnMode устанавливает флаг режима предупреждений
func SetIsWarnMode(value bool) {
	IsWarnMode = value
}

// hasDBExtension проверяет, имеет ли файл расширение .db
func hasDBExtension(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".db")
}

// CustomFormatter - пользовательский форматтер для вывода логов
type CustomFormatter struct{}

// Format форматирует запись лога и записывает в БД, если включено логирование в SQLite
func (f *CustomFormatter) Format(record *slog.Record) ([]byte, error) {
	caller := record.Caller
	fileName := filepath.Base(caller.File)
	funcName := getFunctionName(caller.PC)

	// Форматируем сообщение для вывода
	logMessage := fmt.Sprintf("[%s] [%s] [%s:%d,%s] [%s]\n",
		record.Level.String(),
		record.Time.Format("2006-01-02 15:04:05"),
		fileName,
		caller.Line,
		funcName,
		record.Message,
	)
	return []byte(logMessage), nil
}

// getFunctionName возвращает имя функции из Program Counter (PC)
func getFunctionName(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}
	fullName := fn.Name()
	parts := strings.Split(fullName, "/")
	shortName := parts[len(parts)-1]
	if dotIndex := strings.LastIndex(shortName, "."); dotIndex != -1 {
		shortName = shortName[dotIndex+1:]
	}
	return shortName
}

// setupDBLogger инициализирует базу данных и создает таблицу логов, если она не существует
func setupDBLogger() error {
	dbPath := "log/logs.db"
	if DBPath != "" && len(DBPath) > 3 {
		dbPath = DBPath
	}

	// Проверка на наличие расширения .db
	if !hasDBExtension(dbPath) {
		dbPath = dbPath + ".db"
	}

	// Создание директории для базы данных, если она не существует
	err := os.MkdirAll(filepath.Dir(dbPath), 0755)
	if err != nil {
		return fmt.Errorf("ошибка создания директории для базы данных: %v", err)
	}

	// Открытие соединения с базой данных
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("ошибка открытия базы данных: %v", err)
	}

	// Создание таблицы для логов, если она еще не создана
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME,
        level VARCHAR(10),
        message TEXT,
        file_name TEXT,
        line_number INTEGER,
        function_name TEXT
    );`

	_, err = db.Exec(createTableSQL)
	return err
}

// writeLogToDB записывает лог в базу данных SQLite
func writeLogToDB(level, message, fileName string, lineNumber int, functionName string) error {
	if db == nil {
		return fmt.Errorf("подключение к базе данных не инициализировано")
	}

	// Подготовка SQL-запроса для вставки данных
	stmt, err := db.Prepare(`
        INSERT INTO logs (timestamp, level, message, file_name, line_number, function_name)
        VALUES (?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Выполнение SQL-запроса с текущими данными лога
	_, err = stmt.Exec(
		time.Now(),
		level,
		message,
		fileName,
		lineNumber,
		functionName,
	)
	return err
}

// DBHandler - обработчик для логирования в базу данных
type DBHandler struct{}

// Handle записывает лог в базу данных
func (h *DBHandler) Handle(record *slog.Record) error {
	caller := record.Caller
	fileName := filepath.Base(caller.File)
	funcName := getFunctionName(caller.PC)

	return writeLogToDB(
		record.Level.String(),
		record.Message,
		fileName,
		caller.Line,
		funcName,
	)
}

// Close завершает работу обработчика (метод необходим для интерфейса slog.Handler)
func (h *DBHandler) Close() error {
	return nil // Метод может оставаться пустым, если закрывать ничего не нужно
}

// Flush завершает запись логов (метод необходим для интерфейса slog.Handler)
func (h *DBHandler) Flush() error {
	return nil // Метод может оставаться пустым, если завершать запись логов не требуется
}

// IsHandling проверяет, может ли обработчик обрабатывать указанный уровень логирования
func (h *DBHandler) IsHandling(level slog.Level) bool {
	// Можно вернуть true для обработки всех уровней
	return true
}

// SetupLogger настраивает логгер с указанными обработчиками и форматтером
func SetupLogger() *slog.Logger {
	// Создаем новый логгер
	logger := slog.New()
	formatter := &CustomFormatter{}

	// Настройка логирования в БД, если это включено
	if logToDB {
		err := setupDBLogger()
		if err != nil {
			fmt.Printf("Не удалось настроить логгер базы данных: %v\n", err)
		} else {
			// Добавляем обработчик базы данных
			dbHandler := &DBHandler{}
			logger.AddHandler(dbHandler)
		}
	}

	// Настройка логирования в консоль, если это включено
	if logConsole {
		consoleHandler := handler.NewConsoleHandler(getLogLevels())
		consoleHandler.SetFormatter(formatter)
		logger.AddHandler(consoleHandler)
	} else if !logToDB {
		// Если ни консоль, ни БД не включены, настраиваем логирование в файл
		now := time.Now()
		logFilePath := fmt.Sprintf("log/error-%s.log", now.Format("02-01-2006"))

		err := os.MkdirAll(filepath.Dir(logFilePath), 0755)
		if err != nil {
			panic(fmt.Sprintf("Ошибка создания директории логов: %v", err))
		}

		logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(fmt.Sprintf("Ошибка открытия файла логов: %v", err))
		}

		fileHandler := handler.NewIOWriterHandler(logFile, getLogLevels())
		fileHandler.SetFormatter(formatter)
		logger.AddHandler(fileHandler)
	}

	return logger
}

// getLogLevels возвращает список уровней логирования, которые включены
func getLogLevels() []slog.Level {
	levels := []slog.Level{slog.ErrorLevel, slog.FatalLevel}

	if IsWarnMode {
		levels = append(levels, slog.WarnLevel)
	}
	if IsInfoMode {
		levels = append(levels, slog.InfoLevel)
	}
	if IsDebugMode {
		levels = append(levels, slog.DebugLevel)
	}

	return levels
}

// SetupApplication создает и возвращает экземпляр приложения с настроенным логгером
func SetupApplication() *Application {
	return &Application{Log: SetupLogger()}
}
