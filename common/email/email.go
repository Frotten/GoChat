package email

import (
	"GopherAI/config"
	"fmt"

	"gopkg.in/gomail.v2"
)

const (
	CodeMsg     = "GoAI验证码如下(验证码仅限于2分钟有效): "
	UserNameMsg = "GoAI的账号如下，请保留好，后续可以用账号/邮箱登录 "
)

func SendCaptcha(email, code, msg string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", config.GetConfig().EmailConfig.Email)
	m.SetHeader("To", email)
	m.SetHeader("Subject", "来自GoAI的信息")
	m.SetBody("text/plain", msg+" "+code)
	// 配置 SMTP 服务器和授权码,587：是 SMTP 的明文/STARTTLS 端口号
	d := gomail.NewDialer("smtp.qq.com", 587, config.GetConfig().EmailConfig.Email, config.GetConfig().EmailConfig.Authcode)
	if err := d.DialAndSend(m); err != nil {
		fmt.Printf("DialAndSend err %v:\n", err)
		return err
	}
	fmt.Printf("send mail success\n")
	return nil
}
