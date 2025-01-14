/*
 * @Copyright Reserved By Janusec (https://www.janusec.com/).
 * @Author: U2
 * @Date: 2018-07-14 16:21:38
 * @Last Modified: U2, 2018-07-14 16:21:38
 */

package backend

import (
	"errors"
	"strings"
	"time"

	"github.com/Janusec/janusec/data"
	"github.com/Janusec/janusec/firewall"
	"github.com/Janusec/janusec/models"
	"github.com/Janusec/janusec/utils"
)

var (
	Apps []*models.Application
)

func SelectDestination(app *models.Application) string {
	destLen := len(app.Destinations)
	var dest *models.Destination
	if destLen == 1 {
		dest = app.Destinations[0]
	} else if destLen > 1 {
		ns := time.Now().Nanosecond()
		destIndex := ns % len(app.Destinations)
		dest = app.Destinations[destIndex]
	}
	utils.DebugPrintln("SelectDestination", dest)
	return dest.Destination
}

func GetApplicationByID(appID int64) (*models.Application, error) {
	for _, app := range Apps {
		if app.ID == appID {
			return app, nil
		}
	}
	return nil, errors.New("Not found.")
}

func GetWildDomainName(domain string) string {
	index := strings.Index(domain, ".")
	if index > 0 {
		wildDomain := "*" + domain[index:]
		return wildDomain
	}
	return ""
}

func GetApplicationByDomain(domain string) *models.Application {
	if domainRelation, ok := DomainsMap.Load(domain); ok {
		app := domainRelation.(models.DomainRelation).App //DomainsMap[domain].App
		return app
	}
	wildDomain := GetWildDomainName(domain) // *.janusec.com
	if domainRelation, ok := DomainsMap.Load(wildDomain); ok {
		app := domainRelation.(models.DomainRelation).App //DomainsMap[domain].App
		return app
	}
	return nil
}

func LoadApps() {
	Apps = Apps[0:0]
	if data.IsMaster {
		dbApps := data.DAL.SelectApplications()
		for _, dbApp := range dbApps {
			app := &models.Application{ID: dbApp.ID,
				Name:           dbApp.Name,
				InternalScheme: dbApp.InternalScheme,
				RedirectHttps:  dbApp.RedirectHttps,
				HSTSEnabled:    dbApp.HSTSEnabled,
				WAFEnabled:     dbApp.WAFEnabled,
				ClientIPMethod: dbApp.ClientIPMethod,
				Description:    dbApp.Description,
				Destinations:   []*models.Destination{}}
			Apps = append(Apps, app)
		}
	} else {
		// Slave
		rpcApps := RPCSelectApplications()
		if rpcApps != nil {
			Apps = rpcApps
		}
	}
}

func LoadDestinations() {
	for i, _ := range Apps {
		app := Apps[i]
		app.Destinations = data.DAL.SelectDestinationsByAppID(app.ID)
	}
}

/*
func LoadStaticDirs() {
	for i,_ := range Apps {
		app := Apps[i]
		rows,err := DB.Query("select directory from staticdirs where appID=$1", app.ID)
		utils.CheckError(err)
		for rows.Next() {
			var directory string
			err = rows.Scan(&directory)
			utils.CheckError(err)
			app.StaticDirs = append(app.StaticDirs, directory)
		}
	}
}
*/

func LoadAppDomainNames() {
	for i, _ := range Apps {
		app := Apps[i]
		for _, domain := range Domains {
			if domain.AppID == app.ID {
				app.Domains = append(app.Domains, domain)
			}
		}
	}
}

func GetApplications() ([]*models.Application, error) {
	return Apps, nil
}

