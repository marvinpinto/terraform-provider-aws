package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lexmodelbuildingservice"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

// Many of the Lex resources require complex nested objects. Terraform maps only support simple key
// value pairs and not complex or mixed types. That is why these resources are defined using the
// schema.TypeList and a max of 1 item instead of the schema.TypeMap.

// Convert a slice of items to a map[string]interface{}
// Expects input as a single item slice.
// Required because we use TypeList instead of TypeMap due to TypeMap not supporting nested and mixed complex values.
func expandLexObject(v interface{}) map[string]interface{} {
	return v.([]interface{})[0].(map[string]interface{})
}

// Covert a map[string]interface{} to a slice of items
// Expects a single map[string]interface{}
// Required because we use TypeList instead of TypeMap due to TypeMap not supporting nested and mixed complex values.
func flattenLexObject(m map[string]interface{}) []map[string]interface{} {
	return []map[string]interface{}{m}
}

func expandLexSet(s *schema.Set) (items []map[string]interface{}) {
	for _, rawItem := range s.List() {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			continue
		}

		items = append(items, item)
	}

	return
}

var lexMessageResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"content": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringLenBetween(1, 1000),
		},
		"content_type": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: validation.StringInSlice([]string{
				lexmodelbuildingservice.ContentTypeCustomPayload,
				lexmodelbuildingservice.ContentTypePlainText,
				lexmodelbuildingservice.ContentTypeSsml,
			}, false),
		},
		"group_number": {
			Type:         schema.TypeInt,
			Optional:     true,
			ValidateFunc: validation.IntBetween(1, 5),
		},
	},
}

func flattenLexMessages(messages []*lexmodelbuildingservice.Message) (flattenedMessages []map[string]interface{}) {
	for _, message := range messages {
		flattenedMessages = append(flattenedMessages, map[string]interface{}{
			"content":      aws.StringValue(message.Content),
			"content_type": aws.StringValue(message.ContentType),
			"group_number": aws.Int64Value(message.GroupNumber),
		})
	}

	return
}

// Expects a slice of maps representing the Lex objects.
// The value passed into this function should have been run through the expandLexSet function.
// Example: []map[content: test content_type: PlainText group_number: 1]
func expandLexMessages(rawValues []map[string]interface{}) (messages []*lexmodelbuildingservice.Message) {
	for _, rawValue := range rawValues {
		message := &lexmodelbuildingservice.Message{
			Content:     aws.String(rawValue["content"].(string)),
			ContentType: aws.String(rawValue["content_type"].(string)),
		}

		if v, ok := rawValue["group_number"]; ok && v != 0 {
			message.GroupNumber = aws.Int64(int64(v.(int)))
		}

		messages = append(messages, message)
	}

	return
}

var lexStatementResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"message": {
			Type:     schema.TypeSet,
			Required: true,
			MinItems: 1,
			MaxItems: 15,
			Elem:     lexMessageResource,
		},
		"response_card": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringLenBetween(1, 50000),
		},
	},
}

func flattenLexStatement(statement *lexmodelbuildingservice.Statement) (flattened map[string]interface{}) {
	flattened = map[string]interface{}{}
	flattened["message"] = flattenLexMessages(statement.Messages)

	if statement.ResponseCard != nil {
		flattened["response_card"] = aws.StringValue(statement.ResponseCard)
	}

	return
}

func expandLexStatement(m map[string]interface{}) (statement *lexmodelbuildingservice.Statement) {
	statement = &lexmodelbuildingservice.Statement{}
	statement.Messages = expandLexMessages(expandLexSet(m["message"].(*schema.Set)))

	if v, ok := m["response_card"]; ok && v != "" {
		statement.ResponseCard = aws.String(v.(string))
	}

	return
}

var lexPromptResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"max_attempts": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntBetween(1, 5),
		},
		"message": {
			Type:     schema.TypeSet,
			Required: true,
			MinItems: 1,
			MaxItems: 15,
			Elem:     lexMessageResource,
		},
		"response_card": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringLenBetween(1, 50000),
		},
	},
}

