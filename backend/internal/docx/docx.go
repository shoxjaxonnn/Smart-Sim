package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// ExtractText reads DOCX bytes and returns simple paragraph-separated text.
func ExtractText(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("not a docx zip: %w", err)
	}

	var doc io.ReadCloser
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			doc, err = f.Open()
			if err != nil {
				return "", err
			}
			defer doc.Close()
			break
		}
	}
	if doc == nil {
		return "", fmt.Errorf("word/document.xml not found")
	}

	dec := xml.NewDecoder(doc)
	var (
		out    []string
		para   strings.Builder
		inText bool
	)

	flush := func() {
		text := strings.TrimSpace(para.String())
		if text != "" {
			out = append(out, text)
		}
		para.Reset()
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				flush()
				inText = true
			case "tab":
				if inText {
					para.WriteString("\t")
				}
			case "br":
				if inText {
					para.WriteString("\n")
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				flush()
				inText = false
			}
		case xml.CharData:
			if inText {
				para.WriteString(string(t))
			}
		}
	}
	flush()
	return strings.Join(out, "\n\n"), nil
}

// ExtractFile reads DOCX from disk and returns simple text.
func ExtractFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return ExtractText(b)
}
