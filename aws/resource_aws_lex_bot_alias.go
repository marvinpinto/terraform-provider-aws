package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lexmodelbuildingservice"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsLexBotAlias() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLexBotAliasCreate,
		Read:   resourceAwsLexBotAliasRead,
		Update: resourceAwsLexBotAliasUpdate,
		Delete: resourceAwsLexBotAliasDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsLexBotAliasImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Update: schema.DefaultTimeout(time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"bot_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(2, 50),
					validation.StringMatch(regexp.MustCompile(`^([A-Za-z]_?)+$`), ""),
				),
			},
			"bot_version": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 64),
					validation.StringMatch(regexp.MustCompile(`\$LATEST|[0-9]+`), ""),
				),
			},
			"checksum": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.StringLenBetween(0, 200),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 100),
					validation.StringMatch(regexp.MustCompile(`^([A-Za-z]_?)+$`), ""),
				),
			},
		},
	}
}

func resourceAwsLexBotAliasCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lexmodelconn
	name := d.Get("name").(string)

	input := &lexmodelbuildingservice.PutBotAliasInput{
		BotName:    aws.String(d.Get("bot_name").(string)),
		BotVersion: aws.String(d.Get("bot_version").(string)),
		Name:       aws.String(name),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if _, err := conn.PutBotAlias(input); err != nil {
		return fmt.Errorf("error creating bot alias %s: %s", name, err)
	}

	d.SetId(name)

	return resourceAwsLexBotAliasRead(d, meta)
}

func resourceAwsLexBotAliasRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lexmodelconn

	resp, err := conn.GetBotAlias(&lexmodelbuildingservice.GetBotAliasInput{
		BotName: aws.String(d.Get("bot_name").(string)),
		Name:    aws.String(d.Id()),
	})
	if isAWSErr(err, lexmodelbuildingservice.ErrCodeNotFoundException, "") {
		log.Printf("[WARN] Bot alias (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error getting bot alias %s: %s", d.Id(), err)
	}

	d.Set("bot_name", resp.BotName)
	d.Set("bot_version", resp.BotVersion)
	d.Set("checksum", resp.Checksum)
	d.Set("description", resp.Description)
	d.Set("name", resp.Name)

	return nil
}

func resourceAwsLexBotAliasUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lexmodelconn

	input := &lexmodelbuildingservice.PutBotAliasInput{
		BotName:    aws.String(d.Get("bot_name").(string)),
		BotVersion: aws.String(d.Get("bot_version").(string)),
		Checksum:   aws.String(d.Get("checksum").(string)),
		Name:       aws.String(d.Id()),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	err := resource.Retry(d.Timeout(schema.TimeoutUpdate), func() *resource.RetryError {
		_, err := conn.PutBotAlias(input)

		if isAWSErr(err, lexmodelbuildingservice.ErrCodeConflictException, "") {
			return resource.RetryableError(fmt.Errorf("%q: bot alias still updating", d.Id()))
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error updating bot alias %s: %s", d.Id(), err)
	}

	return resourceAwsLexBotAliasRead(d, meta)
}

func resourceAwsLexBotAliasDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lexmodelconn

	botName := d.Get("bot_name").(string)
	name := d.Get("name").(string)

	err := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		_, err := conn.DeleteBotAlias(&lexmodelbuildingservice.DeleteBotAliasInput{
			BotName: aws.String(botName),
			Name:    aws.String(name),
		})

		if isAWSErr(err, lexmodelbuildingservice.ErrCodeConflictException, "") {
			return resource.RetryableError(fmt.Errorf("%q: bot alias still deleting", d.Id()))
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error deleting bot alias %s: %s", d.Id(), err)
	}

	// Ensure the bot alias is actually deleted before moving on. This avoids issues with deleting
	// bots that have associated bot aliases.

	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		_, err := conn.GetBotAlias(&lexmodelbuildingservice.GetBotAliasInput{
			BotName: aws.String(botName),
			Name:    aws.String(name),
		})
		if err != nil {
			if isAWSErr(err, lexmodelbuildingservice.ErrCodeNotFoundException, "") {
				return nil
			}

			return resource.NonRetryableError(err)
		}

		return nil
	})
}

func resourceAwsLexBotAliasImport(d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid Lex Bot Alias resource id, expected BOT_NAME.BOT_ALIAS_NAME")
	}

	d.SetId(parts[1])
	d.Set("bot_name", parts[0])
	d.Set("name", parts[1])

	return []*schema.ResourceData{d}, nil
}
