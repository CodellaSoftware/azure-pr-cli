package formatter

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildMinimalDocx(documentXML string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
			`</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
			`</Relationships>`,
		"word/_rels/document.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`,
		"word/document.xml": documentXML,
	}

	for name, content := range files {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(content))
	}
	_ = zw.Close()
	return buf.Bytes()
}

func writeTempDocx(t *testing.T, content []byte) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "template-*.docx")
	require.NoError(t, err)
	_, err = f.Write(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func simpleDocumentXML(templateRowContent string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body><w:tbl>` +
		`<w:tr><w:tc><w:p><w:r><w:t>Header</w:t></w:r></w:p></w:tc></w:tr>` +
		templateRowContent +
		`</w:tbl></w:body></w:document>`
}

func TestXmlEscape(t *testing.T) {
	cases := []struct{ input, want string }{
		{`Hello`, `Hello`},
		{`Fix <bug>`, `Fix &lt;bug&gt;`},
		{`a & b`, `a &amp; b`},
		{`say "hi"`, `say &quot;hi&quot;`},
		{`<a & "b">`, `&lt;a &amp; &quot;b&quot;&gt;`},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, xmlEscape(c.input), "input: %q", c.input)
	}
}

func TestXmlUnescape(t *testing.T) {
	cases := []struct{ input, want string }{
		{`Hello`, `Hello`},
		{`Fix &lt;bug&gt;`, `Fix <bug>`},
		{`a &amp; b`, `a & b`},
		{`say &quot;hi&quot;`, `say "hi"`},
		{`it&apos;s`, `it's`},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, xmlUnescape(c.input), "input: %q", c.input)
	}
}

func TestExtractTableRows_SingleRow(t *testing.T) {
	xml := `<w:tbl><w:tr><w:tc><w:t>cell</w:t></w:tc></w:tr></w:tbl>`
	rows := extractTableRows(xml)
	require.Len(t, rows, 1)
	assert.Equal(t, `<w:tr><w:tc><w:t>cell</w:t></w:tc></w:tr>`, rows[0].content)
	assert.Equal(t, strings.Index(xml, "<w:tr>"), rows[0].start)
}

func TestExtractTableRows_MultipleRows(t *testing.T) {
	xml := `<w:tbl>` +
		`<w:tr><w:tc><w:t>row1</w:t></w:tc></w:tr>` +
		`<w:tr><w:tc><w:t>row2</w:t></w:tc></w:tr>` +
		`<w:tr><w:tc><w:t>row3</w:t></w:tc></w:tr>` +
		`</w:tbl>`
	rows := extractTableRows(xml)
	require.Len(t, rows, 3)
	assert.Contains(t, rows[0].content, "row1")
	assert.Contains(t, rows[1].content, "row2")
	assert.Contains(t, rows[2].content, "row3")
}

func TestExtractTableRows_IgnoresWtrPr(t *testing.T) {
	xml := `<w:tbl><w:tr><w:trPr><w:cantSplit/></w:trPr><w:tc><w:t>data</w:t></w:tc></w:tr></w:tbl>`
	rows := extractTableRows(xml)
	require.Len(t, rows, 1)
	assert.Contains(t, rows[0].content, "data")
}

func TestExtractTableRows_Empty(t *testing.T) {
	rows := extractTableRows(`<w:tbl></w:tbl>`)
	assert.Len(t, rows, 0)
}

func TestNormalizeRuns_NoPlaceholder(t *testing.T) {
	input := `<w:tr><w:tc><w:p><w:r><w:t>hello</w:t></w:r><w:r><w:t> world</w:t></w:r></w:p></w:tc></w:tr>`
	result := normalizeRuns(input)
	assert.Equal(t, input, result)
}

func TestNormalizeRuns_SplitPlaceholder(t *testing.T) {
	input := `<w:tr><w:tc><w:p>` +
		`<w:r><w:t>{{ti</w:t></w:r>` +
		`<w:r><w:t>tle}}</w:t></w:r>` +
		`</w:p></w:tc></w:tr>`
	result := normalizeRuns(input)

	assert.Contains(t, result, "{{title}}")
	assert.Equal(t, 1, strings.Count(result, "<w:r>"))
}

