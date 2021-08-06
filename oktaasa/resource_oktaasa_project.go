package oktaasa

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

func resourceOKTAASAProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceOKTAASAProjectCreate,
		Read:   resourceOKTAASAProjectRead,
		Update: resourceOKTAASAProjectUpdate,
		Delete: resourceOKTAASAProjectDelete,

		Schema: map[string]*schema.Schema{
			"project_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"next_unix_uid": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"next_unix_gid": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"create_server_users": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"force_shared_ssh_users": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"forward_traffic": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"rdp_session_recording": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"require_preauthorization": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"shared_admin_user_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"shared_standard_user_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"ssh_session_recording": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
		Importer: &schema.ResourceImporter{
			State: resourceOKTAASAProjectImport,
		},
	}
}

func resourceOKTAASAProjectCreate(d *schema.ResourceData, m interface{}) error {
	token := m.(Bearer)

	project, err := buildProjectFromResourceData(d)
	if err != nil {
		return err
	}
	projectB, _ := json.Marshal(project)

	d.SetId(project.Name)
	log.Printf("[DEBUG] Project POST body: %s", projectB)

	//make API call to create project
	resp, err := SendPost(token.BearerToken, "/teams/"+teamName+"/projects", projectB)

	if err != nil {
		return fmt.Errorf("[ERROR] Error when creating project: %s. Error: %s", project.Name, err)
	}

	status := resp.StatusCode()

	if status == 201 {
		log.Printf("[INFO] Project %s was successfully created", project.Name)
	} else {
		log.Printf("[ERROR] Something went wrong while creating project. Error: %s", resp)
	}

	return resourceOKTAASAProjectRead(d, m)
}

func buildProjectFromResourceData(d *schema.ResourceData) (*Project, error) {
	project := &Project{
		Name:                   d.Get("project_name").(string),
		NextUnixUid:            d.Get("next_unix_uid").(int),
		NextUnixGid:            d.Get("next_unix_gid").(int),
		CreateServerUsers:      d.Get("create_server_users").(bool),
		ForceSharedSshUsers:    d.Get("force_shared_ssh_users").(bool),
		ForwardTraffic:         d.Get("forward_traffic").(bool),
		RDPSessionRecording:    d.Get("rdp_session_recording").(bool),
		RequirePreauth:         d.Get("require_preauthorization").(bool),
		SharedAdminUserName:    d.Get("shared_admin_user_name").(string),
		SharedStandardUserName: d.Get("shared_standard_user_name").(string),
		SshSessionRecording:    d.Get("ssh_session_recording").(bool),
	}

	var err error
	if project.ForceSharedSshUsers && (project.SharedStandardUserName == "" || project.SharedAdminUserName == "") {
		err = fmt.Errorf("error creating resource: shared_standard_user_name and shared_admin_user_name must be provided if force_shared_ssh_users is true")
	}

	return project, err
}

type Project struct {
	Name                   string `json:"name"`
	DeletedAt              string `json:"deleted_at,omitempty"`
	CreateServerUsers      bool   `json:"create_server_users"`
	ForceSharedSshUsers    bool   `json:"force_shared_ssh_users"`
	ForwardTraffic         bool   `json:"forward_traffic"`
	NextUnixUid            int    `json:"next_unix_uid,omitempty"`
	NextUnixGid            int    `json:"next_unix_gid,omitempty"`
	RDPSessionRecording    bool   `json:"rdp_session_recording"`
	RequirePreauth         bool   `json:"require_preauth_for_creds"`
	SharedAdminUserName    string `json:"shared_admin_user_name,omitempty"`
	SharedStandardUserName string `json:"shared_standard_user_name,omitempty"`
	SshSessionRecording    bool   `json:"ssh_session_recording"`
}

