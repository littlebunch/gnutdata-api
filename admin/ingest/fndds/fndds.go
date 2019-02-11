//Package fndds implements an Ingest for Food Survey data
package fndds

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/littlebunch/gnutdata-bfpd-api/admin/ingest"
	"github.com/littlebunch/gnutdata-bfpd-api/admin/ingest/dictionaries"
	"github.com/littlebunch/gnutdata-bfpd-api/ds"
	fdc "github.com/littlebunch/gnutdata-bfpd-api/model"
)

var (
	cnts ingest.Counts
	err  error
)

// Fndds for assigning the inteface
type Fndds struct {
	Doctype string
}

// ProcessFiles loads a set of FNDSS csv files processed
// in this order:
//		food.csv  -- main food file
//		food_portion.csv  -- servings sizes for each food
//		food_nutrient.csv -- nutrient values for each food
func (p Fndds) ProcessFiles(path string, dc ds.DataSource) error {
	rcs := make(chan error)
	rcn := make(chan error)
	rci := make(chan error)
	err = foods(path, dc, p.Doctype)
	if err != nil {
		log.Fatal(err)
	}

	go servings(path, dc, rcs)
	go nutrients(path, dc, rcn)
	go inputFoods(path, dc, rci)
	for i := 0; i < 3; i++ {
		select {
		case errs := <-rcs:
			if errs != nil {
				fmt.Printf("Error from servings: %v\n", errs)
			} else {
				fmt.Printf("Servings ingest complete.\n")
			}

		case errn := <-rcn:
			if err != nil {
				fmt.Printf("Error from nutrients: %v\n", errn)
			} else {
				fmt.Printf("Nutrient ingest complete.\n")
			}

		case erri := <-rci:
			if erri != nil {
				fmt.Printf("Error from foodInput %v\n", erri)
			} else {
				fmt.Printf("Food input complete.\n")
			}

		}
	}

	log.Printf("Finished.  Counts: %d Foods %d Servings %d Nutrients %d Other\n", cnts.Foods, cnts.Servings, cnts.Nutrients, cnts.Other)
	return err
}
func foods(path string, dc ds.DataSource, t string) error {
	fn := path + "food.csv"
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	r := csv.NewReader(f)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		cnts.Foods++
		if cnts.Foods%1000 == 0 {
			log.Println("Count = ", cnts.Foods)
		}
		pubdate, err := time.Parse("2006-01-02", record[4])
		if err != nil {
			log.Println(err)
		}
		dc.Update(record[0],
			fdc.Food{
				FdcID:           record[0],
				Description:     record[2],
				PublicationDate: pubdate,
				Source:          t,
				Type:            "FOOD",
			})
	}
	return err
}

// servings implements an ingest of fdc.Food.ServingSizes for FNDDS foods
func servings(path string, dc ds.DataSource, rc chan error) {
	//defer wg.Done()
	fn := path + "food_portion.csv"
	f, err := os.Open(fn)
	if err != nil {
		rc <- err
		return
	}
	r := csv.NewReader(f)
	cid := ""
	var (
		food fdc.Food
		s    []fdc.Serving
	)

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			rc <- err
			return
		}

		id := record[1]
		if cid != id {
			if cid != "" {
				food.Servings = s
				dc.Update(cid, food)
			}
			cid = id
			dc.Get(id, &food)

			//food.Group = record[7]
			s = nil
		}

		cnts.Servings++
		if cnts.Servings%10000 == 0 {
			log.Println("Servings Count = ", cnts.Servings)
		}

		a, err := strconv.ParseFloat(record[7], 32)
		if err != nil {
			log.Println(record[0] + ": can't parse serving amount " + record[3])
		}
		s = append(s, fdc.Serving{
			Nutrientbasis: "g",
			Description:   record[5],
			Servingamount: float32(a),
		})

	}
	rc <- nil
	return
}

// nutrients implements an ingest of fdc.Food.NutrietData for FNDDS foods
func nutrients(path string, dc ds.DataSource, rc chan error) {
	//defer wg.Done()
	fn := path + "food_nutrient.csv"
	f, err := os.Open(fn)
	if err != nil {
		rc <- err
		return
	}
	r := csv.NewReader(f)
	cid := ""
	var (
		food fdc.Food
		n    []fdc.NutrientData
		il   interface{}
	)
	if err := dc.GetDictionary("gnutdata", "NUT", 0, 500, &il); err != nil {
		rc <- err
		return
	}
	nutmap := dictionaries.InitNutrientInfoMap(il)

	if err := dc.GetDictionary("gnutdata", "DERV", 0, 500, &il); err != nil {
		rc <- err
		return
	}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			rc <- err
			return
		}

		id := record[1]
		if cid != id {
			if cid != "" {
				food.Nutrients = n
				dc.Update(cid, food)
			}
			cid = id
			dc.Get(id, &food)
			n = nil
		}
		cnts.Nutrients++
		w, err := strconv.ParseFloat(record[3], 32)
		if err != nil {
			log.Println(record[0] + ": can't parse value " + record[4])
		}

		v, err := strconv.ParseInt(record[2], 0, 32)
		if err != nil {
			log.Println(record[0] + ": can't parse nutrient no " + record[1])
		}
		var dv *fdc.Derivation
		dv = nil
		n = append(n, fdc.NutrientData{
			Nutrientno: nutmap[uint(v)].Nutrientno,
			Value:      float32(w),
			Nutrient:   nutmap[uint(v)].Name,
			Unit:       nutmap[uint(v)].Unit,
			Derivation: dv,
		})
		if cnts.Nutrients%30000 == 0 {
			log.Println("Nutrients Count = ", cnts.Nutrients)
		}

	}
	rc <- err
	return
}

// inputFoods implements an ingest of fdc.Food.InputFoods for FNDDS foods
func inputFoods(path string, dc ds.DataSource, rc chan error) {
	//defer wg.Done()
	fn := path + "input_food.csv"
	f, err := os.Open(fn)
	if err != nil {
		rc <- err
		return
	}
	r := csv.NewReader(f)
	cid := ""
	var (
		food  fdc.Food
		ifood []fdc.InputFood
	)

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			rc <- err
			return
		}

		id := record[1]
		if cid != id {
			if cid != "" {
				food.InputFoods = ifood
				dc.Update(cid, food)
			}
			cid = id
			dc.Get(id, &food)

			//food.Group = record[7]
			ifood = nil
		}

		cnts.Other++
		if cnts.Servings%10000 == 0 {
			log.Println("Servings Count = ", cnts.Other)
		}

		a, err := strconv.ParseFloat(record[4], 32)
		if err != nil {
			log.Println(record[0] + ": can't parse serving amount " + record[3])
		}
		w, err := strconv.ParseFloat(record[10], 16)
		if err != nil {
			log.Println(record[0] + ": can't parse gram_weight " + record[10])
		}
		c, err := strconv.ParseInt(record[5], 0, 32)
		if err != nil {
			log.Println(record[0] + ": can't parse sr_code " + record[5])
		}
		seq, err := strconv.ParseInt(record[3], 0, 32)
		if err != nil {
			log.Println(record[0] + ": can't parse input seq no " + record[3])
		}
		ifood = append(ifood, fdc.InputFood{
			SeqNo:              int(seq),
			Unit:               record[7],
			SrCode:             int(c),
			Description:        record[6],
			Amount:             float32(a),
			Portion:            record[8],
			PortionDescription: record[9],
			Weight:             float32(w),
		})

	}
	rc <- nil
	return
}