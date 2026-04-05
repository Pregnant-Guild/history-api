package constants

type TaskType string

const (
	TaskTypeSendEmailOTP TaskType = "SEND_EMAIL_OTP"
	TaskTypeDeleteMedia  TaskType = "DELETE_MEDIA"
)

func (t TaskType) String() string {
	return string(t)
}
