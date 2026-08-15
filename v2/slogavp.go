package v2

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gookit/slog"
	"github.com/gookit/slog/handler"
	_ "github.com/mattn/go-sqlite3"
)

// ANSI escape-коды для цветов
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
)

// ---------------------------------------------------------------------
// Новый API: Config + Option. Рекомендуется для нового кода.
// ---------------------------------------------------------------------

// Config описывает поведение логгера. Заменяет собой прежние пакетные
// глобальные переменные, поэтому несколько Application могут жить
// в одном процессе независимо друг от друга и безопасно настраиваться.
type Config struct {
	LogConsole bool   // писать логи в stdout
	LogToDB    bool   // писать логи в SQLite
	Debug      bool   // включить уровень Debug
	Info       bool   // включить уровень Info
	Warn       bool   // включить уровень Warn
	DBPath     string // путь к файлу SQLite (используется, если LogToDB=true)
	LogDir     string // директория для файлового логгера (по умолчанию "log")
}

// Option настраивает Config. Используется вместе с New.
type Option func(*Config)

func WithConsole(enabled bool) Option { return func(c *Config) { c.LogConsole = enabled } }

// WithDB включает логирование в SQLite и задаёт путь к базе.
func WithDB(path string) Option {
	return func(c *Config) {
		c.LogToDB = true
		c.DBPath = path
	}
}
func WithDebug(enabled bool) Option { return func(c *Config) { c.Debug = enabled } }
func WithInfo(enabled bool) Option  { return func(c *Config) { c.Info = enabled } }
func WithWarn(enabled bool) Option  { return func(c *Config) { c.Warn = enabled } }
func WithLogDir(dir string) Option  { return func(c *Config) { c.LogDir = dir } }

func defaultConfig() Config {
	return Config{
		LogConsole: true,
		LogToDB:    false,
		Debug:      true,
		Info:       true,
		Warn:       true,
		LogDir:     "log",
	}
}

func (c Config) levels() []slog.Level {
	levels := []slog.Level{slog.ErrorLevel, slog.FatalLevel}
	if c.Warn {
		levels = append(levels, slog.WarnLevel)
	}
	if c.Info {
		levels = append(levels, slog.InfoLevel)
	}
	if c.Debug {
		levels = append(levels, slog.DebugLevel)
	}
	return levels
}

func (c Config) hasLevel(level slog.Level) bool {
	for _, l := range c.levels() {
		if l == level {
			return true
		}
	}
	return false
}

// Application объединяет настроенный логгер с ресурсами, которыми он
// владеет (соединение с БД, фоновый писатель, открытый файл), чтобы их
// можно было детерминированно освободить через Close.
type Application struct {
	Log *slog.Logger

	cfg     Config
	db      *sql.DB
	stmt    *sql.Stmt
	dbCh    chan dbLogEntry
	dbWG    sync.WaitGroup
	logFile *os.File
}

type dbLogEntry struct {
	timestamp time.Time
	level     string
	message   string
	fileName  string
	line      int
	function  string
}

// hasDBExtension проверяет, имеет ли файл расширение .db
func hasDBExtension(filename string) bool {
	return strings.EqualFold(filepath.Ext(filename), ".db")
}

// CustomFormatter - пользовательский форматтер для вывода логов
type CustomFormatter struct{}

// Format форматирует запись лога в текстовую строку с цветом уровня
func (f *CustomFormatter) Format(record *slog.Record) ([]byte, error) {
	caller := record.Caller
	fileName := filepath.Base(caller.File)
	funcName := getFunctionName(caller.PC)

	levelColor := getLevelColor(record.Level)
	levelStr := fmt.Sprintf("%s[%s]%s", levelColor, record.Level.String(), colorReset)

	logMessage := fmt.Sprintf("%s [%s] [%s:%d,%s] [%s]\n",
		levelStr,
		record.Time.Format("2006-01-02 15:04:05"),
		fileName,
		caller.Line,
		funcName,
		record.Message,
	)

	return []byte(logMessage), nil
}

