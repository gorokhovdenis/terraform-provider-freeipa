package freeipa

import (
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	ipa "github.com/tehwalris/go-freeipa/freeipa"
)

func resourceFreeIPAHost() *schema.Resource {
	return &schema.Resource{
		Create: resourceFreeIPAHostCreate,
		Read:   resourceFreeIPAHostRead,
		Update: resourceFreeIPAHostUpdate,
		Delete: resourceFreeIPAHostDelete,
		Importer: &schema.ResourceImporter{
			State: resourceFreeIPAHostImport,
		},

		Schema: map[string]*schema.Schema{
			"fqdn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"random": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"userpassword": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"randompassword": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"force": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceFreeIPAHostCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO][freeipa] Creating Host: %s", d.Id())
	client, err := meta.(*Config).Client()
	if err != nil {
		return err
	}

	fqdn := d.Get("fqdn").(string)
	description := d.Get("description").(string)
	random := d.Get("random").(bool)
	userpassword := d.Get("userpassword").(string)
	force := d.Get("force").(bool)

	optArgs := ipa.HostAddOptionalArgs{
		Description: &description,
		Random:      &random,
		Force:       &force,
	}

	if userpassword != "" {
		optArgs.Userpassword = &userpassword
	}

	res, err := client.HostAdd(
		&ipa.HostAddArgs{
			Fqdn: fqdn,
		},
		&optArgs,
	)
	if err != nil {
		return err
	}

	d.SetId(fqdn)

	// randompassword is not returned by HostShow
	if d.Get("random").(bool) {
		d.Set("randompassword", *res.Result.Randompassword)
	}

	// FIXME: When using a LB in front of a FreeIPA cluster, sometime the record
	// is not replicated on the server where the read is done, so we have to
	// retry to not have "Error: NotFound (4001)".
	// Maybe we should use resource.StateChangeConf instead...
	sleepDelay := 1 * time.Second
	for {
		err := resourceFreeIPAHostRead(d, meta)
		if err == nil {
			return nil
		}
		time.Sleep(sleepDelay)
		sleepDelay = sleepDelay * 2
	}
}

func resourceFreeIPAHostUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating Host: %s", d.Id())
	client, err := meta.(*Config).Client()
	if err != nil {
		return err
	}

	fqdn := d.Get("fqdn").(string)
	description := d.Get("description").(string)
	random := d.Get("random").(bool)
	userpassword := d.Get("userpassword").(string)

	optArgs := ipa.HostModOptionalArgs{
		Description: &description,
		Random:      &random,
	}

	if userpassword != "" {
		optArgs.Userpassword = &userpassword
	}

	res, err := client.HostMod(
		&ipa.HostModArgs{
			Fqdn: fqdn,
		},
		&optArgs,
	)
	if err != nil {
		return err
	}

	// randompassword is not returned by HostShow
	if d.Get("random").(bool) {
		d.Set("randompassword", *res.Result.Randompassword)
	}

	return resourceFreeIPAHostRead(d, meta)
}

func resourceFreeIPAHostRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Refreshing Host: %s", d.Id())
	client, err := meta.(*Config).Client()
	if err != nil {
		return err
	}

	fqdn := d.Get("fqdn").(string)

	res, err := client.HostShow(
		&ipa.HostShowArgs{
			Fqdn: fqdn,
		},
		&ipa.HostShowOptionalArgs{},
	)
	if err != nil {
		return err
	}

	if res.Result.Description != nil {
		d.Set("description", *res.Result.Description)
	}
	if res.Result.Userpassword != nil {
		d.Set("userpassword", *res.Result.Userpassword)
	}

	return nil
}

func resourceFreeIPAHostDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting Host: %s", d.Id())
	client, err := meta.(*Config).Client()
	if err != nil {
		return err
	}

	fqdn := d.Get("fqdn").(string)

	_, err = client.HostDel(
		&ipa.HostDelArgs{
			Fqdn: []string{fqdn},
		},
		&ipa.HostDelOptionalArgs{},
	)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func resourceFreeIPAHostImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.SetId(d.Id())
	d.Set("fqdn", d.Id())

	err := resourceFreeIPAHostRead(d, meta)
	if err != nil {
		return []*schema.ResourceData{}, err
	}

	return []*schema.ResourceData{d}, nil
}