func TestNormalizeRuns_PreservesRunProps(t *testing.T) {
	input := `<w:tr><w:tc><w:p>` +
		`<w:r><w:rPr><w:b/></w:rPr><w:t>{{ti</w:t></w:r>` +
		`<w:r><w:t>tle}}</w:t></w:r>` +
		`</w:p></w:tc></w:tr>`
	result := normalizeRuns(input)

	assert.Contains(t, result, "<w:rPr><w:b/></w:rPr>")
	assert.Contains(t, result, "{{title}}")
}

func TestNormalizeRuns_CompletePlaceholderInOneRun(t *testing.T) {
	input := `<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`
	result := normalizeRuns(input)
	assert.Equal(t, input, result)
}

func TestNormalizeRuns_PreservesFormattingWhenNoSplit(t *testing.T) {
	input := `<w:tr><w:tc><w:p>` +
		`<w:r><w:rPr><w:b/></w:rPr><w:t>Label: </w:t></w:r>` +
		`<w:r><w:t>{{title}}</w:t></w:r>` +
		`</w:p></w:tc></w:tr>`
	result := normalizeRuns(input)
	assert.Equal(t, input, result)
	assert.Equal(t, 2, strings.Count(result, "<w:r>"), "both runs must be preserved")
}

func TestNormalizeRuns_MultipleFieldsInRow(t *testing.T) {
	input := `<w:tr>` +
		`<w:tc><w:p><w:r><w:t>{{ti</w:t></w:r><w:r><w:t>tle}}</w:t></w:r></w:p></w:tc>` +
		`<w:tc><w:p><w:r><w:t>{{au</w:t></w:r><w:r><w:t>thor}}</w:t></w:r></w:p></w:tc>` +
		`</w:tr>`
	result := normalizeRuns(input)
	assert.Contains(t, result, "{{title}}")
	assert.Contains(t, result, "{{author}}")
}

func testPR() models.PullRequest {
	return models.PullRequest{
		ID:            42,
		Title:         "My PR title",
		Status:        "completed",
		CreatedBy:     models.User{DisplayName: "Jane Doe"},
		Repository:    models.Repository{Name: "my-repo"},
		SourceRefName: "refs/heads/feature/foo",
		TargetRefName: "refs/heads/main",
		MergeStatus:   "succeeded",
		ClosedDate:    time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
		CreationDate:  time.Date(2026, 2, 10, 9, 0, 0, 0, time.UTC),
	}
}

func TestApplyPlaceholders_KnownFields(t *testing.T) {
	pr := testPR()
	cases := []struct {
		placeholder string
		wantPart    string
	}{
		{`{{index}}`, "1"},
		{`{{id}}`, "42"},
		{`{{repo}}`, "my-repo"},
		{`{{title}}`, "My PR title"},
		{`{{author}}`, "Jane Doe"},
		{`{{status}}`, "completed"},
		{`{{source}}`, "feature/foo"},
		{`{{target}}`, "main"},
		{`{{merge_status}}`, "succeeded"},
	}
	for _, c := range cases {
		rowXML := `<w:tr><w:tc><w:p><w:r><w:t>` + c.placeholder + `</w:t></w:r></w:p></w:tc></w:tr>`
		result := applyPlaceholders(rowXML, pr, 1, "02.01.2006", "myorg", "myproject")
		assert.Contains(t, result, c.wantPart, "placeholder %s", c.placeholder)
		assert.NotContains(t, result, c.placeholder, "placeholder %s should be replaced", c.placeholder)
	}
}

func TestApplyPlaceholders_CompletedDate(t *testing.T) {
	pr := testPR()
	rowXML := `<w:tr><w:tc><w:p><w:r><w:t>{{completed}}</w:t></w:r></w:p></w:tc></w:tr>`
	result := applyPlaceholders(rowXML, pr, 1, "02.01.2006", "", "")
	assert.Contains(t, result, "15.02.2026")
}

func TestApplyPlaceholders_CreatedDate(t *testing.T) {
	pr := testPR()
	rowXML := `<w:tr><w:tc><w:p><w:r><w:t>{{created}}</w:t></w:r></w:p></w:tc></w:tr>`
	result := applyPlaceholders(rowXML, pr, 1, "02.01.2006", "", "")
	assert.Contains(t, result, "10.02.2026")
}

func TestApplyPlaceholders_XMLEscape(t *testing.T) {
	pr := testPR()
	pr.Title = `Fix <bug> & issue "now"`
	rowXML := `<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`
	result := applyPlaceholders(rowXML, pr, 1, "", "", "")
	assert.Contains(t, result, `Fix &lt;bug&gt; &amp; issue &quot;now&quot;`)
}

