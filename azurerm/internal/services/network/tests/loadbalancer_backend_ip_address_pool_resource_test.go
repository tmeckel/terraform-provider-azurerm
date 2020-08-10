package tests

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
)

func TestAccArmLoadBalancerBackendIPAddressPool_Create(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_lb_backend_address_pool", "test")

	var lb network.LoadBalancer
	addressPoolName := fmt.Sprintf("%d-address-pool", data.RandomInteger)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerBackEndAddressPool_basic(data, addressPoolName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerBackEndAddressPoolExists(addressPoolName, &lb),
					resource.TestCheckResourceAttr(
						"azurerm_lb_backend_address_pool.test", "id", backendAddressPoolId),
				),
			},
			{
				ResourceName:      "azurerm_lb.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccArmLoadBalancerBackendIPAddressPool_Update(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_lb_backend_address_pool", "test")

	var lb network.LoadBalancer
	addressPoolName := fmt.Sprintf("%d-address-pool", data.RandomInteger)
}

func TestAccArmLoadBalancerBackendIPAddressPool_Remove(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_lb_backend_address_pool", "test")

	var lb network.LoadBalancer
	addressPoolName := fmt.Sprintf("%d-address-pool", data.RandomInteger)
}

func testAccCheckArmLoadBalancerBackendIPAddressPoolExists(addressPoolName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return fmt.Errorf("Not implemented")
	}
}

func testAccCheckArmLoadBalancerBackendIPAddressPoolDestroy(s *terraform.State) error {
}

func testAccHclArmLoadBalancerBackendIPAddressPool(poolName string) string {

}
