// Package fdc describes food products data model
package fdc

import (
	"io/ioutil"
	"log"
	"os"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// BrowseResult is returned from the browse endpoints
type BrowseResult struct {
	Count int32         `json:"count"`
	Start int32         `json:"start"`
	Max   int32         `json:"max"`
	Items []interface{} `json:"items"`
}

// BrowseServings is returned from the browse endpoints
type BrowseServings struct {
	FdcID    string    `json:"fdcId" binding:"required"`
	Servings []Serving `json:"servingSizes"`
}

// BrowseNutrients is returned from the browse endpoints
type BrowseNutrients struct {
	FdcID     string         `json:"fdcId" binding:"required"`
	Nutrients []NutrientData `json:"nutrients"`
}

// FoodMeta abbreviated Food containing only meta-data
type FoodMeta struct {
	FdcID        string `json:"fdcId" binding:"required"`
	Upc          string `json:"upc"`
	Description  string `json:"foodDescription" binding:"required"`
	Source       string `json:"dataSource"`
	Manufacturer string `json:"company"`
	Type         string `json:"type" binding:"required"`
}

// Food reflects JSON used to transfer BFPD foods data from USDA csv
type Food struct {
	UpdatedAt       time.Time      `json:"lastChangeDateTime"`
	FdcID           string         `json:"fdcId" binding:"required"`
	Upc             string         `json:"upc"`
	Description     string         `json:"foodDescription" binding:"required"`
	Source          string         `json:"dataSource"`
	PublicationDate time.Time      `json:"publicationDateTime"`
	Ingredients     string         `json:"ingredients"`
	Manufacturer    string         `json:"company"`
	Servings        []Serving      `json:"servingSizes,omitempty"`
	Nutrients       []NutrientData `json:"nutrients,omitempty"`
	Type            string         `json:"type" binding:"required"`
}

// Serving describes a list nutrients for a given state, weight and amount
// A subdocument of Food
type Serving struct {
	Nutrientbasis string  `json:"100UnitNutrientBasis"`
	Description   string  `json:"householdServingUom"`
	Servingstate  string  `json:"servingState,omitempty"`
	Weight        float32 `json:"weightInGmOrMl"`
	Servingamount float32 `json:"householdServingValue"`
}

// Nutrient is metadata abount nutrients usually in a nutrients collection
type Nutrient struct {
	Nutrientno uint   `json:"nutrientno" binding:"required"`
	Tagname    string `json:"tagname"`
	Name       string `json:"name"  binding:"required"`
	Unit       string `json:"unit"  binding:"required"`
	Type       string `json:"type"  binding:"required"`
}

// Derivation is a code for describing how nutrient values are derived
type Derivation struct {
	Code string `json:"code"`
}

// NutrientData is the list of nutrient values
// A subdocument of Food
type NutrientData struct {
	Value      float32 `json:"valuePer100UnitServing"`
	Unit       string  `json:"unit"  binding:"required"`
	Derivation string  `json:"derivation"`
	Nutrientno uint    `json:"nutrientNumber"`
	Nutrient   string  `json:"nutrientName"`
}

// FoodGroup is the dictionary of FNDDS and SR food groups
type FoodGroup struct {
	Code        string `json:"code" binding:"required"`
	Description string `json:"description" binding:"required"`
	LastUpdate  string `json:"lastUpdate"`
	Type        string `json:"type" binding:"required"`
}

//Config provides basic CouchBase configuration properties for API services.  Properties are normally read in from a YAML file or the environment
type Config struct {
	CouchDb CouchDb
}

// CouchDb configuration for connecting, reading and writing Couchbase nodes
type CouchDb struct {
	URL    string
	Bucket string
	Fts    string
	User   string
	Pwd    string
}

// Defaults sets values for CouchBase configuration properties if none have been provided.
func (cs *Config) Defaults() {
	if os.Getenv("COUCHBASE_URL") != "" {
		cs.CouchDb.URL = os.Getenv("COUCHBASE_URL")
	}
	if os.Getenv("COUCHBASE_BUCKET") != "" {
		cs.CouchDb.Bucket = os.Getenv("COUCHBASE_BUCKET")
	}
	if os.Getenv("COUCHBASE_FTSINDEX") != "" {
		cs.CouchDb.Fts = os.Getenv("COUCHBASE_FTSINDEX")
	}
	if os.Getenv("COUCHBASE_USER") != "" {
		cs.CouchDb.User = os.Getenv("COUCHBASE_USER")
	}
	if os.Getenv("COUCHBASE_PWD") != "" {
		cs.CouchDb.Pwd = os.Getenv("COUCHBASE_PWD")
	}
	if cs.CouchDb.URL == "" {
		cs.CouchDb.URL = "localhost"
	}
	if cs.CouchDb.Bucket == "" {
		cs.CouchDb.Bucket = "gnutdata"
	}
	if cs.CouchDb.Fts == "" {
		cs.CouchDb.Fts = "fd_food"
	}
}

// GetConfig reads config from a file
func (cs *Config) GetConfig(c *string) {
	raw, err := ioutil.ReadFile(*c)
	if err != nil {
		log.Println(err.Error())
	}
	if err = yaml.Unmarshal(raw, cs); err != nil {
		log.Println(err.Error())
	}
	cs.Defaults()
}
