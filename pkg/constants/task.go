package constants

type TaskType string

const (
	TaskTypeSendEmailOTP TaskType = "SEND_EMAIL_OTP"
) 

func (t TaskType) String() string {
	return string(t)
}