package logger

func WithFields(fields ...any) func(...any) []any {
	return func(newFields ...any) []any {
		return append(fields, newFields...)
	}
}
