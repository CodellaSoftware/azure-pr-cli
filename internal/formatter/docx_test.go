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

// buildMinimalDocx creates a minimal valid .docx ZIP in memory using the
// provided content for word/document.xml.
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
		w.Write([]byte(content))
	}
	zw.Close()
	return buf.Bytes()
}

// writeTempDocx writes DOCX bytes to a temp file and returns its path.
func writeTempDocx(t *testing.T, content []byte) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "template-*.docx")
	require.NoError(t, err)
	_, err = f.Write(content)
	require.NoError(t, err)
	f.Close()
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

// --- xmlEscape / xmlUnescape ---

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

// --- extractTableRows ---

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
	// <w:trPr> should not be mistaken for <w:tr>
	xml := `<w:tbl><w:tr><w:trPr><w:cantSplit/></w:trPr><w:tc><w:t>data</w:t></w:tc></w:tr></w:tbl>`
	rows := extractTableRows(xml)
	require.Len(t, rows, 1)
	assert.Contains(t, rows[0].content, "data")
}

func TestExtractTableRows_Empty(t *testing.T) {
	rows := extractTableRows(`<w:tbl></w:tbl>`)
	assert.Len(t, rows, 0)
}

// --- normalizeRuns ---

func TestNormalizeRuns_NoPlaceholder(t *testing.T) {
	input := `<w:tr><w:tc><w:p><w:r><w:t>hello</w:t></w:r><w:r><w:t> world</w:t></w:r></w:p></w:tc></w:tr>`
	result := normalizeRuns(input)
	// No placeholder → paragraph unchanged
	assert.Equal(t, input, result)
}

func TestNormalizeRuns_SplitPlaceholder(t *testing.T) {
	input := `<w:tr><w:tc><w:p>` +
		`<w:r><w:t>{{ti</w:t></w:r>` +
		`<w:r><w:t>tle}}</w:t></w:r>` +
		`</w:p></w:tc></w:tr>`
	result := normalizeRuns(input)

	assert.Contains(t, result, "{{title}}")
	// Should be a single <w:r> now
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
	// When {{placeholder}} is fully within a single <w:t>, no merge should happen.
	input := `<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`
	result := normalizeRuns(input)
	// Paragraph must be returned unchanged — no extra wrapping.
	assert.Equal(t, input, result)
}

func TestNormalizeRuns_PreservesFormattingWhenNoSplit(t *testing.T) {
	// Paragraph has two runs with different formatting; {{title}} is complete in
	// the second run. Because no placeholder is split, both runs must be preserved.
	input := `<w:tr><w:tc><w:p>` +
		`<w:r><w:rPr><w:b/></w:rPr><w:t>Label: </w:t></w:r>` +
		`<w:r><w:t>{{title}}</w:t></w:r>` +
		`</w:p></w:tc></w:tr>`
	result := normalizeRuns(input)
	assert.Equal(t, input, result)
	assert.Equal(t, 2, strings.Count(result, "<w:r>"), "both runs must be preserved")
}

func TestNormalizeRuns_MultipleFieldsInRow(t *testing.T) {
	// Each cell (paragraph) has its own split run
	input := `<w:tr>` +
		`<w:tc><w:p><w:r><w:t>{{ti</w:t></w:r><w:r><w:t>tle}}</w:t></w:r></w:p></w:tc>` +
		`<w:tc><w:p><w:r><w:t>{{au</w:t></w:r><w:r><w:t>thor}}</w:t></w:r></w:p></w:tc>` +
		`</w:tr>`
	result := normalizeRuns(input)
	assert.Contains(t, result, "{{title}}")
	assert.Contains(t, result, "{{author}}")
}

// --- applyPlaceholders ---

