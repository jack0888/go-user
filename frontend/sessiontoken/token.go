package sessiontoken

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
)

const (
	AuthTypeEmailPassword = "email_password" // 邮箱+密码
	AuthTypeEmailCaptcha  = "email_captcha"  // 邮箱+验证码, 验证码推送到邮箱
	AuthTypePhonePassword = "phone_password" // 手机+密码
	AuthTypePhoneCaptcha  = "phone_captcha"  // 手机+验证码, 验证码短信推送给手机
	AuthTypeOAuthQQ       = "oauth_qq"       // QQ oauth
	AuthTypeOAuthWechat   = "oauth_wechat"   // 微信 oauth
	AuthTypeOAuthWeibo    = "oauth_weibo"    // 微博 oauth
)

// 该结构体序列化后, 在客户端和服务器之间传递, 作为认证标识
type SessionToken struct {
	SessionId string `json:"sid"`

	// 以下为 token 相关部分
	TokenId       string `json:"token_id"`      // token 的标识, 每次刷新 token 改变此值
	Authenticated bool   `json:"authenticated"` // 是否是认证后的 token, 否则为临时 token
	UserId        int64  `json:"user_id"`       // token 的拥有者
	AuthType      string `json:"auth_type"`     // token 的认证类型
	PasswordTag   string `json:"password_tag"`  // 认证时的 password_tag, 对于 AuthType 是 AuthTypeEmailPassword, AuthTypePhonePassword 时有效
	ExpAccess     int64  `json:"exp_access"`    // 该 token 的过期时间
	ExpRefresh    int64  `json:"exp_refresh"`   // 通过该 token 换取新的 token 的截至时间, 固定值, 不会变化
}

// trim(url_base64(json(token))) + "." + hex(hmac-sha256(base64_str))
func (token *SessionToken) Encode(securityKey []byte) ([]byte, error) {
	const signatureLen = 64 // hmac-sha256

	jsonBytes, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}

	base64BytesLen := base64.URLEncoding.EncodedLen(len(jsonBytes))
	buf := make([]byte, base64BytesLen+1+signatureLen)

	base64Bytes := buf[:base64BytesLen]
	base64.URLEncoding.Encode(base64Bytes, jsonBytes)
	base64Bytes = base64Trim(base64Bytes)

	base64BytesLen = len(base64Bytes)
	buf[base64BytesLen] = '.'

	Hash := hmac.New(sha256.New, securityKey)
	Hash.Write(base64Bytes)
	hex.Encode(buf[base64BytesLen+1:], Hash.Sum(nil))

	return buf[:base64BytesLen+1+signatureLen], nil
}

var tokenBytesSplitSep = []byte{'.'}

// trim(url_base64(json(token))) + "." + hex(hmac-sha256(base64_str))
func (token *SessionToken) Decode(tokenBytes []byte, securityKey []byte) error {
	const signatureLen = 64 // hmac-sha256

	bytesArray := bytes.Split(tokenBytes, tokenBytesSplitSep)
	if len(bytesArray) < 2 {
		return errors.New("invalid token bytes")
	}
	if len(bytesArray[1]) != signatureLen {
		return errors.New("invalid token bytes, signature mismatch")
	}

	// 验证签名
	Hash := hmac.New(sha256.New, securityKey)
	Hash.Write(bytesArray[0])
	Signatrue := make([]byte, signatureLen)
	hex.Encode(Signatrue, Hash.Sum(nil))

	if !bytes.Equal(Signatrue, bytesArray[1]) {
		return errors.New("invalid token bytes, signature mismatch")
	}

	// 解码
	temp := Signatrue[:4]                       // Signatrue 不再使用, 利用其空间
	copy(temp, tokenBytes[len(bytesArray[0]):]) // 保护 tokenBytes

	base64Bytes := base64Pad(bytesArray[0])
	buf := make([]byte, base64.URLEncoding.DecodedLen(len(base64Bytes)))
	n, err := base64.URLEncoding.Decode(buf, base64Bytes)
	copy(tokenBytes[len(bytesArray[0]):], temp) // 恢复 tokenBytes
	if err != nil {
		return err
	}

	return json.Unmarshal(buf[:n], token)
}
