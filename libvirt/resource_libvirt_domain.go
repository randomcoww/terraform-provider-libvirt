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
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(4 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"xml": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					_, err := parseXML(v.(string))
					if err != nil {
						es = append(es, fmt.Errorf("Failed to unmarshal input: %s", err))
					}
					return ws, es
				},
				StateFunc: func(v interface{}) (state string) {
					state, err := parseXML(v.(string))
					if err != nil {
						state = ""
					}
					return state
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

	err = domain.Create()
	if err != nil {
		return fmt.Errorf("Failed to create domain: %s", err)
	}

	uuid, err := domain.GetUUIDString()
	if err != nil {
		return fmt.Errorf("Failed to get UUID from domain: %s", err)
	}
	d.SetId(uuid)
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
		if err.(libvirt.Error).Code != libvirt.ERR_OPERATION_INVALID {
			return fmt.Errorf("Failed to shut down domain: %s", err)
		}
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
		Timeout:    90 * time.Second,
		MinTimeout: 1 * time.Second,
		Delay:      1 * time.Second,
	}

	_, err = shutdownStateConf.WaitForState()
	if err != nil {
		if err := domain.DestroyFlags(libvirt.DOMAIN_DESTROY_GRACEFUL); err != nil {
			return fmt.Errorf("Failed destroy domain: %s", err)
		}
	}
	if err := domain.UndefineFlags(
			libvirt.DOMAIN_UNDEFINE_MANAGED_SAVE | 
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
		return fmt.Errorf("Failed to get domain: %s", err)
	}
	defer domain.Free()
	return nil
}

func parseXML(inputXML string) (string, error) {
	schema := libvirtxml.Domain{}
	err := schema.Unmarshal(inputXML)
	if err != nil {
		return "", err
	}
	newXML, err := schema.Marshal()
	if err != nil {
		return "", err
	}
	return newXML, nil
}