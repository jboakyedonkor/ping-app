package automators

import (
	"time"

	"github.com/google/uuid"
)

type JobConfig struct {
	CronExpression string
	UID            uuid.UUID
	Task           Task
}

type Task struct {
	URL              string
	Timeout          time.Duration
	AuthHeader       AuthHeader
	ExpectedResponse any
}

type AuthHeader struct {
	Scheme     string
	Parameters string
}
