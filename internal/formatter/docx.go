package formatter

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
)

type hyperlinkEntry struct {
	rID string
	url string
}

var rIDNumRe = regexp.MustCompile(`Id="rId(\d+)"`)

var hyperlinkRunRe = regexp.MustCompile(
	`<w:r(?:\s[^>]*)?>` +
		`(?:<w:rPr>[\s\S]*?</w:rPr>)?` +
		`<w:t(?:\s[^>]*)?>` +
		`\{\{(url|link)\}\}` +
		`</w:t></w:r>`,
)

type DOCXFormatter struct{}

func NewDOCXFormatter() *DOCXFormatter {
	return &DOCXFormatter{}
}

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

	docVars := make(map[string]string)
	for _, key := range []string{"dateFrom", "dateTo", "reportCreationDate"} {
		if v := options[key]; v != "" {
			docVars[key] = v
		}
	}

	maxRID := 0
	for _, zf := range zipReader.File {
		if zf.Name == "word/_rels/document.xml.rels" {
			rc, err := zf.Open()
			if err == nil {
				data, readErr := io.ReadAll(rc)
				_ = rc.Close()
				if readErr == nil {
					maxRID = maxExistingRID(string(data))
				}
			}
			break
		}
	}

	var newDocXML []byte
	var hyperlinks []hyperlinkEntry
	foundDocument := false
	for _, zf := range zipReader.File {
		if zf.Name == "word/document.xml" {
			foundDocument = true
			newDocXML, hyperlinks, err = processDocumentXML(zf, prs, dateFormat, org, project, docVars, maxRID+1)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if !foundDocument {
		return nil, fmt.Errorf("word/document.xml not found in template — is this a valid .docx file?")
	}

	var outBuf bytes.Buffer
	zipWriter := zip.NewWriter(&outBuf)

	for _, zf := range zipReader.File {
		switch zf.Name {
		case "word/document.xml":
			w, err := zipWriter.Create(zf.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create zip entry: %w", err)
			}
			if _, err := w.Write(newDocXML); err != nil {
				return nil, fmt.Errorf("failed to write document.xml: %w", err)
			}
		case "word/_rels/document.xml.rels":
			rc, err := zf.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open %s: %w", zf.Name, err)
			}
			data, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", zf.Name, err)
			}
			newRels := addHyperlinksToRels(string(data), hyperlinks)
			w, err := zipWriter.Create(zf.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create zip entry: %w", err)
			}
			if _, err := w.Write([]byte(newRels)); err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", zf.Name, err)
			}
		default:
			if err := copyZipEntry(zipWriter, zf); err != nil {
				return nil, err
			}
		}
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
	defer func() { _ = rc.Close() }()

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

func processDocumentXML(zf *zip.File, prs []models.PullRequest, dateFormat, org, project string, docVars map[string]string, rIdBase int) ([]byte, []hyperlinkEntry, error) {
	rc, err := zf.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open document.xml: %w", err)
	}
	defer func() { _ = rc.Close() }()

	rawXML, err := io.ReadAll(rc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read document.xml: %w", err)
	}

	result, hyperlinks, err := expandTemplateRows(string(rawXML), prs, dateFormat, org, project, rIdBase)
	if err != nil {
		return nil, nil, err
	}

	result = replaceDocumentPlaceholders(result, docVars)

	return []byte(result), hyperlinks, nil
}

type rowSpan struct {
	start   int
	end     int
	content string
}

func expandTemplateRows(xmlStr string, prs []models.PullRequest, dateFormat, org, project string, rIdBase int) (string, []hyperlinkEntry, error) {
	rows := extractTableRows(xmlStr)

	templateIdx := -1
	for i, row := range rows {
		normalized := normalizeRuns(row.content)
		if hasPRPlaceholder(normalized) {
			templateIdx = i
			rows[i].content = normalized
			break
		}
	}

	if templateIdx == -1 {
		return "", nil, fmt.Errorf("no template row found in DOCX: no <w:tr> contains a PR placeholder (e.g. {{title}}, {{author}})")
	}

	templateRow := ensurePreserveSpace(rows[templateIdx].content)

	var allHyperlinks []hyperlinkEntry
	var expandedRows strings.Builder
	for i, pr := range prs {
		rowWithLinks, prLinks := applyHyperlinkPlaceholders(templateRow, pr, i+1, org, project, rIdBase+len(allHyperlinks))
		allHyperlinks = append(allHyperlinks, prLinks...)

		expanded := applyPlaceholders(rowWithLinks, pr, i+1, dateFormat, org, project)
		expandedRows.WriteString(expanded)
	}

	result := xmlStr[:rows[templateIdx].start] +
		expandedRows.String() +
		xmlStr[rows[templateIdx].end:]

	return result, allHyperlinks, nil
}

func hasPRPlaceholder(trXML string) bool {
	for _, m := range placeholderRe.FindAllStringSubmatch(trXML, -1) {
		if _, ok := AvailableColumns[m[1]]; ok {
			return true
		}
	}
	return false
}

