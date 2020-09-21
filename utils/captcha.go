package utils

import (
	"time"

	"github.com/go-redis/redis"
	"github.com/mojocn/base64Captcha"
)

var config = base64Captcha.ConfigCharacter{
	Height: 60,
	Width:  150,
	//const CaptchaModeNumber:数字,CaptchaModeAlphabet:字母,CaptchaModeArithmetic:算术,CaptchaModeNumberAlphabet:数字字母混合.
	Mode:               base64Captcha.CaptchaModeNumberAlphabet,
	ComplexOfNoiseText: base64Captcha.CaptchaComplexLower,
	ComplexOfNoiseDot:  base64Captcha.CaptchaComplexLower,
	IsUseSimpleFont:    false,
	IsShowHollowLine:   false,
	IsShowNoiseDot:     true,
	IsShowNoiseText:    true,
	IsShowSlimeLine:    false,
	IsShowSineLine:     false,
	CaptchaLen:         4,
}

var configWH = base64Captcha.ConfigCharacter{
	Height: 60,
	Width:  150,
	//const CaptchaModeNumber:数字,CaptchaModeAlphabet:字母,CaptchaModeArithmetic:算术,CaptchaModeNumberAlphabet:数字字母混合.
	Mode:               base64Captcha.CaptchaModeNumberAlphabet,
	ComplexOfNoiseText: base64Captcha.CaptchaComplexLower,
	ComplexOfNoiseDot:  base64Captcha.CaptchaComplexLower,
	IsUseSimpleFont:    false,
	IsShowHollowLine:   false,
	IsShowNoiseDot:     true,
	IsShowNoiseText:    true,
	IsShowSlimeLine:    false,
	IsShowSineLine:     false,
	CaptchaLen:         4,
}

func InitRedisCaptchaStore(client redis.Cmdable) {
	base64Captcha.SetCustomStore(&RedisStore{redisClient: client})
}

func CreateCaptchaWithWH(width, height int64) (id, data string) {
	var digitCap base64Captcha.CaptchaInterface

	configWH.Height = int(height)
	configWH.Width = int(width)
	id, digitCap = base64Captcha.GenerateCaptcha("", configWH)
	data = base64Captcha.CaptchaWriteToBase64Encoding(digitCap)

	return
}

func CreateCaptcha() (id, data string) {
	var digitCap base64Captcha.CaptchaInterface

	id, digitCap = base64Captcha.GenerateCaptcha("", config)
	data = base64Captcha.CaptchaWriteToBase64Encoding(digitCap)

	return
}

func VerifyCaptcha(idKey, verifyValue string) bool {
	return base64Captcha.VerifyCaptcha(idKey, verifyValue)
}

type RedisStore struct {
	redisClient redis.Cmdable
}

func (this_ *RedisStore) Set(id string, value string) {
	this_.redisClient.Set("captcha:"+id, value, time.Minute*5)
}

func (this_ *RedisStore) Get(id string, clear bool) string {
	str := this_.redisClient.Get("captcha:" + id).Val()
	if clear && str != "" {
		this_.redisClient.Del("captcha:" + id)
	}

	return str
}
