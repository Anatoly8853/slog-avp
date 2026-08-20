package v2

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// FieldLogger - логгер с привязанным набором key-value полей. Получается
// через Application.WithFields и позволяет писать структурированные логи:
//
//	app.WithFields(map[string]any{
//	    "user_id": 123,
//	    "chat_id": update.Message.Chat.ID,
//	}).Info("сообщение обработано")
//
// Поля сериализуются в конец текста сообщения в виде "key=value" пар с
// отсортированными по алфавиту ключами (порядок стабилен между вызовами).
// Такой подход не зависит от внутреннего формата записи gookit/slog и
// одинаково работает для консоли, файла и БД - поля просто становятся
// частью строки message, которую видят все обработчики.
type FieldLogger struct {
	app    *Application
	fields map[string]any
}

// WithFields возвращает FieldLogger с привязанными полями для последующих
// вызовов уровня логирования. Исходный app.Log при этом не меняется -
// можно продолжать пользоваться app.Log.Info(...) напрямую там, где
// структурированные поля не нужны.
func (app *Application) WithFields(fields map[string]any) *FieldLogger {
	return &FieldLogger{app: app, fields: fields}
}

func (f *FieldLogger) format(msg string) string {
	if len(f.fields) == 0 {
		return msg
	}

	keys := make([]string, 0, len(f.fields))
	for k := range f.fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, f.fields[k]))
	}

	return msg + " | " + strings.Join(parts, " ")
}

func (f *FieldLogger) Debug(msg string) { f.app.Log.Debug(f.format(msg)) }
func (f *FieldLogger) Info(msg string)  { f.app.Log.Info(f.format(msg)) }
func (f *FieldLogger) Warn(msg string)  { f.app.Log.Warn(f.format(msg)) }
func (f *FieldLogger) Error(msg string) { f.app.Log.Error(f.format(msg)) }

// Debugf, Infof, Warnf, Errorf - варианты с форматированием сообщения,
// аналогичные одноимённым методам gookit/slog.Logger.
func (f *FieldLogger) Debugf(format string, args ...any) {
	f.app.Log.Debug(f.format(fmt.Sprintf(format, args...)))
}
func (f *FieldLogger) Infof(format string, args ...any) {
	f.app.Log.Info(f.format(fmt.Sprintf(format, args...)))
}
func (f *FieldLogger) Warnf(format string, args ...any) {
	f.app.Log.Warn(f.format(fmt.Sprintf(format, args...)))
}
func (f *FieldLogger) Errorf(format string, args ...any) {
	f.app.Log.Error(f.format(fmt.Sprintf(format, args...)))
}

// ---------------------------------------------------------------------
// Поддержка context.Context: позволяет один раз привязать поля (например,
// trace_id/chat_id/request_id) к контексту в начале обработки запроса и
// автоматически прикладывать их ко всем логам ниже по стеку вызовов, не
// прокидывая map явно через каждую функцию.
// ---------------------------------------------------------------------

type ctxFieldsKey struct{}

// ContextWithFields возвращает новый context.Context с добавленными
// полями. Поля из родительского контекста (если были) сохраняются;
// одинаковые ключи в fields их переопределяют. Использование:
//
//	ctx := v2.ContextWithFields(context.Background(), map[string]any{
//	    "chat_id": update.Message.Chat.ID,
//	    "user":    update.Message.From.UserName,
//	})
//	app.InfoContext(ctx, "сообщение получено")
func ContextWithFields(ctx context.Context, fields map[string]any) context.Context {
	merged := mergeFields(fieldsFromContext(ctx), fields)
	return context.WithValue(ctx, ctxFieldsKey{}, merged)
}

func fieldsFromContext(ctx context.Context) map[string]any {
	if ctx == nil {
		return nil
	}
	if f, ok := ctx.Value(ctxFieldsKey{}).(map[string]any); ok {
		return f
	}
	return nil
}

func mergeFields(base, extra map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged
}

// WithContext возвращает FieldLogger с полями, привязанными к ctx через
// ContextWithFields. Если полей в контексте нет, ведёт себя как обычный
// логгер без структурированных полей.
func (app *Application) WithContext(ctx context.Context) *FieldLogger {
	return app.WithFields(fieldsFromContext(ctx))
}

// DebugContext, InfoContext, WarnContext, ErrorContext - удобные обёртки
// над WithContext для одиночного вызова без промежуточной переменной:
//
//	app.InfoContext(ctx, "сообщение обработано")
func (app *Application) DebugContext(ctx context.Context, msg string) {
	app.WithContext(ctx).Debug(msg)
}
func (app *Application) InfoContext(ctx context.Context, msg string) { app.WithContext(ctx).Info(msg) }
func (app *Application) WarnContext(ctx context.Context, msg string) { app.WithContext(ctx).Warn(msg) }
func (app *Application) ErrorContext(ctx context.Context, msg string) {
	app.WithContext(ctx).Error(msg)
}
