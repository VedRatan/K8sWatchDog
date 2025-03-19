package gemini

import "strings"

// Helper function to extract YAML from the Gemini response
func extractYAMLFromResponse(response string) string {
	// Define the start and end markers for the YAML block
	startMarker := "```yaml"
	endMarker := "```"

	// Find the start and end positions of the YAML block
	startIndex := strings.Index(response, startMarker)
	if startIndex == -1 {
		return ""
	}
	startIndex += len(startMarker)

	// Finding the end index making startIndex as the starting point for search (which will be effective 0th index for search)
	endIndex := strings.Index(response[startIndex:], endMarker)
	if endIndex == -1 {
		return ""
	}
	// adding startIndex to endIndex to make endIndex reference according to original start position of the response
	endIndex += startIndex

	// Extract the YAML content
	yamlContent := response[startIndex:endIndex]

	// Trim any leading or trailing whitespace
	yamlContent = strings.TrimSpace(yamlContent)

	return yamlContent
}
