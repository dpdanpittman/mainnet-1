package main

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"os"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	passage "github.com/envadiv/Passage3D/app"

	claimtypes "github.com/envadiv/Passage3D/x/claim/types"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/types"
)

type ClaimAccountRecord struct {
	Address string
	Amount  int64
}

func readCsvFile(filePath string) ([][]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Unable to parse file as CSV for "+filePath, err)
	}

	return records, nil
}

func AddClaimRecords() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "add-claim-records [genesis-file] [claim-records-file]",
		Long: "Add claim records to genesis.json file from claim_records.csv",
		Args: cobra.ExactArgs(2),
		Example: `
		go run main.go add-claim-records genesis-file.json claim-records.csv 
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			genesisFilePath := args[0]
			claimRecordsFilePath := args[1]

			doc, err := types.GenesisDocFromFile(genesisFilePath)
			if err != nil {
				return err
			}

			//
			records, err := readCsvFile(claimRecordsFilePath)
			if err != nil {
				return err
			}

			claimRecords, err := parseClaimAccountRecordsFromCsv(records)
			if err != nil {
				return err
			}

			// add claim records from claim records file
			err = addClaimRecords(doc, claimRecords)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func parseClaimAccountRecordsFromCsv(records [][]string) ([]ClaimAccountRecord, error) {
	var claimAccountsRecords []ClaimAccountRecord
	for _, record := range records {
		var claimAccountRecord ClaimAccountRecord
		claimAccountRecord.Address = record[0]
		amount, err := strconv.ParseInt(record[1], 10, 64)
		if err != nil {
			return claimAccountsRecords, err
		}
		claimAccountRecord.Amount = amount
		claimAccountsRecords = append(claimAccountsRecords, claimAccountRecord)
	}

	return claimAccountsRecords, nil
}

func addClaimRecords(doc *types.GenesisDoc, claimAccountRecords []ClaimAccountRecord) error {
	var genState map[string]json.RawMessage
	err := json.Unmarshal(doc.AppState, &genState)
	if err != nil {
		return err
	}

	cdc := passage.MakeEncodingConfig()

	claimRecords := make([]claimtypes.ClaimRecord, len(claimAccountRecords))
	totalActions := len(claimtypes.Action_name)

	claimModuleAccountBalance := sdk.NewCoin("upasg", sdk.NewInt(0))

	for index, record := range claimAccountRecords {
		var claimRecord claimtypes.ClaimRecord

		actions := make([]bool, totalActions)
		claimAmountForAction := record.Amount / int64(totalActions)

		// adding the each account record amount into module account balance
		claimModuleAccountBalance = claimModuleAccountBalance.Add(sdk.NewCoin("upasg", sdk.NewInt(record.Amount)))

		claimRecord.Address = record.Address
		claimRecord.ActionCompleted = actions
		for i := 0; i < totalActions-1; i++ {
			claimRecord.ClaimableAmount = append(claimRecord.ClaimableAmount, sdk.NewCoin("upasg", sdk.NewInt(claimAmountForAction)))
		}
		a := record.Amount - int64(claimAmountForAction*(int64(totalActions)-1))
		claimRecord.ClaimableAmount = append(claimRecord.ClaimableAmount, sdk.NewCoin("upasg", sdk.NewInt(a)))

		claimRecords[index] = claimRecord
	}

	var claimGenesis claimtypes.GenesisState
	err = cdc.Marshaler.UnmarshalJSON(genState[claimtypes.ModuleName], &claimGenesis)
	if err != nil {
		return err
	}

	claimGenesis.ClaimRecords = claimRecords
	claimGenesis.ModuleAccountBalance = claimModuleAccountBalance

	genState[claimtypes.ModuleName], err = cdc.Marshaler.MarshalJSON(&claimGenesis)
	if err != nil {
		return err
	}

	doc.AppState, err = json.Marshal(genState)
	if err != nil {
		return err
	}

	genFile := "claim-passage-genesis.json"
	return doc.SaveAs(genFile)
}
