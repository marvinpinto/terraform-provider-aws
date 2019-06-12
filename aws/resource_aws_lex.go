package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lexmodelbuildingservice"
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
