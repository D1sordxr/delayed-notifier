package vo

const (
	NotificationsQueue = "notifications.queue.process"
	WaitQueue          = "wait.queue.delay"
	RetryQueue         = "retry.queue.delay"

	NotificationsExchange = "notifications.exchange"
	WaitExchange          = "wait.exchange"
	RetryExchange         = "retry.exchange"

	Direct = "direct"
)
