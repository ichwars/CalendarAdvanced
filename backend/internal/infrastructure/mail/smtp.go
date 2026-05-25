package mail

import "calendaradvanced/internal/infrastructure/config"

type Sender struct {
	Config config.SMTPConfig
}

func (s Sender) Enabled() bool {
	return s.Config.Host != "" && s.Config.From != ""
}
