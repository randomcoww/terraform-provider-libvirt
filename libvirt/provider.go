package libvirt

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	libvirt "github.com/libvirt/libvirt-go"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"uri": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("LIBVIRT_DEFAULT_URI", nil),
				Description: "libvirt connection URI for operations. See https://libvirt.org/uri.html",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"libvirt_domain": resourceLibvirtDomain(),
			"libvirt_network": resourceLibvirtNetwork(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	client, err := libvirt.NewConnect(d.Get("uri").(string))
	if err != nil {
		return nil, err
	}
	return client, nil
}
