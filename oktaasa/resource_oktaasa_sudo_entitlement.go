package oktaasa

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"regexp"
)

func resourceOKTAASASudoEntitlement() *schema.Resource {
	return &schema.Resource{
		Create: resourceOKTAASASudoEntitlementCreate,
		Read:   resourceOKTAASASudoEntitlementRead,
		Update: resourceOKTAASASudoEntitlementUpdate,
		Delete: resourceOKTAASASudoEntitlementDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"run_as": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"no_exec": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"no_passwd": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"command": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"args": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"args_type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"command": {
							Type:     schema.TypeString,
							Required: true,
						},
						"command_type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
			"set_env": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"sub_env": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"add_env": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

type StructuredCommand struct {
	Args        string `json:"args,omitempty"`
	ArgsType    string `json:"args_type,omitempty"`
	Command     string `json:"command"`
	CommandType string `json:"command_type,omitempty"`
	//RenderedCommand string `json:"rendered_command"`
}

type SudoEntitlement struct {
	Id                 string              `json:"id"`
	Name               string              `json:"name"`
	Description        string              `json:"description"`
	NoExec             bool                `json:"opt_no_exec"`
	NoPasswd           bool                `json:"opt_no_passwd"`
	RunAs              string              `json:"opt_run_as,omitempty"`
	SetEnv             bool                `json:"opt_set_env"`
	StructuredCommands []StructuredCommand `json:"structured_commands"`
	SubEnv             []string            `json:"sub_env,omitempty"`
	AddEnv             []string            `json:"add_env,omitempty"`
	Commands           []string            `json:"commands"`
}

func resourceOKTAASASudoEntitlementCreate(d *schema.ResourceData, m interface{}) error {
	// Bearer session token
	token := m.(Bearer)

	sudoEntitlement, err := createSudoEntitlementFromResourceData(d)
	if err != nil {
		return err
	}

	sudoEntitlementB, _ := json.Marshal(sudoEntitlement)

	log.Printf("[DEBUG] Creating sudo entitlement with payload %s", sudoEntitlementB)
	//make API call to create project
	resp, err := SendPost(token.BearerToken, "/teams/"+teamName+"/entitlements/sudo", sudoEntitlementB)

	if err != nil {
		return fmt.Errorf("[ERROR] Error when creating sudo entitlement: %s. Error: %s", sudoEntitlement.Name, err)
	}

	status := resp.StatusCode()

	if status >= 400 {
		return fmt.Errorf("[ERROR] Unexpected error when creating sudo entitlement %d, Error: %s, Response: %s", status, err, resp.Body())
	}

	newSudoEntitlement := SudoEntitlement{}

	jsonErr := json.Unmarshal(resp.Body(), &newSudoEntitlement)
	if jsonErr != nil {
		log.Printf("[DEBUG] Error storing SudoEntitlement: %s", jsonErr)
	}

	// update resource ID with Sudo Entitlement ID.
	d.SetId(newSudoEntitlement.Id)

	return resourceOKTAASASudoEntitlementRead(d, m)
}

func createSudoEntitlementFromResourceData(d *schema.ResourceData) (*SudoEntitlement, error) {
	structuredCommands, err := createStructuredCommandsFromResourceData(d)
	if err != nil {
		return nil, err
	}
	sudoEntitlement := &SudoEntitlement{
		Name:               d.Get("name").(string),
		Description:        d.Get("description").(string),
		StructuredCommands: structuredCommands,
		Commands:           make([]string, 0),
	}
	// name may only contain alphanumeric characters
	// (a-Z, 0-9), hyphens (-), underscores (_), and periods (.)
	ok, err := regexp.Match("^[a-zA-Z0-9-_.]+$", []byte(sudoEntitlement.Name))
	if !ok {
		return nil, fmt.Errorf("Sudo entitlement name \"%s\" is invalid, name may only contain alphanumeric characters (a-Z, 0-9), hyphens (-), underscores (_), and periods (.)", sudoEntitlement.Name)
	}
	return sudoEntitlement, err
}

func createStructuredCommandsFromResourceData(d *schema.ResourceData) ([]StructuredCommand, error) {
	structuredCommands := make([]StructuredCommand, 0)

	if c, ok := d.GetOk("command"); ok {
		cL := c.(*schema.Set).List()
		for _, c := range cL {
			cmd := c.(map[string]interface{})
			command := StructuredCommand{
				CommandType: cmd["command_type"].(string),
				Command:     cmd["command"].(string),
				ArgsType:    cmd["args_type"].(string),
				Args:        cmd["args"].(string),
			}
			structuredCommands = append(structuredCommands, command)
		}
	}

	return structuredCommands, nil
}

func resourceOKTAASASudoEntitlementRead(d *schema.ResourceData, m interface{}) error {
	sessionToken := m.(Bearer)
	sudoEntitlementId := d.Id()

	resp, err := SendGet(sessionToken.BearerToken, "/teams/"+teamName+"/entitlements/sudo/"+sudoEntitlementId)

	if err != nil {
		return fmt.Errorf("[ERROR] Error when reading sudo entitlement. Id: %s. Error: %s", sudoEntitlementId, err)
	}

	status := resp.StatusCode()

	if status == 200 {
		log.Printf("[DEBUG] Sudo entitlement %s exists", sudoEntitlementId)

		var sudoEntitlement SudoEntitlement
		err := json.Unmarshal([]byte(resp.Body()), &sudoEntitlement)

		if err != nil {
			return fmt.Errorf("[ERROR] Error when reading sudo entitlement state. Token: %s. Error: %s", sudoEntitlementId, err)
		}

		d.Set("name", sudoEntitlement.Name)
		d.Set("description", sudoEntitlement.Description)

	} else if status == 404 {
		log.Printf("[DEBUG] No sudo entitlement %s in this project", sudoEntitlementId)
		d.SetId("")
		return nil
	} else {
		return fmt.Errorf("[ERROR] Something went wrong while retrieving a list of sudo entitlements. Error: %s", resp)
	}
	return nil
}

func resourceOKTAASASudoEntitlementUpdate(d *schema.ResourceData, m interface{}) error {
	// Bearer session token
	token := m.(Bearer)

	sudoEntitlement, err := createSudoEntitlementFromResourceData(d)
	if err != nil {
		return err
	}

	sudoEntitlementB, _ := json.Marshal(sudoEntitlement)
	sudoEntitlementId := d.Id()

	log.Printf("[DEBUG] Updating sudo entitlement %s with payload %s", sudoEntitlementId, sudoEntitlementB)
	//make API call to update Sudo Entitlement
	resp, err := SendPut(token.BearerToken, "/teams/"+teamName+"/entitlements/sudo/"+sudoEntitlementId, sudoEntitlementB)

	if err != nil {
		return fmt.Errorf("[ERROR] Error when updating sudo entitlement: %s. Error: %s", sudoEntitlementId, err)
	}

	status := resp.StatusCode()

	if status >= 400 {
		return fmt.Errorf("[ERROR] Unexpected error when updating sudo entitlement %d, Error: %s, Response: %s", status, err, resp.Body())
	}

	newSudoEntitlement := SudoEntitlement{}

	jsonErr := json.Unmarshal(resp.Body(), &newSudoEntitlement)
	if jsonErr != nil {
		log.Printf("[DEBUG] Error storing SudoEntitlement: %s", jsonErr)
	}

	// update resource ID
	d.SetId(newSudoEntitlement.Id)

	return resourceOKTAASASudoEntitlementRead(d, m)
}

func resourceOKTAASASudoEntitlementDelete(d *schema.ResourceData, m interface{}) error {
	token := m.(Bearer)

	//get entitlement id from terraform config.
	sudoEntitlementId := d.Id()

	resp, err := SendDelete(token.BearerToken, "/teams/"+teamName+"/entitlements/sudo/"+sudoEntitlementId, make([]byte, 0))

	if err != nil {
		return fmt.Errorf("[ERROR] Error when deleting token: %s. Error: %s", sudoEntitlementId, err)
	}

	status := resp.StatusCode()

	if status < 300 || status == 404 {
		log.Printf("[INFO] Sudo entitlement %s was successfully deleted", d.Id())
	} else {
		return fmt.Errorf("[ERROR] Error while deleting sudo entitlement: %s, %s", status, resp.Body())
	}

	return nil
}
