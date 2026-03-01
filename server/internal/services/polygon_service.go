package services

import (
	"archive/zip"
	"campuscompile/api/internal/models"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func ProcessPolygonZip(zipPath string) (*models.PolygonProblem, string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, "", err
	}
	defer r.Close()

	var problem models.PolygonProblem
	var xmlFound bool

	// PASS 1: Find and parse problem.xml
	for _, f := range r.File {
		if f.Name == "problem.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, "", err
			}
			data, _ := io.ReadAll(rc)
			rc.Close()

			if err := xml.Unmarshal(data, &problem); err != nil {
				return nil, "", err
			}
			xmlFound = true
			break
		}
	}

	if !xmlFound {
		return nil, "", errors.New("problem.xml not found in package")
	}

	// Figure out where the statement is stored
	statementPath := ""
	for _, stmt := range problem.Statements {
		// Prefer English statement if available
		if stmt.Lang == "english" {
			statementPath = stmt.File
			break
		}
	}

	// Fallback to the first available statement if no English is found
	if statementPath == "" && len(problem.Statements) > 0 {
		statementPath = problem.Statements[0].File
	}

	statementText := "Description not provided in package."

	// PASS 2: Read the actual statement file
	if statementPath != "" {
		for _, f := range r.File {
			if f.Name == statementPath {
				rc, err := f.Open()
				if err == nil {
					data, _ := io.ReadAll(rc)

					// Pass the raw data through the cleaner!
					statementText = CleanPolygonStatement(string(data))

					rc.Close()
				}
				break
			}
		}
	}

	return &problem, statementText, nil
}

// CleanPolygonStatement strips Codeforces Polygon LaTeX structural tags
// and converts them to standard Markdown headers for the frontend.
func CleanPolygonStatement(raw string) string {
	cleaned := raw

	reProblemStart := regexp.MustCompile(`\\begin\{problem\}\{.*?\}\{.*?\}\{.*?\}\{.*?\}\{.*?\}`)
	cleaned = reProblemStart.ReplaceAllString(cleaned, "")

	cleaned = strings.ReplaceAll(cleaned, "\\end{problem}", "")

	reUsePackage := regexp.MustCompile(`\\usepackage(?:\[.*?\])?\{.*?\}`)
	cleaned = reUsePackage.ReplaceAllString(cleaned, "")

	reDef := regexp.MustCompile(`\\(?:def|newcommand).*?\{.*?\}`)
	cleaned = reDef.ReplaceAllString(cleaned, "")

	cleaned = strings.ReplaceAll(cleaned, "\\InputFile", "\n**Input**\n")
	cleaned = strings.ReplaceAll(cleaned, "\\OutputFile", "\n**Output**\n")
	cleaned = strings.ReplaceAll(cleaned, "\\Example", "")
	cleaned = strings.ReplaceAll(cleaned, "\\Note", "\n**Note**\n")

	reExmp := regexp.MustCompile(`\\(?:exmp|exmpfile)\{.*?\}\{.*?\}`)
	cleaned = reExmp.ReplaceAllString(cleaned, "")

	reNewlines := regexp.MustCompile(`\n{3,}`)
	cleaned = reNewlines.ReplaceAllString(cleaned, "\n\n")

	return strings.TrimSpace(cleaned)
}

// ExtractAndSaveTestCases streams test files from the ZIP directly to disk and returns their paths.
func ExtractAndSaveTestCases(r *zip.Reader, problem *models.PolygonProblem, problemID string) ([]models.TestCaseData, error) {
	// 1. Create the persistent storage directory for this specific problem
	baseStoragePath := filepath.Join("/", "sandbox_shared", "problems", problemID, "tests")
	if err := os.MkdirAll(baseStoragePath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	inputs := make(map[string]string)
	outputs := make(map[string]string)

	// 2. Stream files directly from ZIP to Disk
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "tests/") {
			baseName := filepath.Base(f.Name)
			if baseName == "tests" {
				continue // Skip the root directory itself
			}

			// Polygon names outputs with .a, inputs have no extension
			idx := baseName
			isOutput := strings.HasSuffix(baseName, ".a")
			if isOutput {
				idx = strings.TrimSuffix(baseName, ".a")
			}

			// Format the new file name (e.g., "01.in" and "01.out")
			ext := ".in"
			if isOutput {
				ext = ".out"
			}
			destFileName := fmt.Sprintf("%s%s", idx, ext)
			destPath := filepath.Join(baseStoragePath, destFileName)

			// Open file inside the zip
			rc, err := f.Open()
			if err != nil {
				continue
			}

			// Create the destination file on the server's disk
			destFile, err := os.Create(destPath)
			if err != nil {
				rc.Close()
				continue
			}

			// Stream the data in 32KB chunks
			_, err = io.Copy(destFile, rc)

			destFile.Close()
			rc.Close()

			if err != nil {
				return nil, fmt.Errorf("failed to stream file %s to disk: %v", f.Name, err)
			}

			// Record the saved path
			if isOutput {
				outputs[idx] = destPath
			} else {
				inputs[idx] = destPath
			}
		}
	}

	var testCases []models.TestCaseData

	// 3. Map the saved files to the XML metadata
	for i, testMeta := range problem.Judging.Testset.Tests {
		idxStr := fmt.Sprintf("%02d", i+1)

		inPath, hasIn := inputs[idxStr]
		outPath, hasOut := outputs[idxStr]

		if hasIn && hasOut {
			// sample="true" means is_hidden=false
			isHidden := strings.ToLower(testMeta.Sample) != "true"

			testCases = append(testCases, models.TestCaseData{
				TestIndex:      i + 1,
				IsHidden:       isHidden,
				InputFilePath:  inPath,
				OutputFilePath: outPath,
			})
		}
	}

	return testCases, nil
}
