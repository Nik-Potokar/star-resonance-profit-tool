package good

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func SavePrice() {
	b, _ := json.Marshal(ItemArr)
	file := filepath.Join(HomeDir, JSON_DATA_FILE)
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.Write(b)
}

func getPrice(name string, prices map[string]float64) (price float64) {
	if len(prices) > 0 {
		price, ok := prices[name]
		if ok {
			return price
		}
	}
	return float64(Name2Item[name].Price)
}

type Profit struct {
	Name       string   `json:"name"`
	Cnt        float64  `json:"cnt"`        // Quantity
	Cost       float64  `json:"cost"`       // Cost
	Price      float64  `json:"price"`      // Unit price
	ByProducts []Profit `json:"byProducts"` // Byproducts
	Total      float64  `json:"value"`      // Total price

	Comment string  `json:"comment"` // Notes
	Num     float64 `json:"num"`
}

func (p Profit) Value(prices map[string]float64) float64 {
	tmp := p.Price
	if tmp2, ok := prices[p.Name]; ok {
		tmp = tmp2
	}
	rs := p.Cnt * tmp * 0.95 // 税
	for _, bp := range p.ByProducts {
		rs += bp.Value(prices)
	}
	return rs - p.Cost
}

func GetItemProfitByFocus(name string, focus float64, prices map[string]float64) (profit Profit, detail ProductionDetail, err error) {
	profit.Name = name
	price := getPrice(name, prices)
	if price == 0 {
		return profit, detail, nil
	}
	if item, ok := Name2Item[name]; ok {
		detail, err = item.GetMaxCntByFocus(focus)
		if err != nil {
			return profit, detail, nil
		}
		profit := detail.TrProfit(prices)
		return profit, detail, nil
	}
	return profit, detail, fmt.Errorf("item %s not found", name)
}

func GetAllAvgProfitByFocus(focus float64, prices map[string]float64) (profits []Profit, details []ProductionDetail, err error) {
	type tmp struct {
		p Profit
		d ProductionDetail
	}
	arr := []tmp{}
	for name := range Name2Item {
		if strings.Contains(name, "Tablet") {
			continue
		}
		p, detail, err := GetItemProfitByFocus(name, focus, prices)
		if err != nil {
			return nil, nil, err
		}
		if p.Value(prices) == 0 {
			continue
		}
		arr = append(arr, tmp{
			p,
			detail,
		})
	}
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].p.Value(prices) > arr[j].p.Value(prices)
	})
	for i := range arr {
		profits = append(profits, arr[i].p)
		details = append(details, arr[i].d)
	}
	return profits, details, nil
}

func GetItemBestProfitByFocus(name string, focus float64, prices map[string]float64) (profit Profit, detail ProductionDetail, err error) {
	item, ok := Name2Item[name]
	if !ok {
		return profit, detail, fmt.Errorf("item '%s' not found", name)
	}
	if len(item.Recipe) == 0 {
		return Profit{
			Name: item.Name,
		}, detail, nil
	}
	// First get the best output recipe for the item under focus
	detail, err = item.GetMaxCntByFocus(focus)
	if err != nil {
		return profit, detail, err
	}
	// Convert to the best output recipe mixed with purchases
	detail = TrBestProduction(detail, prices)
	profit = detail.TrProfit(prices)
	return profit, detail, nil
}

func GetAllItemBestProfitByFocus(focus float64, prices map[string]float64) (profits []Profit, details []ProductionDetail, err error) {
	type tmp struct {
		p Profit
		d ProductionDetail
	}
	arr := []tmp{}
	for _, item := range Name2Item {
		if strings.Contains(item.Name, "Tablet") {
			continue
		}
		p, detail, err := GetItemBestProfitByFocus(item.Name, focus, prices)
		if err != nil {
			if strings.Contains(err.Error(), "no recipe") {
				continue
			}
			return nil, nil, err
		}
		if p.Value(prices) == 0 {
			continue
		}
		arr = append(arr, tmp{
			p,
			detail,
		})
	}

	sort.Slice(arr, func(i, j int) bool {
		return arr[i].p.Value(prices) > arr[j].p.Value(prices)
	})
	for i := range arr {
		arr[i].p.Comment = arr[i].d.Comment()
		arr[i].p.Num = arr[i].p.Value(prices)
		profits = append(profits, arr[i].p)
		details = append(details, arr[i].d)
	}
	return
}

func TrBestProduction(raw ProductionDetail, prices map[string]float64) (result ProductionDetail) {
	plans := Name2Item[raw.ItemName].GenerateAllFlagCombinations()
	maxProfit := raw.TrProfit(prices).Value(prices)
	result = *raw.Copy()
	for _, plan := range plans {
		tmp := raw.Copy()
		realPlan := map[string]bool{}
		for _, name := range raw.GetAllMetarialNames() {
			realPlan[name] = plan[name]
		}
		tmp.AdjustByPlan(realPlan)
		if tmp.TrProfit(prices).Value(prices) > maxProfit {
			result = *tmp
		}
	}
	return result
}

func GetAllProductions(raw ProductionDetail, prices map[string]float64) (results []ProductionDetail) {
	plans := Name2Item[raw.ItemName].GenerateAllFlagCombinations()
	for _, plan := range plans {
		tmp := raw.Copy()
		realPlan := map[string]bool{}
		for _, name := range raw.GetAllMetarialNames() {
			realPlan[name] = plan[name]
		}
		tmp.AdjustByPlan(realPlan)
		profit := tmp.TrProfit(nil).Value(nil)
		fmt.Printf("%.2f\n", profit)
		fmt.Printf("result.Comment(): %v\n\n", tmp.Comment())
		results = append(results, *tmp)
	}
	return results
}
