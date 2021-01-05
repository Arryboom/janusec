/*
 * @Copyright Reserved By Janusec (https://www.janusec.com/).
 * @Author: U2
 * @Date: 2018-07-14 16:38:10
 * @Last Modified: U2, 2018-07-14 16:38:10
 */

package gateway

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	//"net/http/httputil"
	"strings"

	"janusec/backend"
	"janusec/firewall"
	"janusec/models"
	"janusec/utils"
)

func rewriteResponse(resp *http.Response) (err error) {
	r := resp.Request
	//app := backend.GetApplicationByDomain(r.Host)
	app := backend.GetApplicationByDomain(r.Host)
	///r.host
	locationURL, err := resp.Location()
	utils.DebugPrintln("33_gw_res_locationURL",locationURL)
	if locationURL != nil {
		port := locationURL.Port()
		utils.DebugPrintln("35_gw_res_port",port)
		if (port != "80") && (port != "443") {
			host := locationURL.Hostname()
			utils.DebugPrintln("37_gw_res_host",host)
			//app := backend.GetApplicationByDomain(host)
			if app != nil {
				//newLocation := strings.Replace(locationURL.String(), host+":"+port, host, -1)
				newLocation :=locationURL.String()
				userScheme := "http"
				if resp.Request.TLS != nil {
					userScheme = "https"
				}
				newLocation = strings.Replace(newLocation, locationURL.Scheme, userScheme, 1)
				utils.DebugPrintln("46_gw_res",newLocation)
				resp.Header.Set("Location", newLocation)
			}
		}
	}

	// Hide X-Powered-By
	xPoweredBy := resp.Header.Get("X-Powered-By")
	if xPoweredBy != "" {
		resp.Header.Set("X-Powered-By", "Janusec")
	}

	srcIP := GetClientIP(r, app)
	if app.WAFEnabled {
		if isHit, policy := firewall.IsResponseHitPolicy(resp, app.ID); isHit {
			switch policy.Action {
			case models.Action_Block_100:
				vulnName, _ := firewall.VulnMap.Load(policy.VulnID)
				hitInfo := &models.HitInfo{TypeID: 2, PolicyID: policy.ID, VulnName: vulnName.(string)}
				go firewall.LogGroupHitRequest(r, app.ID, srcIP, policy)
				blockContent := GenerateBlockConcent(hitInfo)
				body := ioutil.NopCloser(bytes.NewReader(blockContent))
				resp.Body = body
				resp.ContentLength = int64(len(blockContent))
				resp.StatusCode = 403
				return nil
			case models.Action_BypassAndLog_200:
				go firewall.LogGroupHitRequest(r, app.ID, srcIP, policy)
			case models.Action_CAPTCHA_300:
				clientID := GenClientID(r, app.ID, srcIP)
				targetURL := r.URL.Path
				if len(r.URL.RawQuery) > 0 {
					targetURL += "?" + r.URL.RawQuery
				}
				hitInfo := &models.HitInfo{TypeID: 2,
					PolicyID: policy.ID, VulnName: "Group Policy Hit",
					Action: policy.Action, ClientID: clientID,
					TargetURL: targetURL, BlockTime: time.Now().Unix()}
				captchaHitInfo.Store(clientID, hitInfo)
				captchaURL := CaptchaEntrance + "?id=" + clientID
				resp.Header.Set("Location", captchaURL)
				resp.ContentLength = 0
				//http.Redirect(w, r, captchaURL, http.StatusTemporaryRedirect)
				return
			default:
				// models.Action_Pass_400 do nothing
			}
		}
	}

	// HSTS
	if (app.HSTSEnabled == true) && (r.TLS != nil) {
		resp.Header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}

	// CSP Content-Security-Policy, 0.9.11+
	if app.CSPEnabled {
		resp.Header.Set("Content-Security-Policy", app.CSP)
	}

	// if client http and backend https, remove "; Secure" and replace https by http
	if (r.TLS == nil) && (app.InternalScheme == "https") {
		cookies := resp.Cookies()
		for _, cookie := range cookies {
			re := regexp.MustCompile(`;\s*Secure`)
			cookieStr := re.ReplaceAllLiteralString(cookie.Raw, "")
			resp.Header.Set("Set-Cookie", cookieStr)
		}
		origin := resp.Header.Get("Access-Control-Allow-Origin")
		if len(origin) > 0 {
			resp.Header.Set("Access-Control-Allow-Origin", strings.Replace(origin, "https", "http", 1))
		}
		csp := resp.Header.Get("Content-Security-Policy")
		if len(csp) > 0 {
			resp.Header.Set("Content-Security-Policy", strings.Replace(origin, "https", "http", -1))
		}
	}

	// Static Cache
	if resp.StatusCode == http.StatusOK && firewall.IsStaticResource(r) {
		if resp.ContentLength < 0 || resp.ContentLength > 1024*1024*10 {
			// Not cache big files which size bigger than 10MB or unkonwn
			return nil
		}
		staticRoot := fmt.Sprintf("./static/cdncache/%d", app.ID)
		targetFile := staticRoot + r.URL.Path
		cacheFilePath := filepath.Dir(targetFile)
		bodyBuf, _ := ioutil.ReadAll(resp.Body)
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBuf))
		err := os.MkdirAll(cacheFilePath, 0666)
		if err != nil {
			utils.DebugPrintln("Cache Path Error", err)
		}
		contentEncoding := resp.Header.Get("Content-Encoding")
		switch contentEncoding {
		case "gzip":
			reader, err := gzip.NewReader(bytes.NewBuffer(bodyBuf))
			defer reader.Close()
			decompressedBodyBuf, err := ioutil.ReadAll(reader)
			if err != nil {
				utils.DebugPrintln("Gzip decompress Error", err)
			}
			err = ioutil.WriteFile(targetFile, decompressedBodyBuf, 0600)
		default:
			err = ioutil.WriteFile(targetFile, bodyBuf, 0600)
		}
		if err != nil {
			utils.DebugPrintln("Cache File Error", targetFile, err)
		}
		lastModified, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
		if err != nil {
			//utils.DebugPrintln("Cache File Parse Last-Modified", targetFile, err)
			return nil
		}
		err = os.Chtimes(targetFile, time.Now(), lastModified)
		if err != nil {
			utils.DebugPrintln("Cache File Chtimes", targetFile, err)
		}
	}
	//body, err := httputil.DumpResponse(resp, true)
	//fmt.Println("Dump Response:")
	//fmt.Println(string(body))
	return nil
}
