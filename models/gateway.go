/*
 * @Copyright Reserved By Janusec (https://www.janusec.com/).
 * @Author: U2
 * @Date: 2018-07-14 16:39:03
 * @Last Modified: U2, 2018-07-14 16:39:03
 */

package models

type HitInfo struct {
	TypeID    int64 // 1: CCPolicy  2:GroupPolicy
	PolicyID  int64
	VulnName  string
	Action    PolicyAction
	ClientID  string // for CC/Attack Client ID
	TargetURL string // for CAPTCHA redirect
	BlockTime int64
}

type CaptchaContext struct {
	CaptchaId string
	ClientID  string
}

type OAuthState struct {
	CallbackURL string
	UserID      string
	AccessToken string
}

// AccessStat record access statistics
type AccessStat struct {
	AppID      int64  `json:"app_id"`
	URLPath    string `json:"url_path"`
	StatDate   string `json:"stat_date"` // Format("20060102")
	Delta      int64  `json:"delta"`
	UpdateTime int64  `json:"update_time"` // Used for expired cleanup
}

// PopularContent i.e. top visited URL Path
type PopularContent struct {
	AppID   int64  `json:"app_id"`
	URLPath string `json:"url_path"`
	Amount  int64  `json:"amount"`
}

// InternalErrorInfo i.e. 502 or server offline
type InternalErrorInfo struct {
	Description string `json:"description"`
}

// GateHealth give basic information
type GateHealth struct {
	StartTime   int64   `json:"start_time"`
	CurrentTime int64   `json:"cur_time"`
	Version     string  `json:"version"`
	CPUPercent  float64 `json:"cpu_percent"`
	CPULoad1    float64 `json:"cpu_load1"`
	CPULoad5    float64 `json:"cpu_load5"`
	CPULoad15   float64 `json:"cpu_load15"`
	MemUsed     uint64  `json:"mem_used"`
	MemTotal    uint64  `json:"mem_total"`
	DiskUsed    uint64  `json:"disk_used"`
	DiskTotal   uint64  `json:"disk_total"`
	TimeZone    string  `json:"time_zone"`
	TimeOffset  int     `json:"time_offset"`
	ConCurrency int64   `json:"concurrency"`
}
