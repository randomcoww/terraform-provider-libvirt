package libvirt

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	libvirt "github.com/libvirt/libvirt-go"
	libvirtxml "github.com/libvirt/libvirt-go-xml"
)

func resourceLibvirtDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceLibvirtDomainCreate,
		Read:   resourceLibvirtDomainRead,
		Update: resourceLibvirtDomainUpdate,
		Delete: resourceLibvirtDomainDelete,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(4 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"xml": {
				Type:     schema.TypeString,
				Required: true,
				StateFunc: func(v interface{}) (state string) {
					schema := libvirtxml.Domain{}
					err := schema.Unmarshal(v.(string))
					if err != nil {
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

func resourceLibvirtDomainCreate(d *schema.ResourceData, meta interface{}) error {
	virConn := meta.(*libvirt.Connect)

	domain, err := virConn.DomainDefineXML(d.Get("xml").(string))
	if err != nil {
		return fmt.Errorf("Failed to define domain: %s", err)
	}
	defer domain.Free()

	uuid, err := domain.GetUUIDString()
	if err != nil {
		return fmt.Errorf("Failed to get UUID from domain: %s", err)
	}
	d.SetId(uuid)
	return nil
}

func resourceLibvirtDomainUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("xml") {
		virConn := meta.(*libvirt.Connect)

		// Try redefining the domain with new XML
		// The input XML will need to be modified to contain the current UUID
		// Otherwise it will fail with domain already exists
		schema := libvirtxml.Domain{}
		err := schema.Unmarshal(d.Get("xml").(string))
		if err != nil {
			return fmt.Errorf("Failed to unmarshal XML: %s", err)
		}
		schema.UUID =  d.Id()
		newXML, err := schema.Marshal()
		if err != nil {
			return fmt.Errorf("Failed to marshal XML: %s", err)
		}
		domain, err := virConn.DomainDefineXML(newXML)
		if err != nil {
			return fmt.Errorf("Failed to redefine domain: %s", err)
		}
		defer domain.Free()
	}
	return nil
}

func resourceLibvirtDomainDelete(d *schema.ResourceData, meta interface{}) error {
	virConn := meta.(*libvirt.Connect)

	domain, err := virConn.LookupDomainByUUIDString(d.Id())
	if err != nil {
		if err.(libvirt.Error).Code == libvirt.ERR_NO_DOMAIN {
			return nil
		}
		return fmt.Errorf("Failed to get domain: %s", err)
	}
	defer domain.Free()

	if err := domain.UndefineFlags(libvirt.DOMAIN_UNDEFINE_MANAGED_SAVE |
		libvirt.DOMAIN_UNDEFINE_SNAPSHOTS_METADATA |
		libvirt.DOMAIN_UNDEFINE_NVRAM |
		libvirt.DOMAIN_UNDEFINE_CHECKPOINTS_METADATA); err != nil {
		return fmt.Errorf("Failed to undefine domain: %s", err)
	}
	return nil
}

func resourceLibvirtDomainRead(d *schema.ResourceData, meta interface{}) error {
	virConn := meta.(*libvirt.Connect)

	domain, err := virConn.LookupDomainByUUIDString(d.Id())
	if err != nil {
		return nil
	}
	defer domain.Free()
	return nil
}