package network

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/locks"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmLoadBalancerBackendIPAddressPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadBalancerBackendIPAddressPoolCreateOrUpdate,
		Read:   resourceArmLoadBalancerBackendIPAddressPoolRead,
		Update: resourceArmLoadBalancerBackendIPAddressPoolCreateOrUpdate,
		Delete: resourceArmLoadBalancerBackendIPAddressPoolDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},

			"loadbalancer_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: azure.ValidateResourceID,
			},

			"ip_address": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						"virtual_network_id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: azure.ValidateResourceID,
						},
						"ip_address": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsIPAddress,
						},
					},
				},
			},
		},
	}
}

func resourceArmLoadBalancerBackendIPAddressPoolCreateOrUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.LoadBalancerBackendAddressPoolsClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	loadBalancerID := d.Get("loadbalancer_id").(string)
	locks.ByID(loadBalancerID)
	defer locks.UnlockByID(loadBalancerID)

	resGroup, lbName, err := resourceGroupAndLBNameFromId(loadBalancerID)
	if err != nil {
		return fmt.Errorf("Parsing Load Balancer ID %q: %+v", loadBalancerID, err)
	}

	_, exists, err := retrieveLoadBalancerById(d, loadBalancerID, meta)
	if err != nil {
		return fmt.Errorf("Getting Load Balancer By ID %q: %+v", loadBalancerID, err)
	}
	if !exists {
		return fmt.Errorf("Load Balancer %q not found", loadBalancerID)
	}

	name := d.Get("name").(string)
	backendIPAddresses := expandArmLoadBalancerBackendIPAddressPool(d.Get("ip_address").([]interface{}))

	future, err := client.CreateOrUpdate(ctx, resGroup, lbName, name, network.BackendAddressPool{
		BackendAddressPoolPropertiesFormat: &network.BackendAddressPoolPropertiesFormat{
			LoadBalancerBackendAddresses: backendIPAddresses,
		},
	})
	if err != nil {
		return fmt.Errorf("Setting Load Balancer Backend Pool addresses: %+v", err)
	}

	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Waiting for update of Load Balancer Backend Pool addresses: %+v", err)
	}

	read, err := client.Get(ctx, resGroup, lbName, name)
	if err != nil {
		return fmt.Errorf("Retrieving Load Balancer backend address pool: %+v", err)
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot find created Load Balancer Backend Address Pool %s", name)
	}

	d.SetId(*read.ID)

	log.Printf("[INFO] Successfully created IP address backend pool [%s]", *read.ID)

	return resourceArmLoadBalancerBackendIPAddressPoolRead(d, meta)
}

func resourceArmLoadBalancerBackendIPAddressPoolRead(d *schema.ResourceData, meta interface{}) error {
	return readArmLoadBalancerBackendIPAddressPool(d, meta, false)
}

func resourceArmLoadBalancerBackendIPAddressPoolDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.LoadBalancerBackendAddressPoolsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	loadBalancerID := d.Get("loadbalancer_id").(string)
	locks.ByID(loadBalancerID)
	defer locks.UnlockByID(loadBalancerID)

	resGroup, lbName, err := resourceGroupAndLBNameFromId(loadBalancerID)
	if err != nil {
		return fmt.Errorf("Parsing Load Balancer ID %q: %+v", loadBalancerID, err)
	}

	loadBalancer, exists, err := retrieveLoadBalancerById(d, loadBalancerID, meta)
	if err != nil {
		return fmt.Errorf("Getting Load Balancer By ID: %+v", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] Load Balancer %q not found. Removing from state", loadBalancerID)
		return nil
	}

	name := d.Get("name").(string)
	_, _, exists = FindLoadBalancerBackEndAddressPoolByName(loadBalancer, name)
	if !exists {
		d.SetId("")
		log.Printf("[INFO] Load Balancer BackEnd Address Pool %q not found. Removing from state", name)
		return nil
	}

	future, err := client.Delete(ctx, resGroup, lbName, name)
	if err != nil {
		return fmt.Errorf("Deleting Load Balancer Backend Pool addresses: %+v", err)
	}
	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Waiting for delete of Load Balancer Backend Pool addresses: %+v", err)
	}
	d.SetId("")

	return nil
}