func flattenLexPrompt(prompt *lexmodelbuildingservice.Prompt) (flattened map[string]interface{}) {
	flattened = map[string]interface{}{}
	flattened["max_attempts"] = aws.Int64Value(prompt.MaxAttempts)
	flattened["message"] = flattenLexMessages(prompt.Messages)

	if prompt.ResponseCard != nil {
		flattened["response_card"] = aws.StringValue(prompt.ResponseCard)
	}

	return
}

func expandLexPrompt(m map[string]interface{}) (prompt *lexmodelbuildingservice.Prompt) {
	prompt = &lexmodelbuildingservice.Prompt{}
	prompt.MaxAttempts = aws.Int64(int64(m["max_attempts"].(int)))
	prompt.Messages = expandLexMessages(expandLexSet(m["message"].(*schema.Set)))

	if v, ok := m["response_card"]; ok && v != "" {
		prompt.ResponseCard = aws.String(v.(string))
	}

	return
}

func flattenLexIntents(intents []*lexmodelbuildingservice.Intent) (flattenedIntents []map[string]interface{}) {
	for _, intent := range intents {
		flattenedIntents = append(flattenedIntents, map[string]interface{}{
			"intent_name":    aws.StringValue(intent.IntentName),
			"intent_version": aws.StringValue(intent.IntentVersion),
		})
	}

	return
}

// Expects a slice of maps representing the Lex objects.
// The value passed into this function should have been run through the expandLexSet function.
// Example: []map[intent_name: OrderFlowers intent_version: $LATEST]
func expandLexIntents(rawValues []map[string]interface{}) (intents []*lexmodelbuildingservice.Intent) {
	for _, rawValue := range rawValues {
		intents = append(intents, &lexmodelbuildingservice.Intent{
			IntentName:    aws.String(rawValue["intent_name"].(string)),
			IntentVersion: aws.String(rawValue["intent_version"].(string)),
		})
	}

	return
}

func resourceAwsLexBot() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLexBotCreate,
		Read:   resourceAwsLexBotRead,
		Update: resourceAwsLexBotUpdate,
		Delete: resourceAwsLexBotDelete,

		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
				// The version is not required for import but it is required for the get request.
				d.Set("version", "$LATEST")
				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Update: schema.DefaultTimeout(time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"abort_statement": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				MaxItems: 1,
				Elem:     lexStatementResource,
			},
			"checksum": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"child_directed": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"clarification_prompt": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				MaxItems: 1,
				Elem:     lexPromptResource,
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.StringLenBetween(0, 200),
			},
			"failure_reason": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"idle_session_ttl_in_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      300,
				ValidateFunc: validation.IntBetween(60, 86400),
			},
			"intent": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				MaxItems: 100,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"intent_name": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.All(
								validation.StringLenBetween(1, 100),
								validation.StringMatch(regexp.MustCompile(`^([A-Za-z]_?)+$`), ""),
							),
						},
						"intent_version": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.All(
								validation.StringLenBetween(1, 64),
								validation.StringMatch(regexp.MustCompile(`\$LATEST|[0-9]+`), ""),
							),
						},
					},
				},
			},
			"locale": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  lexmodelbuildingservice.LocaleEnUs,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(2, 50),
					validation.StringMatch(regexp.MustCompile(`^([A-Za-z]_?)+$`), ""),
				),
			},
			"process_behavior": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  lexmodelbuildingservice.ProcessBehaviorSave,
				ValidateFunc: validation.StringInSlice([]string{
					lexmodelbuildingservice.ProcessBehaviorBuild,
					lexmodelbuildingservice.ProcessBehaviorSave,
				}, false),
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"version": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "$LATEST",
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 64),
					validation.StringMatch(regexp.MustCompile(`\$LATEST|[0-9]+`), ""),
				),
			},
			"voice_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsLexBotCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lexmodelconn
	name := d.Get("name").(string)

	input := &lexmodelbuildingservice.PutBotInput{
		AbortStatement:          expandLexStatement(expandLexObject(d.Get("abort_statement"))),
		ChildDirected:           aws.Bool(d.Get("child_directed").(bool)),
		ClarificationPrompt:     expandLexPrompt(expandLexObject(d.Get("clarification_prompt"))),
		IdleSessionTTLInSeconds: aws.Int64(int64(d.Get("idle_session_ttl_in_seconds").(int))),
		Intents:                 expandLexIntents(expandLexSet(d.Get("intent").(*schema.Set))),
		Locale:                  aws.String(d.Get("locale").(string)),
		Name:                    aws.String(name),
		ProcessBehavior:         aws.String(d.Get("process_behavior").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("voice_id"); ok {
		input.VoiceId = aws.String(v.(string))
	}

	if _, err := conn.PutBot(input); err != nil {
		return fmt.Errorf("error creating bot %s: %s", name, err)
	}

	d.SetId(name)

	return resourceAwsLexBotRead(d, meta)
}

func resourceAwsLexBotRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lexmodelconn

	resp, err := conn.GetBot(&lexmodelbuildingservice.GetBotInput{
		Name:           aws.String(d.Id()),
		VersionOrAlias: aws.String(d.Get("version").(string)),
	})
	if isAWSErr(err, lexmodelbuildingservice.ErrCodeNotFoundException, "") {
		log.Printf("[WARN] Bot (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error getting bot: %s", err)
	}

	// Process behavior is not returned from the API but is used for create and update.
	// Manually write to state file to avoid un-expected diffs.
	processBehavior := lexmodelbuildingservice.ProcessBehaviorSave
	if v, ok := d.GetOk("process_behavior"); ok {
		processBehavior = v.(string)
	}

	d.Set("abort_statement", flattenLexObject(flattenLexStatement(resp.AbortStatement)))
	d.Set("checksum", resp.Checksum)
	d.Set("child_directed", resp.ChildDirected)
	d.Set("clarification_prompt", flattenLexObject(flattenLexPrompt(resp.ClarificationPrompt)))
	d.Set("description", resp.Description)
	d.Set("failure_reason", resp.FailureReason)
	d.Set("idle_session_ttl_in_seconds", resp.IdleSessionTTLInSeconds)
	d.Set("intent", flattenLexIntents(resp.Intents))
	d.Set("locale", resp.Locale)
	d.Set("name", resp.Name)
	d.Set("process_behavior", processBehavior)
	d.Set("status", resp.Status)
	d.Set("version", resp.Version)

	if resp.VoiceId != nil {
		d.Set("voice_id", resp.VoiceId)
	}

	return nil
}

func resourceAwsLexBotUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lexmodelconn

	input := &lexmodelbuildingservice.PutBotInput{
		AbortStatement:          expandLexStatement(expandLexObject(d.Get("abort_statement"))),
		Checksum:                aws.String(d.Get("checksum").(string)),
		ChildDirected:           aws.Bool(d.Get("child_directed").(bool)),
		ClarificationPrompt:     expandLexPrompt(expandLexObject(d.Get("clarification_prompt"))),
		IdleSessionTTLInSeconds: aws.Int64(int64(d.Get("idle_session_ttl_in_seconds").(int))),
		Intents:                 expandLexIntents(expandLexSet(d.Get("intent").(*schema.Set))),
		Locale:                  aws.String(d.Get("locale").(string)),
		Name:                    aws.String(d.Id()),
		ProcessBehavior:         aws.String(d.Get("process_behavior").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("voice_id"); ok {
		input.VoiceId = aws.String(v.(string))
	}

	err := resource.Retry(d.Timeout(schema.TimeoutUpdate), func() *resource.RetryError {
		_, err := conn.PutBot(input)

		if isAWSErr(err, lexmodelbuildingservice.ErrCodeConflictException, "") {
			return resource.RetryableError(fmt.Errorf("%q: bot still updating", d.Id()))
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error updating bot %s: %s", d.Id(), err)
	}

	return resourceAwsLexBotRead(d, meta)
}

func resourceAwsLexBotDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lexmodelconn

	err := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		_, err := conn.DeleteBot(&lexmodelbuildingservice.DeleteBotInput{
			Name: aws.String(d.Id()),
		})

		if isAWSErr(err, lexmodelbuildingservice.ErrCodeConflictException, "") {
			return resource.RetryableError(fmt.Errorf("%q: bot still deleting", d.Id()))
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error deleting bot %s: %s", d.Id(), err)
	}

	return nil
}
