package libvirt

import (
	"io"
	"io/ioutil"
	"fmt"
	"os"
	"path"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	libvirt "github.com/libvirt/libvirt-go"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"endpoint": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "libvirt connection endpoint for operations. See https://libvirt.org/uri.html",
			},
			"client_cert": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "client cert TLS",
			},
			"client_key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "client key TLS",
			},
			"ca": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "client CA TLS",
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
	tempDir, err := ioutil.TempDir("", "libvirtpki-")
	if err != nil {
		return nil, err
	}
	// defer os.RemoveAll(tempDir)
	if err := writeFile(path.Join(tempDir, "libvirt", "clientcert.pem"), d.Get("client_cert").(string)); err != nil {
		return nil, err
	}
	if err := writeFile(path.Join(tempDir, "libvirt", "private", "clientkey.pem"), d.Get("client_key").(string)); err != nil {
		return nil, err
	}
	if err := writeFile(path.Join(tempDir, "cacert.pem"), d.Get("ca").(string)); err != nil {
		return nil, err
	}
	uri, err := libvirt.NewConnect(fmt.Sprintf("%s", d.Get("endpoint").(string)))
	if err != nil {
		return nil, err
	}
	return uri, nil
}

func writeFile(filePath, content string) error {
	f, err := os.Open(filePath)
	if err != nil {
		err = os.MkdirAll(path.Base(filePath), 0755)
		if err != nil {
			return err
		}
		f, err = os.Create(filePath)
		if err != nil {
			return err
		}
	}
	defer f.Close()
	_, err = io.WriteString(f, content)
	if err != nil {
			return err
	}
	return f.Sync()
}