// Package main is the Go code generator for SQL review rules.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	typeFile           = "../advisor.go"
	mysqlTemplate      = "./mysql.template"
	postgresqlTemplate = "./postgresql.template"
	oracleTemplate     = "./oracle.template"
	snowflakeTemplate  = "./snowflake.template"
	mssqlTemplate      = "./mssql.template"
	lowerMySQL         = "mysql"
	lowerPostgreSQL    = "postgresql"
	lowerOracle        = "oracle"
	lowerSnowflake     = "snowflake"
	lowerMSSQL         = "mssql"
)

var (
	flags struct {
		rule string
	}

	cmd = &cobra.Command{
		Use:   "generator",
		Short: "This is a SQL review rule generator",
		Run: func(cmd *cobra.Command, args []string) {
			// Get AdvisorComment, AdvisorName, CheckerName, FileName and TestName
			var advisorComment, advisorName, checkerName, fileName, testName string
			var fileNameTokenList, advisorNameTokenList []string
			var engineType string
			file, err := os.Open(typeFile)
			if err != nil {
				fmt.Printf("cannot open %q: %s\n", typeFile, err.Error())
				return
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if !strings.Contains(line, flags.rule+" ") {
					continue
				}
				wordList := strings.Fields(line)
				if strings.HasPrefix(wordList[0], "//") {
					for i, word := range wordList {
						switch strings.ToLower(word) {
						case lowerMySQL:
							advisorComment = strings.Join(wordList[i+1:], " ")
							engineType = lowerMySQL
						case lowerPostgreSQL:
							advisorComment = strings.Join(wordList[i+1:], " ")
							engineType = lowerPostgreSQL
						case lowerOracle:
							advisorComment = strings.Join(wordList[i+1:], " ")
							engineType = lowerOracle
						case lowerSnowflake:
							advisorComment = strings.Join(wordList[i+1:], " ")
							engineType = lowerSnowflake
						case lowerMSSQL:
							advisorComment = strings.Join(wordList[i+1:], " ")
							engineType = lowerMSSQL
						}
						if advisorComment != "" {
							break
						}
					}
				} else {
					needed := false
					typeToken := strings.Split(strings.Trim(wordList[3], "\""), ".")
					for _, token := range typeToken {
						if needed {
							nameWord := strings.Split(token, "-")
							fileNameTokenList = append(fileNameTokenList, nameWord...)
							continue
						}
						switch token {
						case lowerMySQL, lowerPostgreSQL, lowerOracle, lowerSnowflake, lowerMSSQL:
							needed = true
						}
					}
					break
				}
			}

			fileName = fmt.Sprintf("advisor_%s", strings.Join(fileNameTokenList, "_"))
			for _, token := range fileNameTokenList {
				advisorNameTokenList = append(advisorNameTokenList, cases.Title(language.AmericanEnglish).String(token))
			}
			testName = strings.Join(advisorNameTokenList, "")
			advisorName = fmt.Sprintf("%sAdvisor", testName)
			if engineType == lowerOracle {
				checkerName = fmt.Sprintf("%s%sListener", strings.ToLower(advisorNameTokenList[0]), strings.Join(advisorNameTokenList[1:], ""))
			} else {
				checkerName = fmt.Sprintf("%s%sChecker", strings.ToLower(advisorNameTokenList[0]), strings.Join(advisorNameTokenList[1:], ""))
			}

			fmt.Printf("Try to generate %s...\n", fileName)
			fmt.Printf("SQL rule type is %s\n", flags.rule)
			fmt.Printf("Advisor name is %s\n", advisorName)
			fmt.Printf("This rule checks for %s\n", advisorComment)
			fmt.Printf("Checker name is %s\n", checkerName)

			// generator code
			var templateFile, dir string
			switch engineType {
			case lowerMySQL:
				templateFile = mysqlTemplate
				dir = "mysql"
			case lowerPostgreSQL:
				templateFile = postgresqlTemplate
				dir = "pg"
			case lowerOracle:
				templateFile = oracleTemplate
				dir = "oracle"
			case lowerSnowflake:
				templateFile = snowflakeTemplate
				dir = "snowflake"
			case lowerMSSQL:
				templateFile = mssqlTemplate
				dir = "mssql"
			default:
				fmt.Printf("unknown engine type %s\n", engineType)
				return
			}

			if err := generateFile(path.Join(dir, fmt.Sprintf("%s.go", fileName)), templateFile, flags.rule, advisorName, advisorComment, checkerName, testName); err != nil {
				fmt.Printf("%s\n", err.Error())
			}
		},
	}
)

func generateFile(filePath, tempelatePath, advisorType, advisorName, advisorComment, checkerName, testName string) error {
	templateFile, err := os.Open(tempelatePath)
	if err != nil {
		return errors.Wrapf(err, "failed to open template file %s", tempelatePath)
	}
	defer templateFile.Close()
	goFile, err := os.Create(path.Join("..", filePath))
	if err != nil {
		return errors.Wrapf(err, "failed to create %s", filePath)
	}
	defer goFile.Close()
	scanner := bufio.NewScanner(templateFile)
	writer := bufio.NewWriter(goFile)
	for scanner.Scan() {
		text := scanner.Text()
		text = strings.ReplaceAll(text, `%AdvisorType`, advisorType)
		text = strings.ReplaceAll(text, `%AdvisorName`, advisorName)
		text = strings.ReplaceAll(text, `%AdvisorComment`, advisorComment)
		text = strings.ReplaceAll(text, `%CheckerName`, checkerName)
		text = strings.ReplaceAll(text, `%TestName`, testName)
		_, err := writer.WriteString(text + "\n")
		if err != nil {
			return errors.Wrapf(err, "failed to write string to file %s", filePath)
		}
	}
	if err := writer.Flush(); err != nil {
		return errors.Wrap(err, "failed to flush")
	}
	fmt.Printf("Generate %s successfully!\n", filePath)
	return nil
}

func init() {
	cmd.PersistentFlags().StringVar(&flags.rule, "rule", "", "rule type you want to generate. This rule type and comment must exist in /plugin/advisor/advisor.go")
}

func main() {
	//nolint
	cmd.Execute()
}
