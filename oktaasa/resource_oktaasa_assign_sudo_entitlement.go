package oktaasa

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strings"
	"sync"
	"time"
)

/*
"You have to assign it to the group for a specific project, I believe the endpoint is
https://app.scaleft.com/v1/teams/{{teamname}}/projects/{{Project}}/groups/{{Group}}/entitlements/sudo.
The body will then contain the specific sudo entitlement ID

For reference, the body looked like this for me when I was doing it via Workflows (as an example):

{
 "sudo_id": "{{sudoid}}",
 "order": 99
}
"
*/

/*
type LockableSudoEntitlements struct {
	mu sync.Mutex
	ma map[ProjectAndGroup]*SudoEntitlements
}

var CachedEntitlements = &LockableSudoEntitlements{ma: make(map[ProjectAndGroup]*SudoEntitlements)}
*/

func resourceOKTAASAAssignSudoEntitlement() *schema.Resource {
	return &schema.Resource{
		Create: resourceOKTAASAAssignSudoEntitlementCreate,
		Read:   resourceOKTAASAAssignSudoEntitlementRead,
		Update: resourceOKTAASAAssignSudoEntitlementUpdate,
		Delete: resourceOKTAASAAssignSudoEntitlementDelete,

		Schema: map[string]*schema.Schema{
			"project_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"sudo_entitlement_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"order": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
		},
	}
}

type ProjectAndGroup struct {
	mu      sync.Mutex
	Project string
	Group   string
}

type AssignedSudoEntitlement struct {
	SudoId string `json:"sudo_id"`
	Order  int    `json:"order"`
}

type SudoEntitlements struct {
	List []struct {
		ID        string    `json:"id,omitempty"`
		SudoID    string    `json:"sudo_id,omitempty"`
		SudoName  string    `json:"sudo_name,omitempty"`
		Name      string    `json:"name,omitempty"`
		ProjectID string    `json:"project_id,omitempty"`
		GroupID   string    `json:"group_id,omitempty"`
		Order     int       `json:"order,omitempty"`
		CreatedAt time.Time `json:"created_at,omitempty"`
		DeletedAt time.Time `json:"deleted_at,omitempty"`
	} `json:"list"`
}

func resourceOKTAASAAssignSudoEntitlementCreate(d *schema.ResourceData, m interface{}) error {
	// Bearer session token
	token := m.(Bearer)

	assignedSudoEntitlement, err := createAssignedSudoEntitlementFromResourceData(d)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Sudo Entitlement id is %s", assignedSudoEntitlement.SudoId)
	assignedSudoEntitlementB, _ := json.Marshal(assignedSudoEntitlement)
	projectName := d.Get("project_name").(string)
	groupName := d.Get("group_name").(string)
	assignSudoEntitlementUrl := "/teams/" + teamName +
		"/projects/" + projectName +
		"/groups/" + groupName +
		"/entitlements/sudo"

	log.Printf("[DEBUG] Assigning sudo entitlement by sending payload %s to %s", assignedSudoEntitlementB, assignSudoEntitlementUrl)
	//make API call to assign sudo entitlement to project + group
	resp, err := SendPost(token.BearerToken, assignSudoEntitlementUrl, assignedSudoEntitlementB)

	if err != nil {
		return fmt.Errorf("[ERROR] Error when assigning sudo entitlement: %s", err)
	}

	status := resp.StatusCode()

	log.Printf("[DEBUG] assigned sudo entitlement, status %d, Response: %s", status, resp.Body())

	if status > -200 && status <= 204 {
		// Success
		// update resource ID.
		d.SetId(projectName + "/" + groupName + "/" + assignedSudoEntitlement.SudoId)
	} else {
		return fmt.Errorf("[ERROR] Unexpected error when assigning sudo entitlement %d, Error: %s, Response: %s", status, err, resp.Body())
	}

	d.Set("sudo_id", assignedSudoEntitlement.SudoId)
	d.Set("order", assignedSudoEntitlement.Order)

	return nil
}

func createAssignedSudoEntitlementFromResourceData(d *schema.ResourceData) (*AssignedSudoEntitlement, error) {
	var err error
	assignedSudoEntitlement := &AssignedSudoEntitlement{
		SudoId: d.Get("sudo_entitlement_id").(string),
		Order:  d.Get("order").(int),
	}
	return assignedSudoEntitlement, err
}