func testPR() models.PullRequest {
	return models.PullRequest{
		ID:    42,
		Title: "My PR title",
		Status: "completed",
		CreatedBy: models.User{DisplayName: "Jane Doe"},
		Repository: models.Repository{Name: "my-repo"},
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
	// Unknown placeholder left intact
	assert.Contains(t, result, "{{unknown_xyz}}")
}

// --- expandTemplateRows ---

func TestExpandTemplateRows_Basic(t *testing.T) {
	xmlStr := simpleDocumentXML(
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`,
	)
	prs := createTestPRs()

	result, err := expandTemplateRows(xmlStr, prs, "", "org", "project")
	require.NoError(t, err)

	assert.Contains(t, result, "Add new feature")
	assert.Contains(t, result, "Fix bug in login")
	// Template placeholder itself should be gone
	assert.NotContains(t, result, "{{title}}")
	// Header row should still be there
	assert.Contains(t, result, "Header")
}

func TestExpandTemplateRows_NoTemplateRow(t *testing.T) {
	xmlStr := simpleDocumentXML(
		`<w:tr><w:tc><w:p><w:r><w:t>plain row</w:t></w:r></w:p></w:tc></w:tr>`,
	)
	_, err := expandTemplateRows(xmlStr, createTestPRs(), "", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no template row found")
}

func TestExpandTemplateRows_EmptyPRList(t *testing.T) {
	xmlStr := simpleDocumentXML(
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`,
	)
	result, err := expandTemplateRows(xmlStr, []models.PullRequest{}, "", "", "")
	require.NoError(t, err)
	// Template row replaced with nothing, header row still present
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
	result, err := expandTemplateRows(xmlStr, prs, "", "", "")
	require.NoError(t, err)

	// 1 header row (from simpleDocumentXML) + 3 expanded data rows
	assert.Equal(t, 4, strings.Count(result, "<w:tr>"))
	assert.Contains(t, result, "PR One")
	assert.Contains(t, result, "PR Two")
	assert.Contains(t, result, "PR Three")
}

// --- replaceDocumentPlaceholders ---

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
	// Word splits {{dateFrom}} across two runs
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
	// {{unknown}} is not in docVars, should be untouched
	assert.Contains(t, result, "{{unknown}}")
}

func TestReplaceDocumentPlaceholders_EmptyVars(t *testing.T) {
	xmlStr := `<w:p><w:r><w:t>{{dateFrom}}</w:t></w:r></w:p>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{})
	// Empty vars map → no replacement
	assert.Equal(t, xmlStr, result)
}

func TestReplaceDocumentPlaceholders_PreservesMixedFormatting(t *testing.T) {
	// A paragraph where the first run is bold and the second run (with placeholder)
	// is not. Only the placeholder run should be replaced; the bold run untouched.
	xmlStr := `<w:p>` +
		`<w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">Label: </w:t></w:r>` +
		`<w:r><w:t>{{dateFrom}}</w:t></w:r>` +
		`</w:p>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{
		"dateFrom": "01.02.2026",
	})
	assert.Contains(t, result, "01.02.2026")
	assert.NotContains(t, result, "{{dateFrom}}")
	// The bold run must still exist unchanged — formatting was NOT merged
	assert.Contains(t, result, `<w:rPr><w:b/></w:rPr>`)
	// Two separate <w:r> elements must still be present (no merging happened)
	assert.Equal(t, 2, strings.Count(result, "<w:r>"))
}

func TestReplaceDocumentPlaceholders_InternalOptionsNotLeaked(t *testing.T) {
	// Passing only date keys means internal keys like "org" are never replaced
	// even if {{org}} appears in the document.
	xmlStr := `<w:p><w:r><w:t>{{org}}</w:t></w:r></w:p>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{
		"dateFrom": "01.02.2026",
	})
	// {{org}} is not a date key → must remain untouched
	assert.Contains(t, result, "{{org}}")
}

func TestReplaceDocumentPlaceholders_XMLEscaping(t *testing.T) {
	xmlStr := `<w:p><w:r><w:t>{{dateFrom}}</w:t></w:r></w:p>`
	result := replaceDocumentPlaceholders(xmlStr, map[string]string{
		"dateFrom": "01 & 02 <March>",
	})
	assert.Contains(t, result, "01 &amp; 02 &lt;March&gt;")
}

// --- DOCXFormatter integration with document-level placeholders ---

