//   File Name:  structHandler_test.go.go
//    Description:
//    Author:      Chenghu
//    Date:       2024/3/7 18:06
//    Change Activity:

package baseValue

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestStructSetDefaultValue(t *testing.T) {

	type UserPageInfo struct {
		Education    *string `db:"education" json:"education" default:",canZero"`
		Province     string  `db:"province" json:"province"`
		City         string  `db:"city" json:"city" default:""`
		DistanceHide int     `db:"distance_hide" json:"distance_hide" default:"2"`
	}

	type Uyd struct {
		UserPageInfo
		Name string `db:"name" json:"name" default:"nihao"`
		Nag  *int   `db:"nag" json:"nag" default:",canZero"`
	}
	dd := Uyd{}
	err := StructSetDefaultValue(&dd, "default")
	marshal, err := json.Marshal(dd)
	if err != nil {
		return
	}
	println(string(marshal))
	//fmt.Println(dd, err)

}

func TestStructGetTagValueNames(t *testing.T) {
	type UserPageInfo struct {
		Education    *string `db:"education" json:"education" default:"\"\""`
		Province     string  `db:"province" json:"province"`
		City         string  `db:"city" json:"city" default:""`
		DistanceHide int     `db:"distance_hide" json:"distance_hide" default:"2"`
	}
	type Uyd struct {
		UserPageInfo
		Name string `db:"name" json:"name" default:"nihao"`
	}
	fmt.Println(StructGetTagValueNames(&Uyd{}, "db"))
}

type Location struct {
	Province *string `json:"province"`
	City     *string `json:"city"`
}

func (Location) GetCity() string {
	return "ius"
}

func TestMakeService(t *testing.T) {
	var provvince = "黄山"
	uy := Location{
		Province: &provvince,
	}
	dr := MakeService(&uy)
	for name, i := range dr.Attr {
		fmt.Println(name, i.Interface())
	}
	de := dr.Method["GetCity"].Func.Call([]reflect.Value{dr.Rcvr})
	fmt.Println(de)
}
