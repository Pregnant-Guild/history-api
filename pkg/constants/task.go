package constants

type TaskType string

const (
	TaskTypeSendEmailOTP          TaskType = "SEND_EMAIL_OTP"
	TaskTypeNotifyHistorianReview TaskType = "NOTIFY_HISTORIAN_REVIEW"
	TaskTypeDeleteMedia           TaskType = "DELETE_MEDIA"
	TaskTypeBulkDeleteMedia       TaskType = "BULK_DELETE_MEDIA"
)

func (t TaskType) String() string {
	return string(t)
}
