package main

import (
	"github.com/go-sql-driver/mysql"
	"reflect"
	"math"
)

// NullTime is an alias for mysql.NullTime data type
type NullTime mysql.NullTime

// Scan implements the Scanner interface for NullTime
func (nt *NullTime) Scan(value interface{}) error {
	var t mysql.NullTime
	if err := t.Scan(value); err != nil {
		return err
	}

	// if nil then make Valid false
	if reflect.TypeOf(value) == nil {
		*nt = NullTime{t.Time, false}
	} else {
		*nt = NullTime{t.Time, true}
	}

	return nil
}


func Round(x, unit float64) float64 {
	if unit==0{
		return math.Round(x)
	}else {
		return math.Round(x*unit) / unit
	}
}



