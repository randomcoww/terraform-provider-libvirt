package libvirt

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	libvirt "github.com/libvirt/libvirt-go"
	libvirtxml "github.com/libvirt/libvirt-go-xml"
)

func resourceLibvirtNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceLibvirtNetworkCreate,
		Read:   resourceLibvirtNetworkRead,
		Update: resourceLibvirtNetworkUpdate,
		Delete: resourceLibvirtNetworkDelete,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(4 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"xml": {
				Type:     schema.TypeString,
				Required: true,
				StateFunc: func(v interface{}) (state string) {
					schema := libvirtxml.Network{}
					if err := schema.Unmarshal(v.(string)); err != nil {
						return ""
					}
					newXML, err := schema.Marshal()
					if err != nil {
						return ""
					}
					return newXML
				},
			},
		},
	}
}

func resourceLibvirtNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	virConn := meta.(*libvirt.Connect)

	network, err := virConn.NetworkDefineXML(d.Get("xml").(string))
	if err != nil {
		return fmt.Errorf("Failed to define network: %s", err)
	}
	defer network.Free()

	uuid, err := network.GetUUIDString()
	if err != nil {
		return fmt.Errorf("Failed to get UUID from network: %s", err)
	}
	d.SetId(uuid)

	ok, err := network.IsActive()
	if err != nil {
		return fmt.Errorf("Failed to check network status: %s", err)
	}
	if !ok {
		if err := network.Create(); err != nil {
			return fmt.Errorf("Failed to start network: %s", err)
		}
	}

	return nil
}

func resourceLibvirtNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("xml") {
		virConn := meta.(*libvirt.Connect)

		// Try redefining the network with new XML
		// The input XML will need to be modified to contain the current UUID
		// Otherwise it will fail with network already exists
		schema := libvirtxml.Network{}
		if err := schema.Unmarshal(d.Get("xml").(string)); err != nil {
			return fmt.Errorf("Failed to unmarshal XML: %s", err)
		}
		schema.UUID =  d.Id()
		newXML, err := schema.Marshal()
		if err != nil {
			return fmt.Errorf("Failed to marshal XML: %s", err)
		}
		network, err := virConn.NetworkDefineXML(newXML)
		if err != nil {
			return fmt.Errorf("Failed to redefine network: %s", err)
		}
		defer network.Free()
	}
	return nil
}

func resourceLibvirtNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	virConn := meta.(*libvirt.Connect)

	network, err := virConn.LookupNetworkByUUIDString(d.Id())
	if err != nil {
		if err.(libvirt.Error).Code == libvirt.ERR_NO_NETWORK {
			return nil
		}
		return fmt.Errorf("Failed to get network: %s", err)
	}
	defer network.Free()

	if err := network.Undefine(); err != nil {
		return fmt.Errorf("Failed to undefine network: %s", err)
	}
	return nil
}

func resourceLibvirtNetworkRead(d *schema.ResourceData, meta interface{}) error {
	virConn := meta.(*libvirt.Connect)

	network, err := virConn.LookupNetworkByUUIDString(d.Id())
	if err != nil {
		if err.(libvirt.Error).Code == libvirt.ERR_NO_NETWORK {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Failed to get network: %s", err)
	}
	defer network.Free()
	return nil
}