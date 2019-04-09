package attribute

import (
	"encoding/json"
	"fmt"
	"github.com/open-horizon/anax/api"
	"github.com/open-horizon/anax/cli/cliutils"
	"github.com/open-horizon/anax/persistence"
)

const HTTPSBasicAuthAttributes = "HTTPSBasicAuthAttributes"

// Our form of the attributes output
type OurAttributes struct {
	Type         string                   `json:"type"`
	Label        string                   `json:"label"`
	ServiceSpecs persistence.ServiceSpecs `json:"service_specs,omitempty"`
	Variables    map[string]interface{}   `json:"variables"`
}

func List() {
	// Get the attributes
	apiOutput := map[string][]api.Attribute{}
	httpCode, _ := cliutils.HorizonGet("attribute", []int{200, cliutils.ANAX_NOT_CONFIGURED_YET}, &apiOutput, false)
	if httpCode == cliutils.ANAX_NOT_CONFIGURED_YET {
		cliutils.Fatal(cliutils.HTTP_ERROR, cliutils.MUST_REGISTER_FIRST)
	}
	var ok bool
	if _, ok = apiOutput["attributes"]; !ok {
		cliutils.Fatal(cliutils.HTTP_ERROR, "horizon api attributes output did not include 'attributes' key")
	}
	apiAttrs := apiOutput["attributes"]

	// Only include interesting fields in our output
	attrs := []OurAttributes{}
	for _, a := range apiAttrs {
		if a.ServiceSpecs == nil {
			attrs = append(attrs, OurAttributes{Type: *a.Type, Label: *a.Label, Variables: *a.Mappings})
		} else {
			attrs = append(attrs, OurAttributes{Type: *a.Type, Label: *a.Label, ServiceSpecs: *a.ServiceSpecs, Variables: *a.Mappings})
		}
	}

	// Convert to json and output
	jsonBytes, err := json.MarshalIndent(attrs, "", cliutils.JSON_INDENT)
	if err != nil {
		cliutils.Fatal(cliutils.JSON_PARSING_ERROR, "failed to marshal 'hzn attribute list' output: %v", err)
	}
	fmt.Printf("%s\n", jsonBytes)
}
