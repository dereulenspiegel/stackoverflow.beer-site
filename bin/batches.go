package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/dereulenspiegel/go-brewchild"
)

var (
	brewfatherUserID = os.Getenv("BREWFATHER_USERID")
	brewfatherAPIKey = os.Getenv("BREWFATHER_APIKEY")
)

var (
	dataBaseDir  = "../data/batches"
	batchPostDir = "../content/batches"
)

var (
	untappdIDRegex = regexp.MustCompile(`untappd\(([\d]+)\)`)
)

type batch struct {
	Name        string   `json:"name"`
	ABV         float64  `json:"abv"`
	IBU         int      `json:"ibu"`
	Color       float64  `json:"color"`
	BrewDate    string   `json:"brewDate"`
	Hops        []string `json:"hops"`
	OG          float64  `json:"og"`
	BuGuRation  float64  `json:"buGuRatio"`
	UntappdLink string   `json:"untappdLink"`
	Number      int      `json:"number"`
}

func main() {
	bfClient, err := brewchild.New(brewfatherUserID, brewfatherAPIKey)
	if err != nil {
		log.Fatalf("Failed to create brewfather client: %w", err)
	}

	for _, state := range []string{"Completed", "Planning", "Brewing", "Fermenting", "Conditioning", "Archived"} {
		exportBatches(bfClient, state)
	}
}

func exportBatches(bfClient *brewchild.Client, state string) {
	outFilePath := filepath.Join(dataBaseDir, state+".json")
	batches, err := bfClient.Batches(brewchild.Status(state), brewchild.Complete(true), brewchild.Limit(10))
	if err != nil {
		log.Fatalf("Failed to retrieve batches from brewfather: %s", err)
	}
	outFile, err := os.Create(outFilePath)
	if err != nil {
		log.Fatalf("Failed to create output file")
	}
	defer outFile.Close()

	b := make([]batch, len(batches))

	for i, bt := range batches {
		b[i] = batch{
			Name:   bt.Name,
			ABV:    bt.ABV,
			Color:  bt.EstimatedColor,
			Number: bt.BatchNumber,
		}
		if bt.OG != 0.0 {
			b[i].OG = brewchild.SGToPlato(bt.OG)
		} else {
			b[i].OG = brewchild.SGToPlato(bt.EstimatedOG)
		}
		if b[i].ABV == 0.0 {
			b[i].ABV = bt.MeasuredABV
		}

		if bt.IBU != 0 {
			b[i].IBU = bt.IBU
		} else {
			b[i].IBU = bt.EstimatedIBU
		}

		if bt.BrewDate != nil {
			b[i].BrewDate = bt.BrewDate.String()
		}
		if bt.BuGuRatio != 0.0 {
			b[i].BuGuRation = bt.BuGuRatio
		} else {
			b[i].BuGuRation = bt.EstimatedBuGuRatio
		}

		if bt.BatchNotes != "" {
			if m := untappdIDRegex.FindStringSubmatch(bt.BatchNotes); len(m) > 1 {
				b[i].UntappdLink = fmt.Sprintf("https://untappd.com/qr/beer/%s", m[1])
			}
		}

		ensureBatchPostData(b[i])
	}

	if err := json.NewEncoder(outFile).Encode(b); err != nil {
		log.Fatalf("Failed to encode output data")
	}
}

func ensureBatchPostData(b batch) {
	batchFolder := filepath.Join(batchPostDir, fmt.Sprintf("%d", b.Number))
	if err := os.MkdirAll(batchFolder, 0776); err != nil {
		log.Fatalf("Unable to create batch folder %s: %s", batchFolder, err)
	}
	dataFilePath := filepath.Join(batchFolder, "data.json")

	if _, err := os.Stat(dataFilePath); os.IsExist(err) {
		os.Remove(dataFilePath)
	}
	dataFile, err := os.Create(dataFilePath)
	if err != nil {
		log.Fatalf("Failed to create batch data file %s: %s", dataFilePath, err)
	}
	defer dataFile.Close()
	if err := json.NewEncoder(dataFile).Encode(b); err != nil {
		log.Fatalf("Failed to marshal batch data file %s: %s", dataFilePath, err)
	}
}
