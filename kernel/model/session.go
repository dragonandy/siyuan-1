// SiYuan - Refactor your thinking
// Copyright (c) 2020-present, b3log.org
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package model

import (
	"image/color"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/88250/gulu"
	"github.com/gin-gonic/gin"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/util"
	"github.com/steambap/captcha"
)

func LogoutAuth(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	if "" == Conf.AccessAuthCode {
		ret.Code = -1
		ret.Msg = Conf.Language(86)
		ret.Data = map[string]interface{}{"closeTimeout": 5000}
		return
	}

	session := util.GetSession(c)
	util.RemoveWorkspaceSession(session)
	if err := session.Save(c); nil != err {
		logging.LogErrorf("saves session failed: " + err.Error())
		ret.Code = -1
		ret.Msg = "save session failed"
	}
}

func LoginAuth(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	var inputCaptcha string
	session := util.GetSession(c)
	workspaceSession := util.GetWorkspaceSession(session)
	if util.NeedCaptcha() {
		captchaArg := arg["captcha"]
		if nil == captchaArg {
			ret.Code = 1
			ret.Msg = Conf.Language(21)
			return
		}
		inputCaptcha = captchaArg.(string)
		if "" == inputCaptcha {
			ret.Code = 1
			ret.Msg = Conf.Language(21)
			return
		}

		if strings.ToLower(workspaceSession.Captcha) != strings.ToLower(inputCaptcha) {
			ret.Code = 1
			ret.Msg = Conf.Language(22)
			return
		}
	}

	authCode := arg["authCode"].(string)
	if Conf.AccessAuthCode != authCode {
		ret.Code = -1
		ret.Msg = Conf.Language(83)

		util.WrongAuthCount++
		workspaceSession.Captcha = gulu.Rand.String(7)
		if util.NeedCaptcha() {
			ret.Code = 1 // 需要渲染验证码
		}

		if err := session.Save(c); nil != err {
			logging.LogErrorf("save session failed: " + err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}
		return
	}

	workspaceSession.AccessAuthCode = authCode
	util.WrongAuthCount = 0
	workspaceSession.Captcha = gulu.Rand.String(7)
	if err := session.Save(c); nil != err {
		logging.LogErrorf("save session failed: " + err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
}

func GetCaptcha(c *gin.Context) {
	img, err := captcha.New(100, 26, func(options *captcha.Options) {
		options.CharPreset = "ABCDEFGHKLMNPQRSTUVWXYZ23456789"
		options.Noise = 0.5
		options.CurveNumber = 0
		options.BackgroundColor = color.White
	})
	if nil != err {
		logging.LogErrorf("generates captcha failed: " + err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	session := util.GetSession(c)
	workspaceSession := util.GetWorkspaceSession(session)
	workspaceSession.Captcha = img.Text
	if err = session.Save(c); nil != err {
		logging.LogErrorf("save session failed: " + err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	if err = img.WriteImage(c.Writer); nil != err {
		logging.LogErrorf("writes captcha image failed: " + err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

func CheckReadonly(c *gin.Context) {
	if util.ReadOnly {
		result := util.NewResult()
		result.Code = -1
		result.Msg = Conf.Language(34)
		result.Data = map[string]interface{}{"closeTimeout": 5000}
		c.JSON(http.StatusOK, result)
		c.Abort()
		return
	}
}

func CheckBasicAuth(c *gin.Context) {
	if "" == Conf.AccessAuthCode {
		localhost := isLocalhost(c)
		if localhost {
			c.Next()
		} else {
			abortWithUnauthorized(c)
		}
		return
	}

	// 通过 Cookies
	if certified := checkCookies(c); certified {
		c.Next()
		return
	}

	// 通过 API token
	if certified, ok := checkToken(c); ok {
		if certified {
			c.Next()
		} else {
			abortWithUnauthorized(c)
		}
		return
	}

	// 通过 HTTP Basic
	if certified, ok := checkBasic(c); ok && certified {
		c.Next()
	} else {
		c.Header("WWW-Authenticate", "Basic realm=\"Authorization Required\", charset=\"UTF-8\"")
		c.AbortWithStatus(http.StatusUnauthorized)
	}

	return
}

func CheckAuth(c *gin.Context) {
	//logging.LogInfof("check auth for [%s]", c.Request.RequestURI)
	localhost := isLocalhost(c)

	// 未设置访问授权码
	if "" == Conf.AccessAuthCode {
		// Authenticate requests with the Origin header other than 127.0.0.1 https://github.com/siyuan-note/siyuan/issues/9180
		if !localhost {
			c.JSON(http.StatusUnauthorized, map[string]interface{}{"code": -1, "msg": "Auth failed: for security reasons, please set [Access authorization code] when using non-127.0.0.1 access\n\n为安全起见，使用非 127.0.0.1 访问时请设置 [访问授权码]"})
			c.Abort()
			return
		}

		c.Next()
		return
	}

	// 放过 /appearance/
	if strings.HasPrefix(c.Request.RequestURI, "/appearance/") ||
		strings.HasPrefix(c.Request.RequestURI, "/stage/build/export/") ||
		strings.HasPrefix(c.Request.RequestURI, "/stage/build/fonts/") ||
		strings.HasPrefix(c.Request.RequestURI, "/stage/protyle/") {
		c.Next()
		return
	}

	// 放过来自本机的某些请求
	if localhost {
		if strings.HasPrefix(c.Request.RequestURI, "/assets/") {
			c.Next()
			return
		}
		if strings.HasPrefix(c.Request.RequestURI, "/api/system/exit") {
			c.Next()
			return
		}
	}

	// 通过 Cookies
	cookiesCertified := checkCookies(c)
	if cookiesCertified {
		c.Next()
		return
	}

	// 通过 API token
	if certified, ok := checkToken(c); ok {
		if certified {
			c.Next()
		} else {
			abortWithUnauthorized(c)
		}
		return
	}

	// 通过 HTTP Basic
	if certified, ok := checkBasic(c); ok {
		if certified {
			c.Next()
		} else {
			abortWithUnauthorized(c)
		}
		return
	}

	if "/check-auth" == c.Request.URL.Path { // 跳过访问授权页
		c.Next()
		return
	}

	if !cookiesCertified {
		userAgentHeader := c.GetHeader("User-Agent")
		if strings.HasPrefix(userAgentHeader, "SiYuan/") || strings.HasPrefix(userAgentHeader, "Mozilla/") {
			if "GET" != c.Request.Method {
				c.JSON(http.StatusUnauthorized, map[string]interface{}{"code": -1, "msg": Conf.Language(156)})
				c.Abort()
				return
			}

			location := url.URL{}
			queryParams := url.Values{}
			queryParams.Set("to", c.Request.URL.String())
			location.RawQuery = queryParams.Encode()
			location.Path = "/check-auth"

			c.Redirect(http.StatusFound, location.String())
			c.Abort()
			return
		}

		abortWithUnauthorized(c)
		return
	}

	c.Next()
}

func checkCookies(c *gin.Context) bool {
	session := util.GetSession(c)
	workspaceSession := util.GetWorkspaceSession(session)
	return workspaceSession.AccessAuthCode == Conf.AccessAuthCode
}

func checkToken(c *gin.Context) (certified, ok bool) {
	var token string
	if authHeader := c.GetHeader("Authorization"); "" != authHeader {
		// 通过 API token (header: Authorization)
		if strings.HasPrefix(authHeader, "Token ") {
			ok = true
			token = strings.TrimPrefix(authHeader, "Token ")
		}
	} else {
		// 通过 API token (query-params: token)
		token, ok = c.GetQuery("token")
	}

	certified = Conf.Api.Token == token
	return
}

func checkBasic(c *gin.Context) (certified, ok bool) {
	_, password, ok := c.Request.BasicAuth()
	certified = Conf.AccessAuthCode == password
	return
}

func isLocalhost(c *gin.Context) bool {
	if !util.IsLocalHost(c.Request.RemoteAddr) {
		return false
	}

	host := c.GetHeader("Host")
	if "" != host {
		if !util.IsLocalHost(host) {
			return false
		}
	}

	origin := c.GetHeader("Origin")
	if "" != origin {
		if !util.IsLocalOrigin(origin) && !strings.HasPrefix(origin, "chrome-extension://") {
			return false
		}
	}

	forwardedHost := c.GetHeader("X-Forwarded-Host")
	if "" != forwardedHost {
		if !util.IsLocalHost(forwardedHost) {
			return false
		}
	}

	return true
}

func abortWithUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, map[string]interface{}{"code": -1, "msg": "Auth failed"})
}

var timingAPIs = map[string]int{
	"/api/search/fullTextSearchBlock": 200, // Monitor the search performance and suggest solutions https://github.com/siyuan-note/siyuan/issues/7873
}

func Timing(c *gin.Context) {
	p := c.Request.URL.Path
	tip, ok := timingAPIs[p]
	if !ok {
		c.Next()
		return
	}

	timing := 15 * 1000
	if timingEnv := os.Getenv("SIYUAN_PERFORMANCE_TIMING"); "" != timingEnv {
		val, err := strconv.Atoi(timingEnv)
		if nil == err {
			timing = val
		}
	}

	now := time.Now().UnixMilli()
	c.Next()
	elapsed := int(time.Now().UnixMilli() - now)
	if timing < elapsed {
		logging.LogWarnf("[%s] elapsed [%dms]", c.Request.RequestURI, elapsed)
		util.PushMsg(Conf.Language(tip), 7000)
	}
}

func Recover(c *gin.Context) {
	defer func() {
		logging.Recover()
		c.Status(http.StatusInternalServerError)
	}()

	c.Next()
}
