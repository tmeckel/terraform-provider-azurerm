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
)

func getArmLoadBalancerBackendAddressPoolIPAddressesSchema(forDataSource bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     forDataSource,
			ValidateFunc: validation.StringIsNotWhiteSpace,
		},
		"virtual_network_id": {
			Type:         schema.TypeString,
			Required:     !forDataSource,
			Computed:     forDataSource,
			ValidateFunc: azure.ValidateResourceID,
		},
		"ip_address": {
			Type:         schema.TypeString,
			Required:     !forDataSource,
			Computed:     forDataSource,
			ValidateFunc: validation.IsIPAddress,
		},
	}
}

func flattenArmLoadBalancerBackendAddressPoolIPAddresses(loadBalancerBackendAddresses *[]network.LoadBalancerBackendAddress) []interface{} {
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

func expandArmLoadBalancerBackendAddressPoolIPAddresses(d *schema.ResourceData) *[]network.LoadBalancerBackendAddress {
	backendIPAddresses := make([]network.LoadBalancerBackendAddress, 0)
	return &backendIPAddresses
}

func resourceArmLoadBalancerBackendAddressPoolIPAddresses() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadBalancerBackendAddressPoolIPAddressesCreateOrUpdate,
		Read:   resourceArmLoadBalancerBackendAddressPoolIPAddressesRead,
		Update: resourceArmLoadBalancerBackendAddressPoolIPAddressesCreateOrUpdate,
		Delete: resourceArmLoadBalancerBackendAddressPoolIPAddressesDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"loadbalancer_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: azure.ValidateResourceID,
			},

			"backend_address_pool_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},

			"backend_ip_addresses": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: getArmLoadBalancerBackendAddressPoolIPAddressesSchema(false),
				},
			},
		},
	}
}

func resourceArmLoadBalancerBackendAddressPoolIPAddressesCreateOrUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.LoadBalancerBackendAddressPoolsClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	loadBalancerID := d.Get("loadbalancer_id").(string)
	locks.ByID(loadBalancerID)
	defer locks.UnlockByID(loadBalancerID)

	resGroup, lbName, err := resourceGroupAndLBNameFromId(loadBalancerID)
	if err != nil {
		return fmt.Errorf("Error parsing Load Balancer ID %q: %+v", loadBalancerID, err)
	}

	loadBalancer, exists, err := retrieveLoadBalancerById(d, loadBalancerID, meta)
	if err != nil {
		return fmt.Errorf("Error Getting Load Balancer By ID %q: %+v", loadBalancerID, err)
	}
	if !exists {
		return fmt.Errorf("Load Balancer %q not found", loadBalancerID)
	}

	name := d.Get("backend_address_pool_name").(string)
	_, _, exists = FindLoadBalancerBackEndAddressPoolByName(loadBalancer, name)
	if !exists {
		return fmt.Errorf("Load Balancer BackEnd Address Pool %q not found", name)
	}

	future, err := client.CreateOrUpdate(ctx, resGroup, lbName, name, network.BackendAddressPool{
		BackendAddressPoolPropertiesFormat: &network.BackendAddressPoolPropertiesFormat{
			LoadBalancerBackendAddresses: expandArmLoadBalancerBackendAddressPoolIPAddresses(d),
		},
	})
	if err != nil {
		return fmt.Errorf("Error Setting Load Balancer Backend Pool addresses: %+v", err)
	}
	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for update of Load Balancer Backend Pool addresses: %+v", err)
	}
	return resourceArmLoadBalancerBackendAddressPoolIPAddressesRead(d, meta)
}

func resourceArmLoadBalancerBackendAddressPoolIPAddressesRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.LoadBalancerBackendAddressPoolsClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	loadBalancerID := d.Get("loadbalancer_id").(string)
	locks.ByID(loadBalancerID)
	defer locks.UnlockByID(loadBalancerID)

	resGroup, lbName, err := resourceGroupAndLBNameFromId(loadBalancerID)
	if err != nil {
		return fmt.Errorf("Error parsing Load Balancer ID %q: %+v", loadBalancerID, err)
	}

	loadBalancer, exists, err := retrieveLoadBalancerById(d, loadBalancerID, meta)
	if err != nil {
		return fmt.Errorf("Error Getting Load Balancer By ID: %+v", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] Load Balancer %q not found. Removing from state", loadBalancerID)
		return nil
	}

	name := d.Get("backend_address_pool_name").(string)
	_, _, exists = FindLoadBalancerBackEndAddressPoolByName(loadBalancer, name)
	if !exists {
		d.SetId("")
		log.Printf("[INFO] Load Balancer BackEnd Address Pool %q not found. Removing from state", name)
		return nil
	}

	backendAddressPool, err := client.Get(ctx, resGroup, lbName, name)
	if err != nil {
		return fmt.Errorf("Error Getting Load Balancer Backend Pool addresses: %+v", err)
	}
	d.Set("backend_ip_addresses", flattenArmLoadBalancerBackendAddressPoolIPAddresses(backendAddressPool.LoadBalancerBackendAddresses))

	return nil
}

func resourceArmLoadBalancerBackendAddressPoolIPAddressesDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.LoadBalancerBackendAddressPoolsClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	loadBalancerID := d.Get("loadbalancer_id").(string)
	locks.ByID(loadBalancerID)
	defer locks.UnlockByID(loadBalancerID)

	resGroup, lbName, err := resourceGroupAndLBNameFromId(loadBalancerID)
	if err != nil {
		return fmt.Errorf("Error parsing Load Balancer ID %q: %+v", loadBalancerID, err)
	}

	loadBalancer, exists, err := retrieveLoadBalancerById(d, loadBalancerID, meta)
	if err != nil {
		return fmt.Errorf("Error Getting Load Balancer By ID: %+v", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] Load Balancer %q not found. Removing from state", loadBalancerID)
		return nil
	}

	name := d.Get("backend_address_pool_name").(string)
	_, _, exists = FindLoadBalancerBackEndAddressPoolByName(loadBalancer, name)
	if !exists {
		d.SetId("")
		log.Printf("[INFO] Load Balancer BackEnd Address Pool %q not found. Removing from state", name)
		return nil
	}

	future, err := client.Delete(ctx, resGroup, lbName, name)
	if err != nil {
		return fmt.Errorf("Error Deleting Load Balancer Backend Pool addresses: %+v", err)
	}
	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for delete of Load Balancer Backend Pool addresses: %+v", err)
	}
	d.SetId("")

	return nil
}