func resourceOKTAASAProjectRead(d *schema.ResourceData, m interface{}) error {
	token := m.(Bearer)
	projectName := d.Id()

	resp, err := SendGet(token.BearerToken, "/teams/"+teamName+"/projects/"+projectName)

	if err != nil {
		return fmt.Errorf("[ERROR] Error when reading project state: %s. Error: %s", projectName, err)
	}

	status := resp.StatusCode()

	// API can return 200, but also have deleted_at or removed_at value.
	deleted, err := checkSoftDelete(resp.Body())

	if err != nil {
		return fmt.Errorf("[ERROR] Error when attempting to check for soft delete, while reading project state: %s. Error: %s", projectName, err)
	}

	if status == 200 && deleted == true {
		log.Printf("[INFO] Project %s was removed.", projectName)
		d.SetId("")
		return nil
	} else if status == 200 && deleted == false {
		log.Printf("[INFO] Project %s exists.", projectName)

		var project Project

		err := json.Unmarshal(resp.Body(), &project)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal project settings")
		}

		d.SetId(project.Name)
		d.Set("project_name", project.Name)
		d.Set("create_server_users", project.CreateServerUsers)
		d.Set("force_shared_ssh_users", project.ForceSharedSshUsers)
		d.Set("forward_traffic", project.ForwardTraffic)
		if d.Get("next_unix_uid").(int) != 0 {
			// We don't care what next_unix_uid is serverside
			// unless we have specified a value for it
			d.Set("next_unix_uid", project.NextUnixUid)
		}
		if d.Get("next_unix_gid").(int) != 0 {
			// We don't care what next_unix_gid is serverside
			// unless we have specified a value for it
			d.Set("next_unix_gid", project.NextUnixGid)
		}
		d.Set("rdp_session_recording", project.RDPSessionRecording)
		d.Set("require_preauthorization", project.RequirePreauth)
		d.Set("shared_admin_user_name", project.SharedAdminUserName)
		d.Set("shared_standard_user_name", project.SharedStandardUserName)
		d.Set("ssh_session_recording", project.SshSessionRecording)

		return nil
	} else if status == 404 {
		log.Printf("[INFO] Project %s does not exist", projectName)
		d.SetId("")
		return nil
	} else {
		return fmt.Errorf("[DEBUG] failed to read project state. Project: %s Status code: %d", projectName, status)
	}
}

func resourceOKTAASAProjectImport(d *schema.ResourceData, m interface{}) (imported []*schema.ResourceData, err error) {
	id := d.Id()
	err = resourceOKTAASAProjectRead(d, m)
	if err == nil && d.Id() != "" {
		imported = append(imported, d)
	} else {
		err = fmt.Errorf("[DEBUG] project with id %s not found.", id)
	}
	return imported, err
}

func resourceOKTAASAProjectUpdate(d *schema.ResourceData, m interface{}) error {
	token := m.(Bearer)

	project, err := buildProjectFromResourceData(d)
	if err != nil {
		return err
	}

	projectB, _ := json.Marshal(project)

	d.SetId(project.Name)
	log.Printf("[DEBUG] Project POST body: %s", projectB)

	//make API call to create project
	resp, err := SendPut(token.BearerToken, "/teams/"+teamName+"/projects/"+project.Name, projectB)

	if err != nil {
		return fmt.Errorf("[ERROR] Error updating project settings. Project: %s. Error: %s", project.Name, err)
	}

	status := resp.StatusCode()

	if status == 204 {
		log.Printf("[INFO] Project %s was successfully updated", project.Name)
	} else {
		return fmt.Errorf("[ERROR] Something went wrong while updating the project %s. Error: %s", project.Name, resp)

	}

	return resourceOKTAASAProjectRead(d, m)
}

func resourceOKTAASAProjectDelete(d *schema.ResourceData, m interface{}) error {
	token := m.(Bearer)

	//get project_name from terraform config.
	projectName := d.Get("project_name").(string)

	resp, err := SendDelete(token.BearerToken, "/teams/"+teamName+"/projects/"+projectName, make([]byte, 0))

	if err != nil {
		return fmt.Errorf("[ERROR] Error when deleting project: %s. Error: %s", projectName, err)
	}

	status := resp.StatusCode()

	if status < 300 || status == 400 {
		log.Printf("[INFO] Project %s was successfully deleted", projectName)
	} else {
		log.Printf("[ERROR] Something went wrong while deleting project %s. Error: %s", projectName, resp)
	}

	return nil
}
