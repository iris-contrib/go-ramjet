package ramjet

import (
	utils "github.com/Laisky/go-utils"
	"github.com/Laisky/zap"
	"github.com/pkg/errors"
)

var (
	Email = &EmailType{}
)

type EmailType struct {
	sender *utils.Mail
}

func (e *EmailType) Setup() {
	e.sender = utils.NewMail(utils.Settings.GetString("email.host"), utils.Settings.GetInt("email.port"))
	e.sender.Login(utils.Settings.GetString("email.username"), utils.Settings.GetString("email.password"))
}

func (e *EmailType) Send(to, toName, subject, content string) (err error) {
	if utils.Settings.GetBool("dry") {
		utils.Logger.Info("send email",
			zap.String("cnt", content),
			zap.String("subject", subject),
			zap.String("to", to))
		return nil
	}

	err = e.sender.Send(
		utils.Settings.GetString("email.sender"),
		to,
		utils.Settings.GetString("email.senderName"),
		toName,
		subject,
		content,
	)
	if err != nil {
		return errors.Wrap(err, "go-ramjet try to send email got error")
	}

	utils.Logger.Info("successed send email",
		zap.String("subject", subject),
		zap.String("to", to))
	return nil
}
