package formatter

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
)

// DOCXFormatter fills a .docx template by expanding a template table row
// (containing {{field}} placeholders) once per PR. It follows the same
// ([]byte, error) pattern as XLSXFormatter.
type DOCXFormatter struct{}

func NewDOCXFormatter() *DOCXFormatter {
	return &DOCXFormatter{}
}

// Format reads the .docx template at options["template"], finds the first
// table row containing {{...}} placeholders, and replaces it with one row
// per PR with placeholders substituted by actual PR data.
func (f *DOCXFormatter) Format(prs []models.PullRequest, dateFormat string, options map[string]string) ([]byte, error) {
	templatePath := options["template"]
	if templatePath == "" {
		return nil, fmt.Errorf("DOCX template path is required")
	}

	org := options["org"]
	project := options["project"]

	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read DOCX template: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(templateBytes), int64(len(templateBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to open DOCX as ZIP: %w", err)
	}

	var outBuf bytes.Buffer
	zipWriter := zip.NewWriter(&outBuf)

	foundDocument := false
	for _, zf := range zipReader.File {
		if zf.Name == "word/document.xml" {
			foundDocument = true
			newXML, err := processDocumentXML(zf, prs, dateFormat, org, project)
			if err != nil {
				return nil, err
			}
			w, err := zipWriter.Create(zf.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create zip entry: %w", err)
			}
			if _, err := w.Write(newXML); err != nil {
				return nil, fmt.Errorf("failed to write document.xml: %w", err)
			}
		} else {
			if err := copyZipEntry(zipWriter, zf); err != nil {
				return nil, err
			}
		}
	}

	if !foundDocument {
		return nil, fmt.Errorf("word/document.xml not found in template — is this a valid .docx file?")
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize DOCX: %w", err)
	}

	return outBuf.Bytes(), nil
}

func copyZipEntry(w *zip.Writer, src *zip.File) error {
	rc, err := src.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip entry %s: %w", src.Name, err)
	}
	defer rc.Close()

	dst, err := w.Create(src.Name)
	if err != nil {
		return fmt.Errorf("failed to create zip entry %s: %w", src.Name, err)
	}

	data, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("failed to read zip entry %s: %w", src.Name, err)
	}

	_, err = dst.Write(data)
	return err
}

func processDocumentXML(zf *zip.File, prs []models.PullRequest, dateFormat, org, project string) ([]byte, error) {
	rc, err := zf.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open document.xml: %w", err)
	}
	defer rc.Close()

	rawXML, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read document.xml: %w", err)
	}

	result, err := expandTemplateRows(string(rawXML), prs, dateFormat, org, project)
	if err != nil {
		return nil, err
	}

	return []byte(result), nil
}

// rowSpan holds the character offsets and content of a <w:tr>...</w:tr> block.
type rowSpan struct {
	start   int
	end     int
	content string
}

func expandTemplateRows(xmlStr string, prs []models.PullRequest, dateFormat, org, project string) (string, error) {
	rows := extractTableRows(xmlStr)

	templateIdx := -1
	for i, row := range rows {
		normalized := normalizeRuns(row.content)
		if strings.Contains(normalized, "{{") {
			templateIdx = i
			rows[i].content = normalized
			break
		}
	}

	if templateIdx == -1 {
		return "", fmt.Errorf("no template row found in DOCX: no <w:tr> contains '{{' placeholders")
	}

	templateRow := rows[templateIdx].content

	var expandedRows strings.Builder
	for i, pr := range prs {
		expanded := applyPlaceholders(templateRow, pr, i+1, dateFormat, org, project)
		expandedRows.WriteString(expanded)
	}

	result := xmlStr[:rows[templateIdx].start] +
		expandedRows.String() +
		xmlStr[rows[templateIdx].end:]

	return result, nil
}

func extractTableRows(xmlStr string) []rowSpan {
	var spans []rowSpan
	searchFrom := 0

	for {
		startIdx := strings.Index(xmlStr[searchFrom:], "<w:tr")
		if startIdx == -1 {
			break
		}
		absStart := searchFrom + startIdx

		// Make sure it's actually a <w:tr> tag (not <w:trPr> etc.)
		afterTag := xmlStr[absStart+5:]
		if len(afterTag) == 0 || (afterTag[0] != '>' && afterTag[0] != ' ') {
			searchFrom = absStart + 5
			continue
		}

		endTag := "</w:tr>"
		endIdx := strings.Index(xmlStr[absStart:], endTag)
		if endIdx == -1 {
			break
		}
		absEnd := absStart + endIdx + len(endTag)

		spans = append(spans, rowSpan{
			start:   absStart,
			end:     absEnd,
			content: xmlStr[absStart:absEnd],
		})

		searchFrom = absEnd
	}

	return spans
}

