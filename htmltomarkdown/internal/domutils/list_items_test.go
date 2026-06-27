package domutils

import (
	"context"
	"testing"

	"github.com/yudaprama/tools/htmltomarkdown/internal/tester"
)

func TestMoveListItems(t *testing.T) {
	runs := []struct {
		desc     string
		input    string
		expected string
	}{
		{
			desc:  "not needed in normal list",
			input: "<div><ul><li>A</li><li>B</li><li>C</li></ul></div>",
			expected: `
├─body
│ ├─div
│ │ ├─ul
│ │ │ ├─li
│ │ │ │ ├─#text "A"
│ │ │ ├─li
│ │ │ │ ├─#text "B"
│ │ │ ├─li
│ │ │ │ ├─#text "C"
			`,
		},
		{
			desc:  "#text moves into the previous li",
			input: "<ul><li>A</li>B</ul>",
			expected: `
├─body
│ ├─ul
│ │ ├─li
│ │ │ ├─#text "A"
│ │ │ ├─#text "B"
			`,
		},
		{
			desc:  "div moves into the previous li",
			input: "<ul><li>A</li><div>B</div></ul>",
			expected: `
├─body
│ ├─ul
│ │ ├─li
│ │ │ ├─#text "A"
│ │ │ ├─div
│ │ │ │ ├─#text "B"
			`,
		},
		{
			desc:  "ol moves into the previous li",
			input: "<ul><li>A</li><ol><li>B</li></ol></ul>",
			expected: `
├─body
│ ├─ul
│ │ ├─li
│ │ │ ├─#text "A"
│ │ │ ├─ol
│ │ │ │ ├─li
│ │ │ │ │ ├─#text "B"
			`,
		},
		{
			desc:  "no existing li",
			input: "<ul><span>A</span><span>B</span></ul>",
			expected: `
├─body
│ ├─ul
│ │ ├─li
│ │ │ ├─span
│ │ │ │ ├─#text "A"
│ │ │ ├─span
│ │ │ │ ├─#text "B"
			`,
		},
		{
			desc: "basic moved list",
			input: `
<ol>
	<li>One</li>
	<li>Two</li>
	<ol>
		<li>Two point one</li>
		<li>Two point two</li>
	</ol>
</ol>
			`,
			expected: `
├─body
│ ├─ol
│ │ ├─#text "\n\t"
│ │ ├─li
│ │ │ ├─#text "One"
│ │ ├─#text "\n\t"
│ │ ├─li
│ │ │ ├─#text "Two"
│ │ │ ├─ol
│ │ │ │ ├─#text "\n\t\t"
│ │ │ │ ├─li
│ │ │ │ │ ├─#text "Two point one"
│ │ │ │ ├─#text "\n\t\t"
│ │ │ │ ├─li
│ │ │ │ │ ├─#text "Two point two"
│ │ │ │ ├─#text "\n\t"
│ │ ├─#text "\n\t"
│ │ ├─#text "\n"
			`,
		},
	}
	for _, run := range runs {
		t.Run(run.desc, func(t *testing.T) {
			doc := tester.Parse(t, run.input, "")

			MoveListItems(context.TODO(), doc)

			tester.ExpectRepresentation(t, doc, "output", run.expected)
		})
	}
}
