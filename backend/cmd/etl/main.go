package main

import (
	"context"
	"encoding/csv"
	"log"
	"os"
	"strconv"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

func main() {
	inputCsvFile := os.Args[1]
	log.Println("Opening CSV", inputCsvFile, "...")
	file, err := os.Open(inputCsvFile)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Reading records from CSV...")
	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	headers := rows[0]           // headers, if present, are always row 0
	functionNames := headers[5:] // function NAMES are in the headers, VALUES are in the rows starting with column F
	records := rows[1:]
	count := 0

	conn, err := db.Conn(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Processing scores...")
	// iterate over the records
	for _, columns := range records {
		acronym := columns[0]
		datacenterenvironment := columns[3]
		fismaSystemId := getFismaSystemId(conn, acronym)

		// collection of scores+notes pairs start at column F
		scoresNotes := columns[5:]

		// iterate through the score+notes columns by 2s
		for ii := 0; ii < len(scoresNotes); ii += 2 {
			if scoresNotes[ii] == "" {
				// in case we have trailing empty columns
				continue
			}

			functionName := functionNames[ii]
			functionId := getFunctionId(conn, functionName, datacenterenvironment)
			score, _ := strconv.ParseFloat(scoresNotes[ii][0:1], 32) // first char
			note := scoresNotes[ii+1]                                // notes are always to the right of the score
			funcScore := &functionScore{fismaSystemId, functionId, score, note}
			funcScore.save(conn)
		}
		count++
	}

	log.Printf("FINISHED processing %d records.\n", count)
}

func getFismaSystemId(conn *pgx.Conn, acronym string) int {

	row := conn.QueryRow(context.Background(), "SELECT fismasystemid FROM fismasystems WHERE UPPER(fismaacronym)=UPPER($1)", acronym)

	var fismaSystemId int
	err := row.Scan(&fismaSystemId)
	if err != nil {
		log.Printf("fismasystemid could not be found with fismaacronym: %s\n", acronym)
	}

	return fismaSystemId
}

func getFunctionId(conn *pgx.Conn, functionName, datacenterenvironment string) int {

	row := conn.QueryRow(context.Background(), "SELECT functionid FROM functions WHERE LOWER(function)=LOWER($1) AND LOWER(datacenterenvironment)=LOWER($2)", functionName, datacenterenvironment)

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

func (fs *functionScore) save(conn *pgx.Conn) {
	_, err := conn.Exec(context.Background(), "INSERT INTO functionscores(fismasystemid,functionid,datecalculated,score,notes) VALUES($1,$2,TO_TIMESTAMP('2024-09-01 12:00:00','YYYY-MM-DD HH:MI:SS'),$3,$4)", fs.fismasystemid, fs.functionid, fs.score, fs.notes)
	if err != nil {
		log.Println("function score could not be saved", err)
	}
}
