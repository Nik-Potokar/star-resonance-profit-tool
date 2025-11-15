package good

import (
	"fmt"
	"testing"
)

func TestPrice(t *testing.T) {
	item := Name2Item["Brilliant Stone"]
	_, d, err := GetItemBestProfitByFocus(item.Name, 400, map[string]float64{
		"Aztec Refined Ore": 200,
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("p: %v\n", d.Comment())
}