func getLevelColor(level slog.Level) string {
	switch level {
	case slog.InfoLevel:
		return colorGreen
	case slog.ErrorLevel, slog.FatalLevel:
		return colorRed
	case slog.WarnLevel:
		return colorYellow
	case slog.DebugLevel:
		return colorGray
	default:
		return colorReset
	}
}

// getFunctionName возвращает короткое имя функции по Program Counter (PC)
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

// setupDB открывает SQLite, создаёт схему/индексы, готовит INSERT один раз
// и запускает фоновую горутину-писатель.
func (app *Application) setupDB() error {
	dbPath := "log/logs.db"
	if app.cfg.DBPath != "" && len(app.cfg.DBPath) > 3 {
		dbPath = app.cfg.DBPath
	}
	if !hasDBExtension(dbPath) {
		dbPath += ".db"
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("создание директории для базы данных: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("открытие базы данных: %w", err)
	}

	const createTableSQL = `
    CREATE TABLE IF NOT EXISTS logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME,
        level VARCHAR(10),
        message TEXT,
        file_name TEXT,
        line_number INTEGER,
        function_name TEXT
    );`
	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return fmt.Errorf("создание таблицы logs: %w", err)
	}

	const createIndexesSQL = `
    CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
    CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);`
	if _, err := db.Exec(createIndexesSQL); err != nil {
		db.Close()
		return fmt.Errorf("создание индексов logs: %w", err)
	}

	stmt, err := db.Prepare(`
        INSERT INTO logs (timestamp, level, message, file_name, line_number, function_name)
        VALUES (?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		db.Close()
		return fmt.Errorf("подготовка INSERT-запроса: %w", err)
	}

	app.db = db
	app.stmt = stmt
	app.dbCh = make(chan dbLogEntry, 256)

	app.dbWG.Add(1)
	go app.runDBWriter()

	return nil
}

// runDBWriter вычитывает dbCh и пишет записи в SQLite на одной горутине.
// Это (а) сериализует запись, что важно для SQLite при конкурентных писателях,
// и (б) не блокирует вызывающий код на дисковом I/O.
func (app *Application) runDBWriter() {
	defer app.dbWG.Done()
	for entry := range app.dbCh {
		_, err := app.stmt.Exec(
			entry.timestamp,
			entry.level,
			entry.message,
			entry.fileName,
			entry.line,
			entry.function,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "logger: не удалось записать лог в БД: %v\n", err)
		}
	}
}

// DBHandler - обработчик, передающий записи в фоновый писатель SQLite
type DBHandler struct {
	app *Application
}

// Handle кладёт запись в канал; не блокирует вызывающего при переполнении
// (в этом случае запись отбрасывается и об этом сообщается в stderr).
func (h *DBHandler) Handle(record *slog.Record) error {
	caller := record.Caller
	entry := dbLogEntry{
		timestamp: time.Now(),
		level:     record.Level.String(),
		message:   record.Message,
		fileName:  filepath.Base(caller.File),
		line:      caller.Line,
		function:  getFunctionName(caller.PC),
	}

	select {
	case h.app.dbCh <- entry:
	default:
		fmt.Fprintln(os.Stderr, "logger: канал записи в БД переполнен, запись отброшена")
	}
	return nil
}

func (h *DBHandler) Close() error { return nil }
func (h *DBHandler) Flush() error { return nil }

// IsHandling теперь реально учитывает настроенные уровни (в исходной
// версии всегда возвращал true, и БД получала абсолютно все записи).
func (h *DBHandler) IsHandling(level slog.Level) bool {
	return h.app.cfg.hasLevel(level)
}

// rotatingFileHandler - файловый обработчик с ежедневной ротацией:
// файл переоткрывается заново при смене календарного дня.
type rotatingFileHandler struct {
	app       *Application
	mu        sync.Mutex
	day       string
	formatter slog.Formatter
}

func newRotatingFileHandler(app *Application) *rotatingFileHandler {
	return &rotatingFileHandler{app: app, formatter: &CustomFormatter{}}
}

func (h *rotatingFileHandler) ensureFile() error {
	today := time.Now().Format("02-01-2006")
	if h.day == today && h.app.logFile != nil {
		return nil
	}

	if err := os.MkdirAll(h.app.cfg.LogDir, 0755); err != nil {
		return fmt.Errorf("создание директории логов: %w", err)
	}

	logFilePath := filepath.Join(h.app.cfg.LogDir, fmt.Sprintf("error-%s.log", today))
	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("открытие файла логов: %w", err)
	}

	if h.app.logFile != nil {
		h.app.logFile.Close()
	}
	h.app.logFile = f
	h.day = today
	return nil
}

func (h *rotatingFileHandler) Handle(record *slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.ensureFile(); err != nil {
		return err
	}

	data, err := h.formatter.Format(record)
	if err != nil {
		return err
	}
	_, err = h.app.logFile.Write(data)
	return err
}

func (h *rotatingFileHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.app.logFile != nil {
		return h.app.logFile.Close()
	}
	return nil
}

func (h *rotatingFileHandler) Flush() error { return nil }

func (h *rotatingFileHandler) IsHandling(level slog.Level) bool {
	return h.app.cfg.hasLevel(level)
}

// setupLogger собирает *slog.Logger согласно cfg.
func (app *Application) setupLogger() (*slog.Logger, error) {
	logger := slog.New()

	if app.cfg.LogToDB {
		if err := app.setupDB(); err != nil {
			return nil, fmt.Errorf("настройка логгера БД: %w", err)
		}
		logger.AddHandler(&DBHandler{app: app})
	}

	if app.cfg.LogConsole {
		consoleHandler := handler.NewConsoleHandler(app.cfg.levels())
		consoleHandler.SetFormatter(&CustomFormatter{})
		logger.AddHandler(consoleHandler)
	} else if !app.cfg.LogToDB {
		// Ни консоль, ни БД не включены - логируем в файл с ротацией по дням.
		logger.AddHandler(newRotatingFileHandler(app))
	}

	return logger, nil
}

// New создаёт полностью настроенное Application. Значения по умолчанию:
// консоль включена, debug/info/warn включены, БД выключена. Передайте
// Option, чтобы изменить поведение.
//
//	app, err := v2.New(v2.WithDB("log/app.db"), v2.WithConsole(false))
//	if err != nil { ... }
//	defer app.Close()
func New(opts ...Option) (*Application, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	app := &Application{cfg: cfg}
	logger, err := app.setupLogger()
	if err != nil {
		return nil, err
	}
	app.Log = logger
	return app, nil
}

// Fatalf логирует сообщение с уровнем Error, корректно закрывает все
// ресурсы Application (БД, файл логов, фоновый писатель) через Close и
// затем завершает процесс с кодом 1.
//
// В отличие от app.Log.Fatalf (который сам зовёт os.Exit и тем самым
// пропускает все отложенные defer, включая defer app.Close()), этот метод
// сначала гарантированно освобождает ресурсы и только потом выходит -
// поэтому для аварийного завершения приложения стоит использовать именно
// app.Fatalf, а не app.Log.Fatalf.
func (app *Application) Fatalf(format string, args ...any) {
	app.Log.Errorf(format, args...)
	if err := app.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "logger: ошибка при закрытии ресурсов: %v\n", err)
	}
	os.Exit(1)
}

// Close останавливает фоновую горутину записи в БД, закрывает
// подготовленный запрос, соединение с БД и открытый файл логов.
// Вызывать один раз при завершении работы приложения.
func (app *Application) Close() error {
	var errs []error

	if app.dbCh != nil {
		close(app.dbCh)
		app.dbWG.Wait()
	}
	if app.stmt != nil {
		if err := app.stmt.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if app.db != nil {
		if err := app.db.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if app.logFile != nil {
		if err := app.logFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("закрытие Application: %v", errs)
	}
	return nil
}

// ---------------------------------------------------------------------
// Legacy API: сохранён для обратной совместимости с кодом, написанным
// под первую версию пакета (пакетные глобальные переменные + SetupApplication()
// без аргументов). Новый код должен использовать New(...Option) вместо этого.
//
// ВНИМАНИЕ: как и в оригинальной реализации, эти переменные читаются один раз
// в момент вызова SetupApplication() - изменение их "на лету" после старта
// приложения не влияет на уже созданный логгер. Так было и в исходной версии;
// если нужно несколько независимо настроенных логгеров или горячая
// переконфигурация, используйте New(...Option).
// ---------------------------------------------------------------------

var (
	// Deprecated: используйте WithConsole вместе с New.
	logConsole = true
	// Deprecated: используйте WithDB вместе с New.
	logToDB = false
	// Deprecated: используйте WithDebug вместе с New.
	IsDebugMode = true
	// Deprecated: используйте WithInfo вместе с New.
	IsInfoMode = true
	// Deprecated: используйте WithWarn вместе с New.
	IsWarnMode = true
	// Deprecated: используйте WithDB(path) вместе с New.
	DBPath string
)

// SetLogConsole устанавливает флаг логирования в консоль.
//
// Deprecated: используйте New(WithConsole(value)).
func SetLogConsole(value bool) { logConsole = value }

// SetLogToDB устанавливает флаг логирования в базу данных.
//
// Deprecated: используйте New(WithDB(path)).
func SetLogToDB(value bool) { logToDB = value }

// SetIsDebugMode устанавливает флаг режима отладки.
//
// Deprecated: используйте New(WithDebug(value)).
func SetIsDebugMode(value bool) { IsDebugMode = value }

// SetIsInfoMode устанавливает флаг информационного режима.
//
// Deprecated: используйте New(WithInfo(value)).
func SetIsInfoMode(value bool) { IsInfoMode = value }

// SetIsWarnMode устанавливает флаг режима предупреждений.
//
// Deprecated: используйте New(WithWarn(value)).
func SetIsWarnMode(value bool) { IsWarnMode = value }

// SetupApplication создаёт Application на основе пакетных глобальных
// переменных (SetLogConsole, SetLogToDB, IsDebugMode, ...), как в первой
// версии пакета.
//
// Поведение при ошибках намеренно повторяет оригинал:
//   - если не удалось настроить логирование в БД, ошибка печатается в stdout,
//     и приложение продолжает работу без логирования в БД;
//   - если не удалось создать/открыть файл для файлового логгера (используется,
//     когда отключены и консоль, и БД), функция паникует.
//
// Deprecated: используйте New(...Option), которая возвращает error вместо
// паники и не полагается на глобальное состояние пакета.
func SetupApplication() *Application {
	cfg := Config{
		LogConsole: logConsole,
		Debug:      IsDebugMode,
		Info:       IsInfoMode,
		Warn:       IsWarnMode,
		DBPath:     DBPath,
		LogDir:     "log",
	}

	app := &Application{cfg: cfg}
	logger := slog.New()

	if logToDB {
		if err := app.setupDB(); err != nil {
			fmt.Printf("Не удалось настроить логгер базы данных: %v\n", err)
		} else {
			app.cfg.LogToDB = true
			logger.AddHandler(&DBHandler{app: app})
		}
	}

	if cfg.LogConsole {
		consoleHandler := handler.NewConsoleHandler(cfg.levels())
		consoleHandler.SetFormatter(&CustomFormatter{})
		logger.AddHandler(consoleHandler)
	} else if !app.cfg.LogToDB {
		fh := newRotatingFileHandler(app)
		if err := fh.ensureFile(); err != nil {
			panic(fmt.Sprintf("Ошибка открытия файла логов: %v", err))
		}
		logger.AddHandler(fh)
	}

	app.Log = logger
	return app
}
