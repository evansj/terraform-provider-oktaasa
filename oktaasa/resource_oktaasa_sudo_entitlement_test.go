package oktaasa

import (
	//	"encoding/json"
	//	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestSudoEntitlement(t *testing.T) {
	sudoEntitlementName := "test-sudo-entitlement"
	sudoEntitlement := &SudoEntitlement{}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testSudoEntitlementDestroy(sudoEntitlement),
		Steps: []resource.TestStep{
			{
				Config: testSudoEntitlementCreateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"oktaasa_sudo_entitlement.test", "name", sudoEntitlementName,
					),
					resource.TestCheckResourceAttr(
						"oktaasa_sudo_entitlement.test", "next_unix_uid", "60120",
					),
					resource.TestCheckResourceAttr(
						"oktaasa_sudo_entitlement.test", "next_unix_gid", "63020",
					),
					resource.TestCheckResourceAttr(
						"oktaasa_enrollment_token.test-token", "sudoEntitlement_name", sudoEntitlementName,
					),
					resource.TestCheckResourceAttr(
						"oktaasa_enrollment_token.test-token", "description", "Token for TestAcc",
					),
				),
			},
		},
	})
}

func testSudoEntitlementDestroy(p *SudoEntitlement) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		//config := testAccProvider.Meta().(Bearer)

		/*		resp, err := SendGet(config.BearerToken, "/teams/"+config.TeamName+"/sudoEntitlements/"+p.Name)
				if err != nil {
					return fmt.Errorf("error getting data source: %s", err)
				}

				status := resp.StatusCode()
				deleted, err := checkSoftDelete(resp.Body())
				if err != nil {
					return fmt.Errorf("error while checking deleted status: %s", err)
				}

				if status == 200 && !deleted {
					return fmt.Errorf("sudoEntitlement still exists")
				}
		*/
		return nil
	}
}

/*
func testSudoEntitlementCheckExists(rn string, p *EnrollmentToken) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		sudoEntitlementName := "test-acc-sudoEntitlement2"

		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		// resource ID is token name
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		config := testAccProvider.Meta().(Bearer)

		resp, err := SendGet(config.BearerToken, "/teams/"+config.TeamName+"/sudoEntitlements/"+sudoEntitlementName+"/server_enrollment_tokens/"+rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		err = json.Unmarshal(resp.Body(), p)
		if err != nil {
			return fmt.Errorf("error unmarshaling data source response: %s", err)
		}

		return nil
	}
}
*/
/*
func testSudoEntitlementCheckDestroy(p *EnrollmentToken) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(Bearer)

		resp, err := SendGet(config.BearerToken, "/teams/"+config.TeamName+"/server_enrollment_tokens/"+p.Id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		status := resp.StatusCode()
		deleted, err := checkSoftDelete(resp.Body())
		if err != nil {
			return fmt.Errorf("error while checking deleted status: %s", err)
		}

		if status == 200 && !deleted {
			return fmt.Errorf("token still exists")
		}

		return nil
	}
}
*/

const testSudoEntitlementCreateConfig = `
resource "oktaasa_sudo_entitlement" "test" {
  name        = "test-sudo-entitlement"
  description = "description"
  run_as      = "root"

  command {
		command      = "command"
		command_type = "raw"
	}
}
`