func TestApplyPlaceholders_UnknownField(t *testing.T) {
	pr := testPR()
	rowXML := `<w:tr><w:tc><w:p><w:r><w:t>{{unknown_xyz}}</w:t></w:r></w:p></w:tc></w:tr>`
	result := applyPlaceholders(rowXML, pr, 1, "", "", "")
	assert.Contains(t, result, "{{unknown_xyz}}")
}

func TestExpandTemplateRows_Basic(t *testing.T) {
	xmlStr := simpleDocumentXML(
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`,
	)
	prs := createTestPRs()

	result, _, err := expandTemplateRows(xmlStr, prs, "", "org", "project", 1)
	require.NoError(t, err)

	assert.Contains(t, result, "Add new feature")
	assert.Contains(t, result, "Fix bug in login")
	assert.NotContains(t, result, "{{title}}")
	assert.Contains(t, result, "Header")
}

func TestExpandTemplateRows_NoTemplateRow(t *testing.T) {
	xmlStr := simpleDocumentXML(
		`<w:tr><w:tc><w:p><w:r><w:t>plain row</w:t></w:r></w:p></w:tc></w:tr>`,
	)
	_, _, err := expandTemplateRows(xmlStr, createTestPRs(), "", "", "", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no template row found")
}

func TestExpandTemplateRows_EmptyPRList(t *testing.T) {
	xmlStr := simpleDocumentXML(
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`,
	)
	result, _, err := expandTemplateRows(xmlStr, []models.PullRequest{}, "", "", "", 1)
	require.NoError(t, err)
	assert.Contains(t, result, "Header")
	assert.NotContains(t, result, "{{title}}")
}

func TestExpandTemplateRows_CorrectRowCount(t *testing.T) {
	xmlStr := simpleDocumentXML(
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`,
	)
	prs := []models.PullRequest{
		{ID: 1, Title: "PR One"},
		{ID: 2, Title: "PR Two"},
		{ID: 3, Title: "PR Three"},
	}
	result, _, err := expandTemplateRows(xmlStr, prs, "", "", "", 1)
	require.NoError(t, err)

	assert.Equal(t, 4, strings.Count(result, "<w:tr>"))
	assert.Contains(t, result, "PR One")
	assert.Contains(t, result, "PR Two")
	assert.Contains(t, result, "PR Three")
}

func TestReplaceDocumentPlaceholders_Basic(t *testing.T) {
	xmlStr := `<w:p><w:r><w:t>{{dateFrom}}</w:t></w:r></w:p>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{
		"dateFrom": "01.02.2026",
	})
	assert.Contains(t, result, "01.02.2026")
	assert.NotContains(t, result, "{{dateFrom}}")
}

func TestReplaceDocumentPlaceholders_MultipleVars(t *testing.T) {
	xmlStr := `<w:body>` +
		`<w:p><w:r><w:t>{{dateFrom}}</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>{{dateTo}}</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>{{reportCreationDate}}</w:t></w:r></w:p>` +
		`</w:body>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{
		"dateFrom":           "01.02.2026",
		"dateTo":             "28.02.2026",
		"reportCreationDate": "28.02.2026",
	})
	assert.Contains(t, result, "01.02.2026")
	assert.Contains(t, result, "28.02.2026")
	assert.NotContains(t, result, "{{dateFrom}}")
	assert.NotContains(t, result, "{{dateTo}}")
	assert.NotContains(t, result, "{{reportCreationDate}}")
}

func TestReplaceDocumentPlaceholders_SplitRun(t *testing.T) {
	xmlStr := `<w:p>` +
		`<w:r><w:t>{{date</w:t></w:r>` +
		`<w:r><w:t>From}}</w:t></w:r>` +
		`</w:p>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{
		"dateFrom": "01.02.2026",
	})
	assert.Contains(t, result, "01.02.2026")
	assert.NotContains(t, result, "{{dateFrom}}")
}

func TestReplaceDocumentPlaceholders_UnknownVarLeftIntact(t *testing.T) {
	xmlStr := `<w:p><w:r><w:t>{{unknown}}</w:t></w:r></w:p>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{
		"dateFrom": "01.02.2026",
	})
	assert.Contains(t, result, "{{unknown}}")
}

func TestReplaceDocumentPlaceholders_EmptyVars(t *testing.T) {
	xmlStr := `<w:p><w:r><w:t>{{dateFrom}}</w:t></w:r></w:p>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{})
	assert.Equal(t, xmlStr, result)
}