func readArmLoadBalancerBackendIPAddressPool(d *schema.ResourceData, meta interface{}, dataSourceMode bool) error {
	client := meta.(*clients.Client).Network.LoadBalancerBackendAddressPoolsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	var name string
	if dataSourceMode {
		name = d.Get("name").(string)
	} else {
		id, err := azure.ParseAzureResourceID(d.Id())
		if err != nil {
			return err
		}
		name = id.Path["backendAddressPools"]
	}

	loadBalancerID := d.Get("loadbalancer_id").(string)
	resGroup, lbName, err := resourceGroupAndLBNameFromId(loadBalancerID)
	if err != nil {
		return fmt.Errorf("Parsing Load Balancer ID %q: %+v", loadBalancerID, err)
	}

	_, exists, err := retrieveLoadBalancerById(d, loadBalancerID, meta)
	if err != nil {
		return fmt.Errorf("Getting Load Balancer By ID: %+v", err)
	}
	if !exists {
		if dataSourceMode {
			return fmt.Errorf("Load Balancer %q not found", loadBalancerID)
		}
		d.SetId("")
		log.Printf("[INFO] Load Balancer %q not found. Removing from state", loadBalancerID)
		return nil
	}

	backendAddressPool, err := client.Get(ctx, resGroup, lbName, name)
	if err != nil {
		return fmt.Errorf("Getting Load Balancer Backend Pool addresses: %+v", err)
	}
	if dataSourceMode {
		if backendAddressPool.ID == nil {
			return fmt.Errorf("Load Balancer Backend Pool with name [%s] does not specify an ID", name)
		}
		d.SetId(*backendAddressPool.ID)
	}
	d.Set("name", backendAddressPool.Name)
	d.Set("ip_address", flattenArmLoadBalancerBackendIPAddressPool(backendAddressPool.LoadBalancerBackendAddresses))

	return nil
}

func flattenArmLoadBalancerBackendIPAddressPool(loadBalancerBackendAddresses *[]network.LoadBalancerBackendAddress) []interface{} {
	backendIPAddresses := make([]interface{}, 0)
	if loadBalancerBackendAddresses != nil {
		for _, lbba := range *loadBalancerBackendAddresses {
			ipAddress := make(map[string]interface{})
			if name := lbba.Name; name != nil {
				ipAddress["name"] = name
			}
			if properties := lbba.LoadBalancerBackendAddressPropertiesFormat; properties != nil {
				if vnet := lbba.LoadBalancerBackendAddressPropertiesFormat.VirtualNetwork; vnet != nil {
					ipAddress["virtual_network_id"] = vnet.ID
				}
				if addr := lbba.LoadBalancerBackendAddressPropertiesFormat.IPAddress; addr != nil {
					ipAddress["ip_address"] = addr
				}
			}
			backendIPAddresses = append(backendIPAddresses, ipAddress)
		}
	}
	return backendIPAddresses
}

func expandArmLoadBalancerBackendIPAddressPool(input []interface{}) *[]network.LoadBalancerBackendAddress {
	output := make([]network.LoadBalancerBackendAddress, 0)

	for _, item := range input {
		vals := item.(map[string]interface{})

		var name *string
		if v, ok := vals["name"]; ok {
			name = utils.String(v.(string))
		}
		vnetID := vals["virtual_network_id"].(string)
		ipAddress := vals["ip_address"].(string)

		output = append(output, network.LoadBalancerBackendAddress{
			&network.LoadBalancerBackendAddressPropertiesFormat{
				VirtualNetwork: &network.SubResource{
					ID: &vnetID,
				},
				IPAddress: &ipAddress,
			},
			name,
		})
	}

	return &output
}
