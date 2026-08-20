package v2

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHasDBExtension(t *testing.T) {
	cases := map[string]bool{
		"logs.db":  true,
		"LOGS.DB":  true,
		"logs.DB":  true,
		"logs":     false,
		"logs.txt": false,
		"":         false,
	}
	for name, want := range cases {
		if got := hasDBExtension(name); got != want {
			t.Errorf("hasDBExtension(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestFieldLoggerFormat(t *testing.T) {
	fl := &FieldLogger{fields: map[string]any{
		"user_id": 42,
		"chat_id": 7,
		"action":  "login",
	}}

	got := fl.format("сообщение")
	want := "сообщение | action=login chat_id=7 user_id=42"
	if got != want {
		t.Errorf("format() = %q, want %q (ключи должны быть отсортированы)", got, want)
	}

	// Без полей формат не должен ничего добавлять к сообщению.
	empty := &FieldLogger{}
	if got := empty.format("сообщение"); got != "сообщение" {
		t.Errorf("format() без полей = %q, want %q", got, "сообщение")
	}
}

func TestContextWithFields(t *testing.T) {
	ctx := context.Background()
	if f := fieldsFromContext(ctx); f != nil {
		t.Fatalf("fieldsFromContext(background) = %v, want nil", f)
	}

	ctx = ContextWithFields(ctx, map[string]any{"chat_id": 1, "user": "alice"})
	f := fieldsFromContext(ctx)
	if f["chat_id"] != 1 || f["user"] != "alice" {
		t.Fatalf("fieldsFromContext = %v, unexpected content", f)
	}

	// Повторный вызов должен смержить поля, а не затереть их полностью.
	ctx = ContextWithFields(ctx, map[string]any{"user": "bob", "extra": true})
	f = fieldsFromContext(ctx)
	if f["chat_id"] != 1 {
		t.Errorf("chat_id потерян после повторного ContextWithFields: %v", f)
	}
	if f["user"] != "bob" {
		t.Errorf("user не переопределён: %v", f)
	}
	if f["extra"] != true {
		t.Errorf("extra не добавлен: %v", f)
	}
}

func TestNew_FileLoggerWithRotation(t *testing.T) {
	dir := t.TempDir()

	app, err := New(WithConsole(false), WithLogDir(dir))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	app.Log.Info("тестовое сообщение")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("не удалось прочитать директорию логов: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ожидался ровно один файл лога, найдено %d", len(entries))
	}

	today := time.Now().Format("02-01-2006")
	wantName := "error-" + today + ".log"
	if entries[0].Name() != wantName {
		t.Errorf("имя файла = %q, want %q", entries[0].Name(), wantName)
	}

	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("не удалось прочитать файл лога: %v", err)
	}
	if !strings.Contains(string(data), "тестовое сообщение") {
		t.Errorf("файл лога не содержит ожидаемое сообщение: %s", data)
	}
}

func TestNew_DBLoggerBatchWrite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test-logs.db")

	app, err := New(WithConsole(false), WithDB(dbPath), WithDebug(false))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	app.Log.Info("info сообщение")
	app.Log.Error("error сообщение")
	app.Log.Debug("debug сообщение") // Debug выключен через WithDebug(false) - не должен попасть в БД

	app.Flush() // дождаться, пока фоновый писатель обработает очередь

	rows, err := app.db.Query(`SELECT level, message FROM logs ORDER BY id`)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	var got []struct{ level, message string }
	for rows.Next() {
		var level, message string
		if err := rows.Scan(&level, &message); err != nil {
			t.Fatalf("scan error: %v", err)
		}
		got = append(got, struct{ level, message string }{level, message})
	}

	if len(got) != 2 {
		t.Fatalf("ожидалось 2 записи в БД (Debug должен быть отфильтрован), получено %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].message, "info сообщение") {
		t.Errorf("первая запись = %+v, ожидалось info-сообщение", got[0])
	}
	if !strings.Contains(got[1].message, "error сообщение") {
		t.Errorf("вторая запись = %+v, ожидалось error-сообщение", got[1])
	}
}

func TestNew_DBLoggerWithFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test-logs.db")

	app, err := New(WithConsole(false), WithDB(dbPath))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	app.WithFields(map[string]any{"chat_id": 123}).Info("сообщение получено")
	app.Flush()

	var message string
	err = app.db.QueryRow(`SELECT message FROM logs LIMIT 1`).Scan(&message)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if !strings.Contains(message, "chat_id=123") {
		t.Errorf("сообщение = %q, ожидалось наличие chat_id=123", message)
	}
}

func TestDBRetention(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test-logs.db")

	app, err := New(WithConsole(false), WithDB(dbPath), WithDBRetention(10*time.Millisecond))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	app.Log.Info("устареет через 10ms")
	app.Flush()

	var countBefore int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM logs`).Scan(&countBefore); err != nil {
		t.Fatalf("query error: %v", err)
	}
	if countBefore != 1 {
		t.Fatalf("countBefore = %d, want 1", countBefore)
	}

	time.Sleep(30 * time.Millisecond)
	app.pruneOldLogs()

	var countAfter int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM logs`).Scan(&countAfter); err != nil {
		t.Fatalf("query error: %v", err)
	}
	if countAfter != 0 {
		t.Errorf("countAfter = %d, want 0 (запись должна была устареть)", countAfter)
	}
}

func TestCloseWithTimeout_Succeeds(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test-logs.db")

	app, err := New(WithConsole(false), WithDB(dbPath))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	app.Log.Info("сообщение перед закрытием")

	if err := app.CloseWithTimeout(2 * time.Second); err != nil {
		t.Fatalf("CloseWithTimeout() error = %v", err)
	}
}

func TestNew_InvalidDBPath(t *testing.T) {
	// Директория с именем как у "файла" в пути делает MkdirAll невозможным
	// на большинстве ОС только в специфических случаях; вместо этого
	// проверяем, что New корректно возвращает ошибку, если директория для
	// БД не может быть создана из-за конфликта имени файла/директории.
	tmp := t.TempDir()
	blockerFile := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blockerFile, []byte("x"), 0644); err != nil {
		t.Fatalf("не удалось создать файл-блокер: %v", err)
	}

	// Путь к БД лежит "внутри" обычного файла - MkdirAll должен вернуть ошибку.
	dbPath := filepath.Join(blockerFile, "sub", "logs.db")

	app, err := New(WithConsole(false), WithDB(dbPath))
	if err == nil {
		t.Fatalf("ожидалась ошибка New(), но её не было; app = %+v", app)
	}
	if app != nil {
		t.Errorf("при ошибке New() должен возвращать nil *Application")
	}
}