func TestReplaceDocumentPlaceholders_PreservesMixedFormatting(t *testing.T) {
	xmlStr := `<w:p>` +
		`<w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">Label: </w:t></w:r>` +
		`<w:r><w:t>{{dateFrom}}</w:t></w:r>` +
		`</w:p>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{
		"dateFrom": "01.02.2026",
	})
	assert.Contains(t, result, "01.02.2026")
	assert.NotContains(t, result, "{{dateFrom}}")
	assert.Contains(t, result, `<w:rPr><w:b/></w:rPr>`)
	assert.Equal(t, 2, strings.Count(result, "<w:r>"))
}

func TestReplaceDocumentPlaceholders_InternalOptionsNotLeaked(t *testing.T) {
	xmlStr := `<w:p><w:r><w:t>{{org}}</w:t></w:r></w:p>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{
		"dateFrom": "01.02.2026",
	})
	assert.Contains(t, result, "{{org}}")
}

func TestReplaceDocumentPlaceholders_XMLEscaping(t *testing.T) {
	xmlStr := `<w:p><w:r><w:t>{{dateFrom}}</w:t></w:r></w:p>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{
		"dateFrom": "01 & 02 <March>",
	})
	assert.Contains(t, result, "01 &amp; 02 &lt;March&gt;")
}

func TestDOCXFormatter_DocumentVarsOutsideTable(t *testing.T) {
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body>` +
		`<w:p><w:r><w:t>Period: {{dateFrom}} - {{dateTo}}</w:t></w:r></w:p>` +
		`<w:tbl>` +
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>` +
		`</w:tbl>` +
		`<w:p><w:r><w:t>Created: {{reportCreationDate}}</w:t></w:r></w:p>` +
		`</w:body></w:document>`
	path := writeTempDocx(t, buildMinimalDocx(docXML))

	f := NewDOCXFormatter()
	prs := []models.PullRequest{{ID: 1, Title: "My PR"}}
	output, err := f.Format(prs, "02.01.2006", map[string]string{
		"template":           path,
		"dateFrom":           "01.02.2026",
		"dateTo":             "28.02.2026",
		"reportCreationDate": "28.02.2026",
	})
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
	require.NoError(t, err)

	for _, zf := range zr.File {
		if zf.Name == "word/document.xml" {
			rc, _ := zf.Open()
			data, _ := io.ReadAll(rc)
			_ = rc.Close()
			content := string(data)

			assert.Contains(t, content, "01.02.2026")
			assert.Contains(t, content, "28.02.2026")
			assert.Contains(t, content, "My PR")
			assert.NotContains(t, content, "{{dateFrom}}")
			assert.NotContains(t, content, "{{dateTo}}")
			assert.NotContains(t, content, "{{reportCreationDate}}")
		}
	}
}

func TestDOCXFormatter_NoTemplatePath(t *testing.T) {
	f := NewDOCXFormatter()
	_, err := f.Format(createTestPRs(), "", map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template path is required")
}

func TestDOCXFormatter_MissingFile(t *testing.T) {
	f := NewDOCXFormatter()
	_, err := f.Format(createTestPRs(), "", map[string]string{
		"template": "/nonexistent/path/template.docx",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read DOCX template")
}

func TestDOCXFormatter_InvalidZip(t *testing.T) {
	f := NewDOCXFormatter()
	path := writeTempDocx(t, []byte("this is not a zip file"))
	_, err := f.Format(createTestPRs(), "", map[string]string{"template": path})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open DOCX as ZIP")
}

func TestDOCXFormatter_ValidTemplate(t *testing.T) {
	docXML := simpleDocumentXML(
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc>` +
			`<w:tc><w:p><w:r><w:t>{{author}}</w:t></w:r></w:p></w:tc></w:tr>`,
	)
	path := writeTempDocx(t, buildMinimalDocx(docXML))

	f := NewDOCXFormatter()
	output, err := f.Format(createTestPRs(), "02.01.2006", map[string]string{
		"template": path,
		"org":      "myorg",
		"project":  "myproject",
	})

	require.NoError(t, err)
	require.NotEmpty(t, output)

	assert.Equal(t, "PK", string(output[:2]))

	zr, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
	require.NoError(t, err)

	var docContent string
	for _, zf := range zr.File {
		if zf.Name == "word/document.xml" {
			rc, _ := zf.Open()
			data, _ := io.ReadAll(rc)
			_ = rc.Close()
			docContent = string(data)
			break
		}
	}

	require.NotEmpty(t, docContent, "word/document.xml not found in output")
	assert.Contains(t, docContent, "Add new feature")
	assert.Contains(t, docContent, "Fix bug in login")
	assert.NotContains(t, docContent, "{{title}}")
	assert.NotContains(t, docContent, "{{author}}")
}

