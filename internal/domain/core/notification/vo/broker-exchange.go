package vo

const (
	NotificationsQueue = "notifications.queue.process"
	WaitQueue          = "wait.queue.delay"
	RetryQueue         = "retry.queue.delay"

	WaitExchange          = "wait.exchange"
	RetryExchange         = "retry.exchange"
	NotificationsExchange = "notifications.exchange"

	Direct = "direct"
)
