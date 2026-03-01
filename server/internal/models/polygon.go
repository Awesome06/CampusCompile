package models

import "encoding/xml"

// PolygonProblem maps the problem.xml structure
type PolygonProblem struct {
	XMLName    xml.Name    `xml:"problem"`
	Names      []Name      `xml:"names>name"`
	Statements []Statement `xml:"statements>statement"`
	Judging    Judging     `xml:"judging"`
}

type Name struct {
	Value string `xml:"value,attr"`
	Lang  string `xml:"language,attr"`
}

type Statement struct {
	Lang string `xml:"language,attr"`
	Type string `xml:"type,attr"`
	File string `xml:"path,attr"`
}

type Judging struct {
	InputFile  string  `xml:"input-file,attr"`
	OutputFile string  `xml:"output-file,attr"`
	Testset    Testset `xml:"testset"`
}

type Testset struct {
	TimeLimit   int64  `xml:"time-limit"`   // In milliseconds
	MemoryLimit int64  `xml:"memory-limit"` // In bytes
	Tests       []Test `xml:"tests>test"`
}

type Test struct {
	Method string `xml:"method,attr"` // "manual" or "generated"
	Cmd    string `xml:"cmd,attr"`    // Generator command if applicable
	Sample string `xml:"sample,attr"` // Flags if this is a public sample test case
}

// TestCaseData holds the paired file paths for our disk-streaming extraction
type TestCaseData struct {
	TestIndex      int
	IsHidden       bool // Maps to our database column (Sample="true" -> IsHidden=false)
	InputFilePath  string
	OutputFilePath string
}