func TestDOCXFormatter_DocumentVarsOutsideTable(t *testing.T) {
	// Placeholders appear in a paragraph outside the table
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
		"template":            path,
		"dateFrom":            "01.02.2026",
		"dateTo":              "28.02.2026",
		"reportCreationDate":  "28.02.2026",
	})
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
	require.NoError(t, err)

	for _, zf := range zr.File {
		if zf.Name == "word/document.xml" {
			rc, _ := zf.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
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

// --- DOCXFormatter integration ---

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

	// Output must be a valid ZIP (DOCX)
	assert.Equal(t, "PK", string(output[:2]))

	// Open output ZIP and verify document.xml content
	zr, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
	require.NoError(t, err)

	var docContent string
	for _, zf := range zr.File {
		if zf.Name == "word/document.xml" {
			rc, _ := zf.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
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
	// Simulate Word splitting {{title}} across multiple runs
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
			rc.Close()
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
			rc.Close()
			content := string(data)
			assert.Contains(t, content, `Fix &lt;XSS&gt; &amp; &quot;injection&quot;`)
			// Raw unescaped special chars must not appear in XML
			assert.NotContains(t, content, `<XSS>`)
		}
	}
}

func TestDOCXFormatter_MissingDocumentXML(t *testing.T) {
	// Build a valid ZIP that contains no word/document.xml
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("[Content_Types].xml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Types/>`))
	zw.Close()

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

// --- hasPRPlaceholder ---

func TestHasPRPlaceholder_KnownColumn(t *testing.T) {
	row := `<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>`
	assert.True(t, hasPRPlaceholder(row))
}

func TestHasPRPlaceholder_DocumentLevelOnly(t *testing.T) {
	// {{dateFrom}} is not a PR column — must not qualify as template row
	row := `<w:tr><w:tc><w:p><w:r><w:t>{{dateFrom}}</w:t></w:r></w:p></w:tc></w:tr>`
	assert.False(t, hasPRPlaceholder(row))
}

func TestHasPRPlaceholder_NoPlaceholders(t *testing.T) {
	row := `<w:tr><w:tc><w:p><w:r><w:t>plain text</w:t></w:r></w:p></w:tc></w:tr>`
	assert.False(t, hasPRPlaceholder(row))
}

// --- ensurePreserveSpace ---

func TestEnsurePreserveSpace_AddsMissing(t *testing.T) {
	input := `<w:r><w:t>hello </w:t></w:r>`
	result := ensurePreserveSpace(input)
	assert.Contains(t, result, `<w:t xml:space="preserve">hello </w:t>`)
}

func TestEnsurePreserveSpace_DoesNotDuplicate(t *testing.T) {
	input := `<w:r><w:t xml:space="preserve">hello </w:t></w:r>`
	result := ensurePreserveSpace(input)
	// Must not create double attribute
	assert.Equal(t, input, result)
	assert.Equal(t, 1, strings.Count(result, `xml:space="preserve"`))
}

// --- multi-table: document-level vars in first table, PR data in second ---

func TestExpandTemplateRows_SkipsDocVarRows(t *testing.T) {
	// First table has only document-level vars (not PR columns) → must be skipped.
	// Second table has PR columns → must be expanded.
	xmlStr := `<w:body>` +
		`<w:tbl>` +
		`<w:tr><w:tc><w:p><w:r><w:t>From: {{dateFrom}}</w:t></w:r></w:p></w:tc></w:tr>` +
		`</w:tbl>` +
		`<w:tbl>` +
		`<w:tr><w:tc><w:p><w:r><w:t>{{title}}</w:t></w:r></w:p></w:tc></w:tr>` +
		`</w:tbl>` +
		`</w:body>`

	prs := []models.PullRequest{{ID: 1, Title: "Multi-table PR"}}
	result, err := expandTemplateRows(xmlStr, prs, "", "", "")

	require.NoError(t, err)
	// PR title must appear (second table expanded)
	assert.Contains(t, result, "Multi-table PR")
	// Document-level placeholder in first table must be left intact for
	// replaceDocumentPlaceholders to handle later
	assert.Contains(t, result, "{{dateFrom}}")
	// The template placeholder itself must be gone
	assert.NotContains(t, result, "{{title}}")
}

func TestDOCXFormatter_MultiTable(t *testing.T) {
	// Full integration: document has a header table with {{dateFrom}}/{{dateTo}}
	// and a data table with {{title}}. Both must be filled correctly.
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
			rc.Close()
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