func TestDOCXFormatter_EmptyPRList(t *testing.T) {
	docXML := simpleDocumentXML(
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`,
	)
	path := writeTempDocx(t, buildMinimalDocx(docXML))

	f := NewDOCXFormatter()
	output, err := f.Format([]models.PullRequest{}, "", map[string]string{"template": path})

	require.NoError(t, err)
	assert.Equal(t, "PK", string(output[:2]))
}

func TestDOCXFormatter_SplitRunTemplate(t *testing.T) {
	docXML := simpleDocumentXML(
		`<w:tr><w:tc><w:p>` +
			`<w:r><w:t>{{ti</w:t></w:r>` +
			`<w:r><w:t>tle}}</w:t></w:r>` +
			`</w:p></w:tc></w:tr>`,
	)
	path := writeTempDocx(t, buildMinimalDocx(docXML))

	f := NewDOCXFormatter()
	prs := []models.PullRequest{{ID: 1, Title: "Split Run PR"}}
	output, err := f.Format(prs, "", map[string]string{"template": path})

	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
	require.NoError(t, err)

	for _, zf := range zr.File {
		if zf.Name == "word/document.xml" {
			rc, _ := zf.Open()
			data, _ := io.ReadAll(rc)
			_ = rc.Close()
			assert.Contains(t, string(data), "Split Run PR")
			assert.NotContains(t, string(data), "{{title}}")
		}
	}
}

func TestDOCXFormatter_XMLSpecialCharsInPRData(t *testing.T) {
	docXML := simpleDocumentXML(
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`,
	)
	path := writeTempDocx(t, buildMinimalDocx(docXML))

	f := NewDOCXFormatter()
	prs := []models.PullRequest{{ID: 1, Title: `Fix <XSS> & "injection"`}}
	output, err := f.Format(prs, "", map[string]string{"template": path})

	require.NoError(t, err)

	zr, _ := zip.NewReader(bytes.NewReader(output), int64(len(output)))
	for _, zf := range zr.File {
		if zf.Name == "word/document.xml" {
			rc, _ := zf.Open()
			data, _ := io.ReadAll(rc)
			_ = rc.Close()
			content := string(data)
			assert.Contains(t, content, `Fix &lt;XSS&gt; &amp; &quot;injection&quot;`)
			assert.NotContains(t, content, `<XSS>`)
		}
	}
}

func TestDOCXFormatter_MissingDocumentXML(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("[Content_Types].xml")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Types/>`))
	_ = zw.Close()

	path := writeTempDocx(t, buf.Bytes())

	f := NewDOCXFormatter()
	_, err := f.Format(createTestPRs(), "", map[string]string{"template": path})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "word/document.xml not found")
}

func TestDOCXFormatter_NonDocumentFilesPreserved(t *testing.T) {
	docXML := simpleDocumentXML(
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`,
	)
	path := writeTempDocx(t, buildMinimalDocx(docXML))

	f := NewDOCXFormatter()
	output, err := f.Format(createTestPRs(), "", map[string]string{"template": path})
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, zf := range zr.File {
		names[zf.Name] = true
	}

	assert.True(t, names["[Content_Types].xml"])
	assert.True(t, names["_rels/.rels"])
	assert.True(t, names["word/document.xml"])
}

func TestMaxExistingRID_Empty(t *testing.T) {
	assert.Equal(t, 0, maxExistingRID(`<Relationships/>`))
}

func TestMaxExistingRID_MultipleIDs(t *testing.T) {
	relsXML := `<Relationships>` +
		`<Relationship Id="rId1" Type="styles" Target="styles.xml"/>` +
		`<Relationship Id="rId3" Type="settings" Target="settings.xml"/>` +
		`</Relationships>`
	assert.Equal(t, 3, maxExistingRID(relsXML))
}

