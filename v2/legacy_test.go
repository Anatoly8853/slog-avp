package v2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/slog"
)

// withLegacyGlobals сохраняет текущее значение legacy-глобалов, выполняет f
// и восстанавливает исходные значения после теста - чтобы тесты legacy-API
// не влияли на порядок выполнения остальных тестов пакета.
func withLegacyGlobals(t *testing.T, f func()) {
	t.Helper()

	origConsole := logConsole
	origToDB := logToDB
	origDebug := IsDebugMode
	origInfo := IsInfoMode
	origWarn := IsWarnMode
	origDBPath := DBPath

	t.Cleanup(func() {
		logConsole = origConsole
		logToDB = origToDB
		IsDebugMode = origDebug
		IsInfoMode = origInfo
		IsWarnMode = origWarn
		DBPath = origDBPath
	})

	f()
}

func TestLegacy_SetupApplication_FileLogger(t *testing.T) {
	dir := t.TempDir()

	withLegacyGlobals(t, func() {
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd: %v", err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("Chdir: %v", err)
		}
		defer os.Chdir(wd)

		SetLogConsole(false)
		SetLogToDB(false)

		app := SetupApplication()
		defer func(app *Application) {
			err := app.Close()
			if err != nil {

			}
		}(app)

		app.Log.Info("legacy сообщение")

		entries, err := os.ReadDir(filepath.Join(dir, "log"))
		if err != nil {
			t.Fatalf("не удалось прочитать директорию log: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("ожидался один файл лога, найдено %d", len(entries))
		}

		data, err := os.ReadFile(filepath.Join(dir, "log", entries[0].Name()))
		if err != nil {
			t.Fatalf("не удалось прочитать файл лога: %v", err)
		}
		if !strings.Contains(string(data), "legacy сообщение") {
			t.Errorf("файл лога не содержит ожидаемое сообщение: %s", data)
		}
	})
}

func TestLegacy_SetupApplication_DB(t *testing.T) {
	dir := t.TempDir()

	withLegacyGlobals(t, func() {
		SetLogConsole(false)
		SetLogToDB(true)
		DBPath = filepath.Join(dir, "legacy-logs.db")

		app := SetupApplication()
		defer func(app *Application) {
			err := app.Close()
			if err != nil {

			}
		}(app)

		app.Log.Info("legacy db сообщение")
		app.Flush()

		var count int
		if err := app.db.QueryRow(`SELECT COUNT(*) FROM logs`).Scan(&count); err != nil {
			t.Fatalf("query error: %v", err)
		}
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})
}

func TestLegacy_SetIsDebugMode_AffectsLevels(t *testing.T) {
	withLegacyGlobals(t, func() {
		SetIsDebugMode(false)
		SetIsInfoMode(true)
		SetIsWarnMode(true)

		cfg := Config{Debug: IsDebugMode, Info: IsInfoMode, Warn: IsWarnMode}
		if cfg.hasLevel(slog.DebugLevel) {
			t.Error("Debug должен быть выключен после SetIsDebugMode(false)")
		}
		if !cfg.hasLevel(slog.InfoLevel) {
			t.Error("Info должен быть включён")
		}
	})
}
