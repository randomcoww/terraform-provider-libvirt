package libvirt

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	libvirt "github.com/libvirt/libvirt-go"
	libvirtxml "github.com/libvirt/libvirt-go-xml"
)

func resourceLibvirtDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceLibvirtDomainCreate,
		Read:   resourceLibvirtDomainRead,
		Delete: resourceLibvirtDomainDelete,
		Exists: resourceLibvirtDomainExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(4 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"xml": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateXML,
			},
			"domain": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func validateXML(v interface{}, k string) (ws []string, es []error) {
	schema := libvirtxml.Domain{}
	err := schema.Unmarshal(v.(string))
	if err != nil {
		es = append(es, fmt.Errorf("Failed to unmarshal input: %s", err))
	}
	return ws, es
}

func resourceLibvirtDomainCreate(d *schema.ResourceData, meta interface{}) error {
	virConn := meta.(*libvirt.Connect)

	domain, err := virConn.DomainCreateXML(d.Get("xml").(string), 24)
	if err != nil {
		return fmt.Errorf("Failed to create domain: %s", err)
	}
	defer domain.Free()

	resultXML, err := domain.GetXMLDesc(8)
	if err != nil {
		return fmt.Errorf("Failed to get XML from domain: %s", err)
	}
	d.Set("domain", resultXML)

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
		schema.UUID = d.Id()
		newXML, err := schema.Marshal()
		if err != nil {
			return fmt.Errorf("Failed to marshal XML with UUID: %s", err)
		}

		// Now try defining and check if the resulting XML changed
		// If there is a change, trigger destroy and create through ForceNew
		domain, err := virConn.DomainDefineXML(newXML)
		if err != nil {
			return fmt.Errorf("Failed to redefine domain: %s", err)
		}
		defer domain.Free()

		resultXML, err := domain.GetXMLDesc(8)
		if err != nil {
			return fmt.Errorf("Failed to get XML from domain: %s", err)
		}
		d.Set("domain", resultXML)
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

	if err := domain.ShutdownFlags(0); err != nil {
		return fmt.Errorf("Failed to shut down domain: %s", err)
	}

	shutdownStateConf := &resource.StateChangeConf{
		Refresh: func() (interface{}, string, error) {
			state, _, err := domain.GetState()
			if err != nil {
				return 0, "", err
			}
			return state, fmt.Sprintf("%d", state), nil
		},
		Pending: []string{
			fmt.Sprintf("%d", libvirt.DOMAIN_SHUTDOWN),
		},
		Target: []string{
			fmt.Sprintf("%d", libvirt.DOMAIN_SHUTOFF),
		},
		Timeout:    3 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      5 * time.Second,
	}
	_, err = shutdownStateConf.WaitForState()
	if err != nil {
		if err := domain.DestroyFlags(1); err != nil {
			return fmt.Errorf("Failed destroy domain: %s", err)
		}
	}
	if err := domain.UndefineFlags(23); err != nil {
		return fmt.Errorf("Failed to undefine domain: %s", err)
	}
	return nil
}

func resourceLibvirtDomainRead(d *schema.ResourceData, meta interface{}) error {
	virConn := meta.(*libvirt.Connect)

	domain, err := virConn.LookupDomainByUUIDString(d.Id())
	if err != nil {
		return fmt.Errorf("Failed to get domain: %s", err)
	}
	defer domain.Free()

	resultXML, err := domain.GetXMLDesc(8)
	if err != nil {
		return fmt.Errorf("Failed to get XML from domain: %s", err)
	}
	d.Set("domain", resultXML)
	return nil
}

func resourceLibvirtDomainExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	virConn := meta.(*libvirt.Connect)

	domain, err := virConn.LookupDomainByUUIDString(d.Id())
	if err != nil {
		if err.(libvirt.Error).Code == libvirt.ERR_NO_DOMAIN {
			return false, nil
		}
		return false, err
	}
	defer domain.Free()
	return true, nil
}