func TestAddHyperlinksToRels_OpenForm(t *testing.T) {
	relsXML := `<?xml version="1.0"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="styles" Target="styles.xml"/>` +
		`</Relationships>`
	links := []hyperlinkEntry{{rID: "rId2", url: "https://example.com/pr/42"}}
	result := addHyperlinksToRels(relsXML, links)

	assert.Contains(t, result, `Id="rId2"`)
	assert.Contains(t, result, `Target="https://example.com/pr/42"`)
	assert.Contains(t, result, `TargetMode="External"`)
	assert.Contains(t, result, `</Relationships>`)
}

func TestAddHyperlinksToRels_SelfClosingForm(t *testing.T) {
	relsXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`
	links := []hyperlinkEntry{{rID: "rId1", url: "https://example.com/pr/1"}}
	result := addHyperlinksToRels(relsXML, links)

	assert.Contains(t, result, `Id="rId1"`)
	assert.Contains(t, result, `Target="https://example.com/pr/1"`)
	assert.Contains(t, result, `</Relationships>`)
	assert.NotContains(t, result, `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`)
}

func TestAddHyperlinksToRels_NoLinks(t *testing.T) {
	relsXML := `<Relationships/>`
	result := addHyperlinksToRels(relsXML, nil)
	assert.Equal(t, relsXML, result)
}

func TestApplyHyperlinkPlaceholders_URLColumn(t *testing.T) {
	pr := models.PullRequest{
		ID:         42,
		Repository: models.Repository{Name: "my-repo"},
	}
	row := `<w:tr><w:tc><w:p>` +
		`<w:r><w:t xml:space="preserve">{{url}}</w:t></w:r>` +
		`</w:p></w:tc></w:tr>`

	result, entries := applyHyperlinkPlaceholders(row, pr, 1, "org", "project", 10)

	require.Len(t, entries, 1)
	assert.Equal(t, "rId10", entries[0].rID)
	assert.Contains(t, entries[0].url, "/pullrequest/42")
	assert.Contains(t, result, `<w:hyperlink`)
	assert.Contains(t, result, `r:id="rId10"`)
	assert.NotContains(t, result, "{{url}}")
}

func TestApplyHyperlinkPlaceholders_LinkColumn(t *testing.T) {
	pr := models.PullRequest{
		ID:         7,
		Repository: models.Repository{Name: "repo"},
	}
	row := `<w:tr><w:tc><w:p>` +
		`<w:r><w:t xml:space="preserve">{{link}}</w:t></w:r>` +
		`</w:p></w:tc></w:tr>`

	result, entries := applyHyperlinkPlaceholders(row, pr, 1, "org", "proj", 5)

	require.Len(t, entries, 1)
	assert.Equal(t, "rId5", entries[0].rID)
	assert.Contains(t, result, "LINK TO PR")
	assert.NotContains(t, result, "{{link}}")
}

func TestApplyHyperlinkPlaceholders_MixedText_FallsThrough(t *testing.T) {
	pr := models.PullRequest{ID: 1, Repository: models.Repository{Name: "r"}}
	row := `<w:r><w:t xml:space="preserve">See {{url}} here</w:t></w:r>`

	result, entries := applyHyperlinkPlaceholders(row, pr, 1, "org", "proj", 1)

	assert.Empty(t, entries)
	assert.Contains(t, result, "{{url}}")
}

func TestDOCXFormatter_HyperlinkInOutput(t *testing.T) {
	docXML := simpleDocumentXML(
		`<w:tr>` +
			`<w:tc><w:p><w:r><w:t xml:space="preserve">{{title}}</w:t></w:r></w:p></w:tc>` +
			`<w:tc><w:p><w:r><w:t xml:space="preserve">{{url}}</w:t></w:r></w:p></w:tc>` +
			`</w:tr>`,
	)
	path := writeTempDocx(t, buildMinimalDocx(docXML))

	f := NewDOCXFormatter()
	prs := []models.PullRequest{
		{ID: 42, Title: "My Feature", Repository: models.Repository{Name: "repo"}},
		{ID: 99, Title: "Another PR", Repository: models.Repository{Name: "repo"}},
	}
	output, err := f.Format(prs, "", map[string]string{
		"template": path,
		"org":      "myorg",
		"project":  "myproject",
	})
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
	require.NoError(t, err)

	var docContent, relsContent string
	for _, zf := range zr.File {
		rc, _ := zf.Open()
		data, _ := io.ReadAll(rc)
		_ = rc.Close()
		switch zf.Name {
		case "word/document.xml":
			docContent = string(data)
		case "word/_rels/document.xml.rels":
			relsContent = string(data)
		}
	}

	assert.Contains(t, docContent, `<w:hyperlink`)
	assert.Contains(t, docContent, "My Feature")
	assert.Contains(t, docContent, "Another PR")
	assert.NotContains(t, docContent, "{{url}}")

	assert.Equal(t, 2, strings.Count(relsContent, `TargetMode="External"`))
	assert.Contains(t, relsContent, "/pullrequest/42")
	assert.Contains(t, relsContent, "/pullrequest/99")
}

func TestHasPRPlaceholder_KnownColumn(t *testing.T) {
	row := `<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`
	assert.True(t, hasPRPlaceholder(row))
}

func TestHasPRPlaceholder_DocumentLevelOnly(t *testing.T) {
	row := `<w:tr><w:tc><w:p><w:r><w:t>{{dateFrom}}</w:t></w:r></w:p></w:tc></w:tr>`
	assert.False(t, hasPRPlaceholder(row))
}

func TestHasPRPlaceholder_NoPlaceholders(t *testing.T) {
	row := `<w:tr><w:tc><w:p><w:r><w:t>plain text</w:t></w:r></w:p></w:tc></w:tr>`
	assert.False(t, hasPRPlaceholder(row))
}

func TestEnsurePreserveSpace_AddsMissing(t *testing.T) {
	input := `<w:r><w:t>hello </w:t></w:r>`
	result := ensurePreserveSpace(input)
	assert.Contains(t, result, `<w:t xml:space="preserve">hello </w:t>`)
}

func TestEnsurePreserveSpace_DoesNotDuplicate(t *testing.T) {
	input := `<w:r><w:t xml:space="preserve">hello </w:t></w:r>`
	result := ensurePreserveSpace(input)
	assert.Equal(t, input, result)
	assert.Equal(t, 1, strings.Count(result, `xml:space="preserve"`))
}

func TestExpandTemplateRows_SkipsDocVarRows(t *testing.T) {
	xmlStr := `<w:body>` +
		`<w:tbl>` +
		`<w:tr><w:tc><w:p><w:r><w:t>From: {{dateFrom}}</w:t></w:r></w:p></w:tc></w:tr>` +
		`</w:tbl>` +
		`<w:tbl>` +
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>` +
		`</w:tbl>` +
		`</w:body>`

	prs := []models.PullRequest{{ID: 1, Title: "Multi-table PR"}}
	result, _, err := expandTemplateRows(xmlStr, prs, "", "", "", 1)

	require.NoError(t, err)
	assert.Contains(t, result, "Multi-table PR")
	assert.Contains(t, result, "{{dateFrom}}")
	assert.NotContains(t, result, "{{title}}")
}

