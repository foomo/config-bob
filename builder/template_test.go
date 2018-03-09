package builder

import (
	"fmt"
	"os"
	"testing"
)

func TestMissingError(t *testing.T) {
	_, err := process("", "{{.foo}}", map[string]interface{}{})
	if err == nil {
		t.Fatal("missing keys are not an option")
	}
}
func renderTemplate(templ string, data map[string]interface{}) (string, error) {
	result, err := process("", templ, data)
	return string(result), err
}

func TestTemplateFuncs(t *testing.T) {
	data := map[string]interface{}{
		"hello": "test",
		"nested": map[string]string{
			"foo": "bar",
		},
	}
	assert := func(templ, expected string) {
		result, err := renderTemplate(templ, data)
		if err != nil {
			t.Fatal("could not process template", t, err)
		}
		if result != expected {
			t.Fatal(fmt.Sprintf("expected %q got %q", expected, result))
		}
	}
	assertErr := func(templ string) {
		_, err := renderTemplate(templ, data)
		if err == nil {
			t.Fatal("that sould have been an error")
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

	// .hello contains "test"
	assert(`{{ substr .hello ":2"}}`, `te`)
	assert(`{{ substr .hello "1:"}}`, `est`)
	assert(`{{ substr .hello "0:"}}`, `test`)
	assert(`{{ substr .hello ":0"}}`, ``)
	assert(`{{ substr .hello "1:2"}}`, `e`)
	assert(`{{ substr .hello "1:1"}}`, ``)

	assertErr(`{{ substr .hello "1:10"}}`)
	assertErr(`{{ substr .hello "-1:1"}}`)
	assertErr(`{{ substr .hello ":-1"}}`)

}

func TestTemplateReplace(t *testing.T) {
	data := map[string]interface{}{"data": "test\ntest"}
	template := `{{ replace "\n" " " .data}}`
	content, _ := renderTemplate(template, data)
	if content != "test test" {
		t.Fatal("Template didn't work")
	}
}

func TestTemplateReplaceChaining(t *testing.T) {
	data := map[string]interface{}{"data": "a-test-test-a"}
	template := `{{ substr .data "2:11" | replace "-" " "}}`
	content, err := renderTemplate(template, data)
	if err != nil {
		t.Fatal("Template error occurred", err)
	}
	if content != "test test" {
		t.Fatal("Template didn't work")
	}
}