func UpdateDestinations(app *models.Application, destinations []interface{}) {
	for _, appDest := range app.Destinations {
		// delete outdated destinations from DB
		if !InterfaceContainsDestinationID(destinations, appDest.ID) {
			data.DAL.DeleteDestinationByID(appDest.ID)
		}
	}
	var newDestinations []*models.Destination
	for _, destinationInterface := range destinations {
		// add new destinations to DB and app
		destMap := destinationInterface.(map[string]interface{})
		destID := int64(destMap["id"].(float64))
		destDest := strings.TrimSpace(destMap["destination"].(string))
		appID := app.ID //int64(destMap["appID"].(float64))
		nodeID := int64(destMap["node_id"].(float64))
		if destID == 0 {
			destID, _ = data.DAL.InsertDestination(destDest, appID, nodeID)
		} else {
			data.DAL.UpdateDestinationNode(destDest, appID, nodeID, destID)
		}
		dest := &models.Destination{ID: destID, Destination: destDest, AppID: appID, NodeID: nodeID}
		newDestinations = append(newDestinations, dest)
	}
	app.Destinations = newDestinations
	//fmt.Printf("Now Destinations: %v\n", app.Destinations)
}

func UpdateAppDomains(app *models.Application, appDomains []interface{}) {
	newAppDomains := []*models.Domain{}
	newDomainNames := []string{}
	for _, domainMap := range appDomains {
		domain := UpdateDomain(app, domainMap)
		newAppDomains = append(newAppDomains, domain)
		newDomainNames = append(newDomainNames, domain.Name)
	}
	for _, oldDomain := range app.Domains {
		if !InterfaceContainsDomainID(appDomains, oldDomain.ID) {
			data.DAL.DeleteDomainByDomainID(oldDomain.ID)
		}
	}
	app.Domains = newAppDomains
}

func UpdateApplication(param map[string]interface{}) (*models.Application, error) {
	application := param["object"].(map[string]interface{})
	appID := int64(application["id"].(float64))
	appName := application["name"].(string)
	internalScheme := application["internal_scheme"].(string)
	redirectHttps := application["redirect_https"].(bool)
	hstsEnabled := application["hsts_enabled"].(bool)
	wafEnabled := application["waf_enabled"].(bool)
	ipMethod := models.IPMethod(application["ip_method"].(float64))
	var description string
	var ok bool
	if description, ok = application["description"].(string); !ok {
		description = ""
	}
	var app *models.Application
	if appID == 0 {
		// new application
		newID := data.DAL.InsertApplication(appName, internalScheme, redirectHttps, hstsEnabled, wafEnabled, ipMethod, description)
		app = &models.Application{
			ID: newID, Name: appName,
			InternalScheme: internalScheme,
			Destinations:   []*models.Destination{},
			Domains:        []*models.Domain{},
			RedirectHttps:  redirectHttps,
			HSTSEnabled:    hstsEnabled,
			WAFEnabled:     wafEnabled,
			ClientIPMethod: ipMethod,
			Description:    description}
		Apps = append(Apps, app)
	} else {
		app, _ = GetApplicationByID(appID)
		if app != nil {
			data.DAL.UpdateApplication(appName, internalScheme, redirectHttps, hstsEnabled, wafEnabled, ipMethod, description, appID)
			app.Name = appName
			app.InternalScheme = internalScheme
			app.RedirectHttps = redirectHttps
			app.HSTSEnabled = hstsEnabled
			app.WAFEnabled = wafEnabled
			app.ClientIPMethod = ipMethod
			app.Description = description
		} else {
			return nil, errors.New("Application not found.")
		}
	}
	destinations := application["destinations"].([]interface{})
	UpdateDestinations(app, destinations)
	appDomains := application["domains"].([]interface{})
	UpdateAppDomains(app, appDomains)
	data.UpdateBackendLastModified()
	return app, nil
}

func GetApplicationIndex(appID int64) int {
	for i := 0; i < len(Apps); i++ {
		if Apps[i].ID == appID {
			return i
		}
	}
	return -1
}

func DeleteDestinationsByApp(appID int64) {
	data.DAL.DeleteDestinationsByAppID(appID)
}

func DeleteApplicationByID(appID int64) error {
	app, err := GetApplicationByID(appID)
	if err != nil {
		return err
	}
	DeleteDomainsByApp(app)
	DeleteDestinationsByApp(appID)
	firewall.DeleteCCPolicyByAppID(appID)
	err = data.DAL.DeleteApplication(appID)
	if err != nil {
		return err
	}
	i := GetApplicationIndex(appID)
	Apps = append(Apps[:i], Apps[i+1:]...)
	data.UpdateBackendLastModified()
	return nil
}
