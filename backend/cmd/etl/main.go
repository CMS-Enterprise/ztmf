package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
)

func main() {

	inputCsvFile := os.Args[1]

	file, err := os.Open(inputCsvFile)
	if err != nil {
		log.Fatal(err)
	}

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	headers := rows[0]
	functionNames := headers[5:]
	records := rows[1:]

	for i, columns := range records {
		fmt.Println("Processing record", i, "-------------------------------------------------------------------")
		acronym := columns[0]
		datacenterenvironment := columns[3]
		scoresNotes := columns[5:]

		fismaSystemId := getFismaSystemId(acronym)

		for ii := 0; ii < len(scoresNotes); ii += 2 {
			if scoresNotes[ii] == "" {
				// in case we have trailing empty columns
				continue
			}

			functionName := functionNames[ii]
			functionId := getFunctionId(functionName, datacenterenvironment)
			score, _ := strconv.ParseFloat(scoresNotes[ii][0:1], 32) // first char
			note := scoresNotes[ii+1]
			funcScore := &functionScore{fismaSystemId, functionId, score, note}
			funcScore.save()
		}
	}
}

func getFismaSystemId(acronym string) int {
	dbpool := db.GetPool()
	row := dbpool.QueryRow(context.Background(), "SELECT fismasystemid FROM fismasystems WHERE LOWER(fismaacronym)=LOWER($1)", acronym)

	var fismaSystemId int
	err := row.Scan(&fismaSystemId)
	if err != nil {
		log.Printf("fismasystemid could not be found with fismaacronum: %s\n", acronym)
	}

	return fismaSystemId
}

func getFunctionId(functionName, datacenterenvironment string) int {
	dbpool := db.GetPool()
	row := dbpool.QueryRow(context.Background(), "SELECT functionid FROM functions WHERE LOWER(function)=LOWER($1) AND LOWER(datacenterenvironment)=LOWER($2)", functionName, datacenterenvironment)

	var functionid int

	err := row.Scan(&functionid)
	if err != nil {
		log.Printf("functionid could not be found with function: %s and datacenterenvironment:%s\n", functionName, datacenterenvironment)
	}

	return functionid
}

type functionScore struct {
	fismasystemid int
	functionid    int
	score         float64
	notes         string
}

func (fs *functionScore) save() {
	dbpool := db.GetPool()

	_, err := dbpool.Exec(context.Background(), "INSERT INTO functionscores(fismasystemid,functionid,datecalculated,score,notes) VALUES($1,$2,TO_TIMESTAMP('2024-09-01 12:00:00','YYYY-MM-DD HH:MI:SS'),$3,$4)", fs.fismasystemid, fs.functionid, fs.score, fs.notes)
	if err != nil {
		log.Println("function score could not be saved", err)
	}
}
