package slogavp

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

var (
	logConsole  = true  // Вывод в консоль (по умолчанию: включен)
	logToDB     = false // Логирование в SQLite (по умолчанию: выключено)
	IsDebugMode = true  // Режим отладки (по умолчанию: включен)
	IsInfoMode  = true  // Информационный режим (по умолчанию: включен)
	IsWarnMode  = true  // Режим предупреждений (по умолчанию: включен)
	db          *sql.DB // Подключение к SQLite
	DBPath      string
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

// CustomFormatter предоставляет пользовательский формат логов
type CustomFormatter struct{}

// Format реализует интерфейс slog.Formatter
func (f *CustomFormatter) Format(record *slog.Record) ([]byte, error) {
	caller := record.Caller
	fileName := filepath.Base(caller.File)
	funcName := getFunctionName(caller.PC)

	// Если включено логирование в БД, записываем лог
	if logToDB {
		err := writeLogToDB(
			record.Level.String(),
			record.Message,
			fileName,
			caller.Line,
			funcName,
		)
		if err != nil {
			fmt.Printf("Ошибка записи в БД: %v\n", err)
		}
	}

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

// getFunctionName возвращает короткое имя функции из PC
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

// setupDBLogger инициализирует подключение к SQLite и создает таблицу логов
func setupDBLogger() error {
	// Определяем путь к базе данных
	dbPath := "log/logs.db"
	if DBPath != "" && len(DBPath) > 3 {
		dbPath = DBPath
	}

	// Проверяем и добавляем расширение .db если его нет
	if !hasDBExtension(dbPath) {
		dbPath = dbPath + ".db"
	}

	// Создаем директорию, если она не существует
	err := os.MkdirAll(filepath.Dir(dbPath), 0755)
	if err != nil {
		return fmt.Errorf("ошибка создания директории для базы данных: %v", err)
	}

	// Открываем соединение с базой данных
	var err1 error
	db, err1 = sql.Open("sqlite3", dbPath)
	if err1 != nil {
		return fmt.Errorf("ошибка открытия базы данных: %v", err1)
	}

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

// logToDB записывает лог в базу данных SQLite
func writeLogToDB(level, message, fileName string, lineNumber int, functionName string) error {
	if db == nil {
		return fmt.Errorf("подключение к базе данных не инициализировано")
	}

	stmt, err := db.Prepare(`
        INSERT INTO logs (timestamp, level, message, file_name, line_number, function_name)
        VALUES (?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

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

// SetupLogger настраивает и возвращает логгер
func SetupLogger() *slog.Logger {
	// Создаем логгер
	logger := slog.New()

	// Устанавливаем форматтер
	formatter := &CustomFormatter{}

	// Настройка цветного вывода
	slog.Configure(func(logger *slog.SugaredLogger) {
		f := logger.Formatter.(*slog.TextFormatter)
		f.EnableColor = true
	})

	if logToDB {
		err := setupDBLogger()
		if err != nil {
			fmt.Printf("Не удалось настроить логгер базы данных: %v\n", err)
		}
	}

	if logConsole {
		consoleHandler := handler.NewConsoleHandler(getLogLevels())
		consoleHandler.SetFormatter(formatter)
		logger.AddHandler(consoleHandler)
	} else {
		if !logToDB {
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
	}

	return logger
}

// getLogLevels возвращает список активных уровней логирования
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

// SetupApplication создает и настраивает новый экземпляр приложения
func SetupApplication() *Application {
	return &Application{Log: SetupLogger()}
}