func TestDOCXFormatter_MultiTable(t *testing.T) {
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body>` +
		`<w:tbl>` +
		`<w:tr><w:tc><w:p><w:r><w:t>{{dateFrom}} - {{dateTo}}</w:t></w:r></w:p></w:tc></w:tr>` +
		`</w:tbl>` +
		`<w:tbl>` +
		`<w:tr><w:tc><w:p><w:r><w:t>Title</w:t></w:r></w:p></w:tc></w:tr>` +
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>` +
		`</w:tbl>` +
		`</w:body></w:document>`

	path := writeTempDocx(t, buildMinimalDocx(docXML))
	f := NewDOCXFormatter()
	prs := []models.PullRequest{{ID: 1, Title: "Feature A"}}

	output, err := f.Format(prs, "02.01.2006", map[string]string{
		"template": path,
		"dateFrom": "01.02.2026",
		"dateTo":   "28.02.2026",
	})
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
	require.NoError(t, err)

	for _, zf := range zr.File {
		if zf.Name == "word/document.xml" {
			rc, _ := zf.Open()
			data, _ := io.ReadAll(rc)
			_ = rc.Close()
			content := string(data)

			assert.Contains(t, content, "01.02.2026", "dateFrom must be substituted")
			assert.Contains(t, content, "28.02.2026", "dateTo must be substituted")
			assert.Contains(t, content, "Feature A", "PR title must be substituted")
			assert.NotContains(t, content, "{{dateFrom}}")
			assert.NotContains(t, content, "{{dateTo}}")
			assert.NotContains(t, content, "{{title}}")
		}
	}
}
