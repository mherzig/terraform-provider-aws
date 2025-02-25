package ec2_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
)

func TestAccVPCEndpointServiceAllowedPrincipal_basic(t *testing.T) {
	lbName := fmt.Sprintf("testAccNLB-basic-%s", sdkacctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckVPCEndpointServiceAllowedPrincipalDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVPCEndpointServiceAllowedPrincipalConfig_basic(lbName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCEndpointServiceAllowedPrincipalExists("aws_vpc_endpoint_service_allowed_principal.foo"),
				),
			},
		},
	})
}

func testAccCheckVPCEndpointServiceAllowedPrincipalDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpc_endpoint_service_allowed_principal" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeVpcEndpointServicePermissions(&ec2.DescribeVpcEndpointServicePermissionsInput{
			ServiceId: aws.String(rs.Primary.Attributes["vpc_endpoint_service_id"]),
		})
		if err != nil {
			// Verify the error is what we want
			ec2err, ok := err.(awserr.Error)
			if !ok {
				return err
			}
			if ec2err.Code() != "InvalidVpcEndpointServiceId.NotFound" {
				return err
			}
			return nil
		}

		if len(resp.AllowedPrincipals) > 0 {
			return fmt.Errorf(
				"VCP Endpoint Service %s has allowed principals", rs.Primary.Attributes["vpc_endpoint_service_id"])
		}
	}

	return nil
}

func testAccCheckVPCEndpointServiceAllowedPrincipalExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPC Endpoint Service ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

		resp, err := conn.DescribeVpcEndpointServicePermissions(&ec2.DescribeVpcEndpointServicePermissionsInput{
			ServiceId: aws.String(rs.Primary.Attributes["vpc_endpoint_service_id"]),
		})
		if err != nil {
			return err
		}

		for _, principal := range resp.AllowedPrincipals {
			if aws.StringValue(principal.Principal) == rs.Primary.Attributes["principal_arn"] {
				return nil
			}
		}

		return fmt.Errorf("VPC Endpoint Service allowed principal not found")
	}
}

func testAccVPCEndpointServiceAllowedPrincipalConfig_basic(lbName string) string {
	return acctest.ConfigCompose(acctest.ConfigAvailableAZsNoOptIn(), fmt.Sprintf(
		`
resource "aws_vpc" "nlb_test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-vpc-endpoint-service-allowed-principal"
  }
}

resource "aws_lb" "nlb_test_1" {
  name = "%s"

  subnets = [
    aws_subnet.nlb_test_1.id,
    aws_subnet.nlb_test_2.id,
  ]

  load_balancer_type         = "network"
  internal                   = true
  idle_timeout               = 60
  enable_deletion_protection = false

  tags = {
    Name = "testAccVpcEndpointServiceBasicConfig_nlb1"
  }
}

resource "aws_subnet" "nlb_test_1" {
  vpc_id            = aws_vpc.nlb_test.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "tf-acc-vpc-endpoint-service-allowed-principal-1"
  }
}

resource "aws_subnet" "nlb_test_2" {
  vpc_id            = aws_vpc.nlb_test.id
  cidr_block        = "10.0.2.0/24"
  availability_zone = data.aws_availability_zones.available.names[1]

  tags = {
    Name = "tf-acc-vpc-endpoint-service-allowed-principal-2"
  }
}

data "aws_caller_identity" "current" {}

resource "aws_vpc_endpoint_service" "foo" {
  acceptance_required = false

  network_load_balancer_arns = [
    aws_lb.nlb_test_1.id,
  ]
}

resource "aws_vpc_endpoint_service_allowed_principal" "foo" {
  vpc_endpoint_service_id = aws_vpc_endpoint_service.foo.id

  principal_arn = data.aws_caller_identity.current.arn
}
`, lbName))
}
