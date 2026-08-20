package main

import (
	"context"
	"fmt"
	"os"
	"time"

	slogavp "github.com/Anatoly8853/slog-avp/v2"
)

func main() {
	// Настраиваем логгер через новый API: консоль + БД одновременно,
	// с автоматической очисткой БД от записей старше 30 дней.
	app, err := slogavp.New(
		slogavp.WithConsole(true),
		slogavp.WithDB("log/logs.db"),
		slogavp.WithDebug(true),
		slogavp.WithDBRetention(30*24*time.Hour),
		slogavp.WithDBRetentionCheckInterval(time.Hour),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "не удалось настроить логгер: %v\n", err)
		os.Exit(1)
	}
	// CloseWithTimeout вместо Close: если БД зависнет при остановке,
	// приложение всё равно завершится максимум за 5 секунд.
	defer func(app *slogavp.Application, timeout time.Duration) {
		err := app.CloseWithTimeout(timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "не удалось настроить логгер: %v\n", err)
		}
	}(app, 5*time.Second)

	app.Log.Info("Приложение запущено")
	app.Log.Debug("Отладочное сообщение")

	// Структурированные поля: попадают в сообщение как key=value,
	// одинаково видны в консоли, файле и БД.
	app.WithFields(map[string]any{
		"version": "2.1.0",
		"pid":     os.Getpid(),
	}).Info("конфигурация загружена")

	// Привязываем поля к context.Context один раз - дальше по стеку вызовов
	// достаточно прокидывать только ctx, без явной передачи map.
	ctx := slogavp.ContextWithFields(context.Background(), map[string]any{
		"request_id": "req-001",
	})

	if err := doSomething(ctx, app); err != nil {
		// app.Fatalf сам логирует ошибку, корректно закрывает ресурсы
		// (в отличие от app.Log.Fatalf, который пропускает defer) и
		// завершает процесс.
		app.Fatalf("doSomething завершился с ошибкой: %v", err)
	}

	// Flush гарантирует, что все ранее отправленные в БД записи уже
	// записаны - полезно перед действиями, которые от этого зависят.
	app.Flush()
	app.Log.Info("Работа завершена штатно")
}

func doSomething(ctx context.Context, app *slogavp.Application) error {
	// Добавляем ещё одно поле поверх уже привязанных к ctx (request_id
	// сохранится, chat_id добавится).
	ctx = slogavp.ContextWithFields(ctx, map[string]any{
		"chat_id": 123456,
	})

	app.InfoContext(ctx, "выполняется doSomething")

	if err := doRiskyStep(); err != nil {
		app.ErrorContext(ctx, "шаг завершился с ошибкой")
		return fmt.Errorf("doRiskyStep: %w", err)
	}

	app.InfoContext(ctx, "doSomething выполнен успешно")
	return nil
}

func doRiskyStep() error {
	// Здесь могла бы быть реальная операция (запрос к API, работа с БД и т.п.).
	// Чтобы увидеть путь с app.Fatalf, замените return на:
	//   return fmt.Errorf("что-то пошло не так")
	return nil
}
