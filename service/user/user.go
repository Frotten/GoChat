package user

import (
	"GopherAI/common/code"
	myemail "GopherAI/common/email"
	myredis "GopherAI/common/redis"
	"GopherAI/dao/user"
	"GopherAI/model"
	"GopherAI/utils"
	"GopherAI/utils/myjwt"
	"log"
	"os"
	"strings"
)

func Login(username, password string) (string, code.Code) {
	var userInformation *model.User
	var ok bool
	if ok, userInformation = user.IsExistUser(username); !ok {
		return "", code.CodeUserNotExist
	}
	if userInformation.Password != utils.MD5(password) {
		return "", code.CodeInvalidPassword
	}
	token, err := myjwt.GenerateToken(userInformation.ID, userInformation.Username)
	if err != nil {
		return "", code.CodeServerBusy
	}
	return token, code.CodeSuccess
}

func Register(email, password, captcha string) (string, code.Code) {
	var ok bool
	var userInformation *model.User
	if ok, _ := user.IsExistUser(email); ok {
		return "", code.CodeUserExist
	}
	if ok, _ := myredis.CheckCaptchaForEmail(email, captcha); !ok {
		return "", code.CodeInvalidCaptcha
	}
	username := utils.GetRandomNumbers(11)
	if userInformation, ok = user.Register(username, email, password); !ok {
		return "", code.CodeServerBusy
	}
	if err := myemail.SendCaptcha(email, username, user.UserNameMsg); err != nil {
		return "", code.CodeServerBusy
	}
	token, err := myjwt.GenerateToken(userInformation.ID, userInformation.Username)
	if err != nil {
		return "", code.CodeServerBusy
	}
	return token, code.CodeSuccess
}

func SendCaptcha(email_ string) code.Code {
	sendCode := utils.GetRandomNumbers(6)
	if err := myredis.SetCaptchaForEmail(email_, sendCode); err != nil {
		log.Printf("[captcha] Redis 写入失败: %v", err)
		return code.CodeServerBusy
	}

	if strings.EqualFold(os.Getenv("CAPTCHA_DEV_MODE"), "true") {
		log.Printf("[captcha] DEV 模式 邮箱=%s 验证码=%s（2分钟内有效）", email_, sendCode)
		return code.CodeSuccess
	}

	if err := myemail.SendCaptcha(email_, sendCode, myemail.CodeMsg); err != nil {
		log.Printf("[captcha] 邮件发送失败: %v", err)
		return code.CodeServerBusy
	}
	return code.CodeSuccess
}
