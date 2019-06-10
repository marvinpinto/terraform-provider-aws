package aws

// Amazon Lex Resource Constants. Data models are documented here
// https://docs.aws.amazon.com/lex/latest/dg/API_Types_Amazon_Lex_Model_Building_Service.html

const (

	// General

	lexNameMinLength = 1
	lexNameMaxLength = 100
	lexNameRegex     = "^([A-Za-z]_?)+$"

	lexVersionMinLength = 1
	lexVersionMaxLength = 64
	lexVersionRegex     = "\\$LATEST|[0-9]+"
	lexVersionLatest    = "$LATEST"

	lexDescriptionMinLength = 0
	lexDescriptionMaxLength = 200

	// Bot

	lexBotNameMinLength = 2
	lexBotNameMaxLength = 50

	// Bot Alias

	lexBotAliasDeleteRetryTimeoutMinutes = 5
)
