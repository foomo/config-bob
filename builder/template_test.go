package builder

import (
	"fmt"
	"os"
	"testing"

	"github.com/foomo/config-bob/vaultdummy"
)

func TestTemplateFuncs(t *testing.T) {
	ts := vaultdummy.DummyVaultServerSecretEcho()
	defer ts.Close()

	data := map[string]interface{}{
		"hello": "test",
		"nested": map[string]string{
			"foo": "bar",
		},
	}
	runTemplate := func(templ string) string {
		result, err := process("", templ, data)
		if err != nil {
			t.Fatal("could not process template", t, err)
		}
		return string(result)
	}
	assert := func(templ, expected string) {
		result := runTemplate(templ)
		if result != expected {
			t.Fatal(fmt.Sprintf("expected %q got %q", expected, result))
		}
	}
	assert(`{{ yaml (secret "path/to/hello.token") }}`, "well-a-token")
	assert(`{{ json (yaml (secret "path/to/hello.escape")) }}`, "\"muha\\\"haha\"")

	const (
		testEnvName  = "THIS_IS_JUST_A_TEST"
		testEnvValue = "a test value"
	)
	os.Setenv(testEnvName, testEnvValue)
	assert("{{ env \""+testEnvName+"\" }}", testEnvValue)

	assert(`{{ json .nested }}`, `{"foo":"bar"}`)
	assert(`{{ json . }}`, `{"hello":"test","nested":{"foo":"bar"}}`)

	assert(`{{ jsonindent . "  " "  " }}`, `{
    "hello": "test",
    "nested": {
      "foo": "bar"
    }
  }`)

	//assert(`{{ js . }}`, `{"hello":"test","nested":{"foo":"bar"}}`)
	const (
		yamlIndentTemplate = `foo:
{{ indent (yaml .) "  " }}
`
		yamlIndentExpected = `foo:
  hello: test
  nested:
    foo: bar
`
	)

	assert(yamlIndentTemplate, yamlIndentExpected)
}