func ensurePreserveSpace(xmlStr string) string {
	return strings.ReplaceAll(xmlStr, "<w:t>", `<w:t xml:space="preserve">`)
}

func applyHyperlinkPlaceholders(rowXML string, pr models.PullRequest, index int, org, project string, rIdBase int) (string, []hyperlinkEntry) {
	var entries []hyperlinkEntry

	result := replaceParagraphs(rowXML, func(pXML string) string {
		return hyperlinkRunRe.ReplaceAllStringFunc(pXML, func(match string) string {
			sub := hyperlinkRunRe.FindStringSubmatch(match)
			if len(sub) < 2 {
				return match
			}
			fieldName := sub[1]

			prURL := AvailableColumns["url"].GetValue(pr, index, org, project)

			var displayText string
			if fieldName == "link" {
				displayText = AvailableColumns["link"].GetValue(pr, index, org, project)
			} else {
				displayText = prURL
			}

			rID := fmt.Sprintf("rId%d", rIdBase+len(entries))
			entries = append(entries, hyperlinkEntry{rID: rID, url: prURL})

			return fmt.Sprintf(
				`<w:hyperlink r:id="%s" w:history="1"`+
					` xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`+
					`<w:r>%s`+
					`<w:t xml:space="preserve">%s</w:t></w:r></w:hyperlink>`,
				rID,
				buildHyperlinkRunProps(match),
				xmlEscape(displayText),
			)
		})
	})

	return result, entries
}

func buildHyperlinkRunProps(runXML string) string {
	original := runPropsRe.FindString(runXML)
	if original == "" {
		return `<w:rPr><w:rStyle w:val="Hyperlink"/></w:rPr>`
	}
	inner := original[len("<w:rPr>") : len(original)-len("</w:rPr>")]
	return `<w:rPr><w:rStyle w:val="Hyperlink"/>` + inner + `</w:rPr>`
}

func maxExistingRID(relsXML string) int {
	max := 0
	for _, m := range rIDNumRe.FindAllStringSubmatch(relsXML, -1) {
		n, _ := strconv.Atoi(m[1])
		if n > max {
			max = n
		}
	}
	return max
}

func addHyperlinksToRels(relsXML string, hyperlinks []hyperlinkEntry) string {
	if len(hyperlinks) == 0 {
		return relsXML
	}

	const hlType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink"

	var entries strings.Builder
	for _, h := range hyperlinks {
		fmt.Fprintf(&entries,
			`<Relationship Id="%s" Type="%s" Target="%s" TargetMode="External"/>`,
			h.rID, hlType, xmlEscape(h.url),
		)
	}

	if strings.Contains(relsXML, "</Relationships>") {
		return strings.Replace(relsXML, "</Relationships>", entries.String()+"</Relationships>", 1)
	}

	relStart := strings.Index(relsXML, "<Relationships")
	if relStart == -1 {
		return relsXML
	}
	closeIdx := strings.Index(relsXML[relStart:], "/>")
	if closeIdx == -1 {
		return relsXML
	}
	abs := relStart + closeIdx
	return relsXML[:abs] + ">" + entries.String() + "</Relationships>" + relsXML[abs+2:]
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

func normalizeRuns(trXML string) string {
	return replaceParagraphs(trXML, func(pXML string) string {
		allText := extractAllText(pXML)

		if !strings.Contains(allText, "{{") && !strings.Contains(allText, "}}") {
			return pXML
		}

		if !hasAnySplitPlaceholder(pXML, allText) {
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

func hasAnySplitPlaceholder(pXML, allText string) bool {
	for _, m := range placeholderRe.FindAllString(allText, -1) {
		if isSplitAcrossRuns(pXML, m) {
			return true
		}
	}
	return false
}

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

func replaceRuns(pXML string, newRunXML string) string {
	locs := allRunsRe.FindAllStringIndex(pXML, -1)
	if len(locs) == 0 {
		return pXML
	}

	firstStart := locs[0][0]
	lastEnd := locs[len(locs)-1][1]

	return pXML[:firstStart] + newRunXML + pXML[lastEnd:]
}

func isSplitAcrossRuns(pXML, placeholder string) bool {
	matches := wtTextRe.FindAllStringSubmatch(pXML, -1)
	for _, m := range matches {
		if strings.Contains(xmlUnescape(m[1]), placeholder) {
			return false
		}
	}
	return true
}

func replaceDocumentPlaceholders(xmlStr string, docVars map[string]string) string {
	if len(docVars) == 0 {
		return xmlStr
	}

	xmlStr = replaceParagraphs(xmlStr, func(pXML string) string {
		allText := extractAllText(pXML)
		needsNormalization := false
		for key := range docVars {
			placeholder := "{{" + key + "}}"
			if strings.Contains(allText, placeholder) && isSplitAcrossRuns(pXML, placeholder) {
				needsNormalization = true
				break
			}
		}
		if !needsNormalization {
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

	for key, value := range docVars {
		xmlStr = strings.ReplaceAll(xmlStr, "{{"+key+"}}", xmlEscape(value))
	}

	return xmlStr
}

var placeholderRe = regexp.MustCompile(`\{\{(\w+)\}\}`)

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
