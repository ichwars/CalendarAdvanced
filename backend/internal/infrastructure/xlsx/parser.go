package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

type Workbook struct {
	Sheets []Sheet
}

type Sheet struct {
	Name string
	Rows []Row
}

type Row struct {
	Index int
	Cells []string
}

func ParseWorkbook(data []byte) (Workbook, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Workbook{}, fmt.Errorf("xlsx could not be opened")
	}
	files := map[string]*zip.File{}
	for _, file := range reader.File {
		files[file.Name] = file
	}
	shared, err := parseSharedStrings(files["xl/sharedStrings.xml"])
	if err != nil {
		return Workbook{}, err
	}
	sheets, err := parseWorkbookSheets(files["xl/workbook.xml"], files["xl/_rels/workbook.xml.rels"])
	if err != nil {
		return Workbook{}, err
	}
	workbook := Workbook{Sheets: make([]Sheet, 0, len(sheets))}
	for _, item := range sheets {
		file := files[item.Path]
		if file == nil {
			continue
		}
		rows, err := parseSheetRows(file, shared)
		if err != nil {
			return Workbook{}, err
		}
		workbook.Sheets = append(workbook.Sheets, Sheet{Name: item.Name, Rows: rows})
	}
	return workbook, nil
}

type workbookSheet struct {
	Name string
	Path string
}

func parseWorkbookSheets(workbookFile, relsFile *zip.File) ([]workbookSheet, error) {
	if workbookFile == nil || relsFile == nil {
		return nil, fmt.Errorf("xlsx workbook metadata is missing")
	}
	relationships := map[string]string{}
	if err := decodeZipXML(relsFile, func(decoder *xml.Decoder) error {
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			start, ok := token.(xml.StartElement)
			if !ok || start.Name.Local != "Relationship" {
				continue
			}
			var id, target string
			for _, attr := range start.Attr {
				switch attr.Name.Local {
				case "Id":
					id = attr.Value
				case "Target":
					target = attr.Value
				}
			}
			if id != "" && target != "" {
				relationships[id] = workbookTargetPath(target)
			}
		}
	}); err != nil {
		return nil, err
	}
	var sheets []workbookSheet
	if err := decodeZipXML(workbookFile, func(decoder *xml.Decoder) error {
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			start, ok := token.(xml.StartElement)
			if !ok || start.Name.Local != "sheet" {
				continue
			}
			var name, relID string
			for _, attr := range start.Attr {
				switch attr.Name.Local {
				case "name":
					name = attr.Value
				case "id":
					relID = attr.Value
				}
			}
			if target := relationships[relID]; name != "" && target != "" {
				sheets = append(sheets, workbookSheet{Name: name, Path: target})
			}
		}
	}); err != nil {
		return nil, err
	}
	return sheets, nil
}

func parseSharedStrings(file *zip.File) ([]string, error) {
	if file == nil {
		return nil, nil
	}
	var values []string
	var current strings.Builder
	inString := false
	inText := false
	if err := decodeZipXML(file, func(decoder *xml.Decoder) error {
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			switch item := token.(type) {
			case xml.StartElement:
				if item.Name.Local == "si" {
					inString = true
					current.Reset()
				}
				if inString && item.Name.Local == "t" {
					inText = true
				}
			case xml.CharData:
				if inString && inText {
					current.Write([]byte(item))
				}
			case xml.EndElement:
				if item.Name.Local == "t" {
					inText = false
				}
				if item.Name.Local == "si" {
					values = append(values, current.String())
					inString = false
				}
			}
		}
	}); err != nil {
		return nil, err
	}
	return values, nil
}

func parseSheetRows(file *zip.File, shared []string) ([]Row, error) {
	rowCells := map[int]map[int]string{}
	maxColumnByRow := map[int]int{}
	var rowIndex int
	var columnIndex int
	var cellType string
	var cellValue strings.Builder
	inCell := false
	inValue := false
	if err := decodeZipXML(file, func(decoder *xml.Decoder) error {
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			switch item := token.(type) {
			case xml.StartElement:
				switch item.Name.Local {
				case "row":
					rowIndex = attrInt(item.Attr, "r")
				case "c":
					inCell = true
					cellValue.Reset()
					cellType = attrString(item.Attr, "t")
					ref := attrString(item.Attr, "r")
					if ref != "" {
						columnIndex = columnIndexFromRef(ref)
						if parsedRow := rowIndexFromRef(ref); parsedRow > 0 {
							rowIndex = parsedRow
						}
					}
				case "v", "t":
					if inCell {
						inValue = true
					}
				}
			case xml.CharData:
				if inCell && inValue {
					cellValue.Write([]byte(item))
				}
			case xml.EndElement:
				switch item.Name.Local {
				case "v", "t":
					inValue = false
				case "c":
					if rowIndex > 0 && columnIndex >= 0 {
						if rowCells[rowIndex] == nil {
							rowCells[rowIndex] = map[int]string{}
						}
						rowCells[rowIndex][columnIndex] = normalizeCellValue(cellValue.String(), cellType, shared)
						if columnIndex > maxColumnByRow[rowIndex] {
							maxColumnByRow[rowIndex] = columnIndex
						}
					}
					inCell = false
					columnIndex = 0
					cellType = ""
				}
			}
		}
	}); err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(rowCells))
	for index := 1; index <= maxRow(rowCells); index++ {
		cellsByColumn, ok := rowCells[index]
		if !ok {
			continue
		}
		cells := make([]string, maxColumnByRow[index]+1)
		for column, value := range cellsByColumn {
			cells[column] = value
		}
		rows = append(rows, Row{Index: index, Cells: cells})
	}
	return rows, nil
}

func decodeZipXML(file *zip.File, fn func(*xml.Decoder) error) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	return fn(xml.NewDecoder(reader))
}

func workbookTargetPath(target string) string {
	target = strings.TrimPrefix(target, "/")
	if strings.HasPrefix(target, "xl/") {
		return path.Clean(target)
	}
	return path.Clean("xl/" + target)
}

func normalizeCellValue(value, cellType string, shared []string) string {
	value = strings.TrimSpace(value)
	if cellType == "s" {
		index, err := strconv.Atoi(value)
		if err == nil && index >= 0 && index < len(shared) {
			return strings.TrimSpace(shared[index])
		}
	}
	return value
}

func attrString(attrs []xml.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func attrInt(attrs []xml.Attr, name string) int {
	value, _ := strconv.Atoi(attrString(attrs, name))
	return value
}

func columnIndexFromRef(ref string) int {
	column := 0
	for _, char := range ref {
		if char < 'A' || char > 'Z' {
			break
		}
		column = column*26 + int(char-'A'+1)
	}
	return column - 1
}

func rowIndexFromRef(ref string) int {
	digits := strings.Builder{}
	for _, char := range ref {
		if char >= '0' && char <= '9' {
			digits.WriteRune(char)
		}
	}
	value, _ := strconv.Atoi(digits.String())
	return value
}

func maxRow(rows map[int]map[int]string) int {
	maximum := 0
	for row := range rows {
		if row > maximum {
			maximum = row
		}
	}
	return maximum
}