/* There is no way to read a single Sudo Entitlement. Our only option
 * is to GET :team_name/projects/:project_name/groups/:group_name/entitlements/sudo/
 * which will return a list like this:
 * {
 *     "list": [
 *         {
 *             "id": "a9a33f2f-edb4-43f6-9799-ac77ce64df07",
 *             "sudo_id": "2604bf50-1952-45b8-8ffe-0876cced9069",
 *             "sudo_name": "full-sudo",
 *             "name": "full-sudo",
 *             "project_id": "1c828acb-7bdd-412e-9d33-24cf6fd045aa",
 *             "group_id": "0d59af5c-cf76-42da-8c3d-a8136a443e6a",
 *             "order": 50,
 *             "created_at": "2021-06-11T15:55:21.630317Z",
 *             "deleted_at": null
 *         }
 *     ]
 * }
 */
func resourceOKTAASAAssignSudoEntitlementRead(d *schema.ResourceData, m interface{}) error {
	sessionToken := m.(Bearer)
	assignedSudoEntitlementId := d.Id()
	projectName := d.Get("project_name").(string)
	groupName := d.Get("group_name").(string)
	projectAndGroup := ProjectAndGroup{
		Project: projectName,
		Group:   groupName,
	}

	entitlementsList := new(SudoEntitlements)
	//entitlementsList, found := GetSudoEntitlementsFromCache(projectAndGroup)
	//log.Printf("[DEBUG] ASER %t Object at key %+v was %+v", found, projectAndGroup, entitlementsList)
	//if !found {
	// Fetch the full list for this project and group
	url := "/teams/" + teamName + "/projects/" + projectName + "/groups/" + groupName + "/entitlements/sudo/"
	log.Printf("[DEBUG] ASER Going to fetch all sudo entitlements for %+v from %s", projectAndGroup, url)
	resp, err := SendGet(sessionToken.BearerToken, url)

	if err != nil {
		return fmt.Errorf("[ERROR] ASER Error when reading sudo entitlement. Id: %+v. Error: %+v", assignedSudoEntitlementId, err)
	}

	status := resp.StatusCode()
	if status == 200 {
		body := resp.Body()
		log.Printf("[DEBUG] ASER Got response body %s", body)

		err := json.Unmarshal([]byte(body), &entitlementsList)

		if err != nil {
			return fmt.Errorf("[ERROR] ASER Error when reading sudo entitlement assignments. Error: %s\n%s", err, body)
		}

		log.Printf("[DEBUG] ASER %s: %+v", url, entitlementsList)
		//CachedEntitlements[projectAndGroup] = entitlementsList
	} else if status == 404 {
		log.Printf("[DEBUG] ASER No sudo entitlements %s in %+v", assignedSudoEntitlementId, projectAndGroup)
		d.SetId("")
		return nil
	} else {
		return fmt.Errorf("[ERROR] ASER Something went wrong while retrieving a list of sudo entitlements for %+v. Error: %s", projectAndGroup, resp)
	}
	//}

	// assignedSudoEntitlementId will be "projectName/groupName/sudoId"
	aseiSplit := strings.Split(assignedSudoEntitlementId, "/")
	sudoId := aseiSplit[2]
	for _, entitlement := range entitlementsList.List {
		if sudoId == entitlement.SudoID {
			if entitlement.DeletedAt.IsZero() {
				d.Set("sudo_id", entitlement.SudoID)
				// If our order is 0, we don't care what the server order is
				if d.Get("order").(int) != 0 {
					d.Set("order", entitlement.Order)
				}
				return nil
			} else {
				log.Printf("[DEBUG] ASER %s found, but it was deleted %v", sudoId, entitlement.DeletedAt)
			}
		}
	}

	d.SetId("")
	return nil
}

func resourceOKTAASAAssignSudoEntitlementUpdate(d *schema.ResourceData, m interface{}) error {
	// not possible to update token.
	return nil
}

func resourceOKTAASAAssignSudoEntitlementDelete(d *schema.ResourceData, m interface{}) error {
	token := m.(Bearer)

	//get entitlement id from terraform config.
	assignedSudoEntitlementId := d.Id()

	resp, err := SendDelete(token.BearerToken, "/teams/"+teamName+"/entitlements/sudo/"+assignedSudoEntitlementId, make([]byte, 0))

	if err != nil {
		return fmt.Errorf("[ERROR] Error when deleting token: %s. Error: %s", assignedSudoEntitlementId, err)
	}

	status := resp.StatusCode()

	if status < 300 || status == 404 {
		log.Printf("[INFO] Sudo entitlement %s was successfully deleted", d.Id())
	} else {
		return fmt.Errorf("[ERROR] Error while deleting sudo entitlement: %s, %s", status, resp.Body())
	}

	return nil
}
