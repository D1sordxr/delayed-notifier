package vo

import "fmt"

const (
	NotifyStorageKey  = "notification"
	DeletedStorageKey = "deleted"
	DeletedValue      = "1"
)

func WithStorageKeyPrefix(v string) string {
	return fmt.Sprintf("%s:%s", NotifyStorageKey, v)
}

func WithStorageKeyPrefixDeleted(v string) string {
	return fmt.Sprintf("%s:%s:%s", NotifyStorageKey, DeletedStorageKey, v)
}
