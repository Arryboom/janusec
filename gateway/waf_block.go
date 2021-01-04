/*
 * @Copyright Reserved By Janusec (https://www.janusec.com/).
 * @Author: U2
 * @Date: 2018-07-14 16:38:30
 * @Last Modified: U2, 2018-07-14 16:38:30
 */

package gateway

import (
	"bytes"
	"html/template"
	"net/http"

	"janusec/models"
	"janusec/utils"
)

var tmplBlockReq, tmplBlockResp *template.Template

// GenerateBlockPage ...
func GenerateBlockPage(w http.ResponseWriter, hitInfo *models.HitInfo) {
	if tmplBlockReq == nil {
		tmplBlockReq, _ = template.New("blockReq").Parse(blockHTML)
	}
	w.WriteHeader(403)
	err := tmplBlockReq.Execute(w, hitInfo)
	if err != nil {
		utils.DebugPrintln("GenerateBlockPage tmpl.Execute error", err)
	}
}

// GenerateBlockConcent ...
func GenerateBlockConcent(hitInfo *models.HitInfo) []byte {
	if tmplBlockResp == nil {
		tmplBlockResp, _ = template.New("blockResp").Parse(blockHTML)
	}
	buf := &bytes.Buffer{}
	err := tmplBlockResp.Execute(buf, hitInfo)
	if err != nil {
		utils.DebugPrintln("GenerateBlockConcent tmpl.Execute error", err)
	}
	return buf.Bytes()
}

const blockHTML = `<!DOCTYPE html>
<html>
<head>
<title>403 Forbidden</title>
</head>
<style>
body {
    font-family: Arial, Helvetica, sans-serif;
    text-align: center;
}

.text-logo {
    display: block;
	width: 260px;
    font-size: 48px;  
    background-color: #F9F9F9;    
    color: #f5f5f5;    
    text-decoration: none;
    text-shadow: 2px 2px 4px #000000;
    box-shadow: 2px 2px 3px #D5D5D5;
    padding: 15px; 
    margin: auto;    
}

.block_div {
    padding: 10px;
    width: 70%;    
    margin: auto;
}

</style>
<body>
<div class="block_div">
<h1 class="text-logo">JANUSEC</h1>
<hr>
Reason: {{.VulnName}}, Policy ID: {{.PolicyID}}, by Janusec Application Gateway
</div>
</body>
</html>
`