// normalizeRuns fixes Word's split-run problem within a <w:tr> block.
// For any <w:p> whose concatenated text contains "{{" or "}}", it merges
// all <w:r> elements into a single run, preserving the first run's <w:rPr>.
func normalizeRuns(trXML string) string {
	return replaceParagraphs(trXML, func(pXML string) string {
		allText := extractAllText(pXML)

		if !strings.Contains(allText, "{{") && !strings.Contains(allText, "}}") {
			return pXML
		}

		firstRunProps := extractFirstRunProps(pXML)

		var newRun strings.Builder
		newRun.WriteString("<w:r>")
		if firstRunProps != "" {
			newRun.WriteString(firstRunProps)
		}
		newRun.WriteString(`<w:t xml:space="preserve">`)
		newRun.WriteString(xmlEscape(allText))
		newRun.WriteString("</w:t>")
		newRun.WriteString("</w:r>")

		return replaceRuns(pXML, newRun.String())
	})
}

// replaceParagraphs finds each <w:p>...</w:p> in xml and calls fn on each,
// replacing the paragraph with fn's return value.
func replaceParagraphs(xml string, fn func(string) string) string {
	var result strings.Builder
	searchFrom := 0

	for {
		startIdx := strings.Index(xml[searchFrom:], "<w:p")
		if startIdx == -1 {
			result.WriteString(xml[searchFrom:])
			break
		}
		absStart := searchFrom + startIdx

		// Ensure it's <w:p> or <w:p > or <w:p>, not <w:pPr> etc.
		afterTag := xml[absStart+4:]
		if len(afterTag) == 0 || (afterTag[0] != '>' && afterTag[0] != ' ') {
			result.WriteString(xml[searchFrom : absStart+4])
			searchFrom = absStart + 4
			continue
		}

		endTag := "</w:p>"
		endIdx := strings.Index(xml[absStart:], endTag)
		if endIdx == -1 {
			result.WriteString(xml[searchFrom:])
			break
		}
		absEnd := absStart + endIdx + len(endTag)

		result.WriteString(xml[searchFrom:absStart])
		result.WriteString(fn(xml[absStart:absEnd]))
		searchFrom = absEnd
	}

	return result.String()
}

var wtTextRe = regexp.MustCompile(`<w:t(?:\s[^>]*)?>([^<]*)</w:t>`)

// extractAllText concatenates the text of all <w:t> elements in the XML.
func extractAllText(pXML string) string {
	matches := wtTextRe.FindAllStringSubmatch(pXML, -1)
	var sb strings.Builder
	for _, m := range matches {
		sb.WriteString(xmlUnescape(m[1]))
	}
	return sb.String()
}

var firstRunRe = regexp.MustCompile(`<w:r[ >]`)
var runPropsRe = regexp.MustCompile(`<w:rPr>[\s\S]*?</w:rPr>`)

// extractFirstRunProps returns the <w:rPr>...</w:rPr> from the first <w:r> in pXML, or "".
func extractFirstRunProps(pXML string) string {
	loc := firstRunRe.FindStringIndex(pXML)
	if loc == nil {
		return ""
	}

	endTag := "</w:r>"
	endIdx := strings.Index(pXML[loc[0]:], endTag)
	if endIdx == -1 {
		return ""
	}
	firstRunXML := pXML[loc[0] : loc[0]+endIdx+len(endTag)]

	m := runPropsRe.FindString(firstRunXML)
	return m
}

var allRunsRe = regexp.MustCompile(`<w:r[ >][\s\S]*?</w:r>`)

// replaceRuns replaces all <w:r>...</w:r> elements in pXML with newRunXML.
func replaceRuns(pXML string, newRunXML string) string {
	locs := allRunsRe.FindAllStringIndex(pXML, -1)
	if len(locs) == 0 {
		return pXML
	}

	firstStart := locs[0][0]
	lastEnd := locs[len(locs)-1][1]

	return pXML[:firstStart] + newRunXML + pXML[lastEnd:]
}

var placeholderRe = regexp.MustCompile(`\{\{(\w+)\}\}`)

// applyPlaceholders replaces {{field}} patterns in templateRowXML with PR data.
func applyPlaceholders(templateRowXML string, pr models.PullRequest, index int, dateFormat, org, project string) string {
	return placeholderRe.ReplaceAllStringFunc(templateRowXML, func(match string) string {
		fieldName := match[2 : len(match)-2]

		col, exists := AvailableColumns[fieldName]
		if !exists {
			return match
		}

		var value string
		switch col.ID {
		case "completed":
			value = pr.FormatCompletionDate(dateFormat)
		case "created":
			if pr.CreationDate.IsZero() {
				value = "N/A"
			} else if dateFormat != "" {
				value = pr.CreationDate.Format(dateFormat)
			} else {
				value = pr.CreationDate.Format("2006-01-02 15:04:05")
			}
		default:
			value = col.GetValue(pr, index, org, project)
		}

		return xmlEscape(value)
	})
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func xmlUnescape(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&apos;", "'")
	return s
}
