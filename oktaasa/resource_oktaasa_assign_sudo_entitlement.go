package oktaasa

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
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

type AssignedSudoEntitlement struct {
	SudoId string `json:"sudo_id"`
	Order  int    `json:"order"`
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

func resourceOKTAASAAssignSudoEntitlementRead(d *schema.ResourceData, m interface{}) error {
	sessionToken := m.(Bearer)
	assignedSudoEntitlementId := d.Id()

	//get project_name from terraform config.

	resp, err := SendGet(sessionToken.BearerToken, "/teams/"+teamName+"/entitlements/sudo/"+assignedSudoEntitlementId)

	if err != nil {
		return fmt.Errorf("[ERROR] Error when reading sudo entitlement. Id: %s. Error: %s", assignedSudoEntitlementId, err)
	}

	status := resp.StatusCode()

	if status == 200 {
		log.Printf("[DEBUG] assigned Sudo entitlement %s exists", assignedSudoEntitlementId)

		var assignedSudoEntitlement AssignedSudoEntitlement
		err := json.Unmarshal([]byte(resp.Body()), &assignedSudoEntitlement)

		if err != nil {
			return fmt.Errorf("[ERROR] Error when reading sudo entitlement state. Token: %s. Error: %s", assignedSudoEntitlementId, err)
		}

		d.Set("sudo_id", assignedSudoEntitlement.SudoId)
		d.Set("order", assignedSudoEntitlement.Order)

	} else if status == 404 {
		log.Printf("[DEBUG] No sudo entitlement %s in this project", assignedSudoEntitlementId)
		d.SetId("")
		return nil
	} else {
		return fmt.Errorf("[ERROR] Something went wrong while retrieving a list of sudo entitlements. Error: %s", resp)
	}
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
