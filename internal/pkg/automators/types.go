package automators

import (
	"time"

	"github.com/google/uuid"
)

type JobConfig struct {
	CronExpression string    `json:"cron_expression,omitempty"`
	UID            uuid.UUID `json:"uid,omitempty"`
	Task           Task      `json:"task,omitempty"`
}

type Task struct {
	URL              string        `json:"url"`
	Timeout          time.Duration `json:"task,omitempty"`
	AuthHeader       AuthHeader    `json:"auth_header,omitempty"`
	ExpectedResponse any           `json:"expected_response,omitempty"`
}

type AuthHeader struct {
	Scheme     string `json:"scheme,omitempty"`
	Parameters string `json:"parameters,omitempty"`
}
