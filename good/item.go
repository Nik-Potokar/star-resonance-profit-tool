package good

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"xhgm_price_tool/good/templates"
)

var ErrNoDrop error = errors.New("no drop")
var ErrNoRecipe error = errors.New("no recipe")
var ErrNoPrice error = errors.New("no price")

const JSON_DATA_FILE = "xhgm_items.json"

var HomeDir string

func init() {
	var err error
	HomeDir, err = os.UserHomeDir()
	if err != nil {
		panic("Failed to get user home directory")
	}

	file, err := os.Open(filepath.Join(HomeDir, JSON_DATA_FILE))
	if err != nil {
		fmt.Printf("File %s does not exist in directory %s\n", JSON_DATA_FILE, HomeDir)
		fsFile, err := templates.Templates.Open("items.json")
		if err != nil {
			panic(err)
		}
		b, err := io.ReadAll(fsFile)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(b, &ItemArr)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Using preset data: %v\n", len(ItemArr))
	} else {
		ItemArr, err = loadDiskData(file)
		if err != nil {
			// If loading local data fails, use initial data
			fmt.Printf("Failed to load file %s from directory %s: %s\n", JSON_DATA_FILE, HomeDir, err.Error())
			fsFile, err := templates.Templates.Open("items.json")
			if err != nil {
				panic(err)
			}
			b, err := io.ReadAll(fsFile)
			if err != nil {
				panic(err)
			}
			err = json.Unmarshal(b, &ItemArr)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Using preset data: %v\n", len(ItemArr))
		} else {
			fmt.Println("Successfully loaded local data")
		}
	}

	// Build category mapping
	itemsMap := map[string][]*Good{}
	for i := range ItemArr {
		if len(ItemArr[i].PriceRange) >= 2 {
			item := ItemArr[i]
			item.PriceRange = []int{
				item.PriceRange[0] * 9 / 10,
				item.PriceRange[1] * 11 / 10,
			}
		}
		key := fmt.Sprintf("%d_%d", ItemArr[i].Sidebar, ItemArr[i].Tab)
		itemsMap[key] = append(itemsMap[key], ItemArr[i])
		Name2Item[ItemArr[i].Name] = ItemArr[i]
	}
	cols := 6
	for k := range itemsMap {
		for i := range itemsMap[k] {
			if itemsMap[k][i].Private {
				continue
			}
			var ok bool
			itemsMap[k][i].Category1, ok = GetSiderbarName(itemsMap[k][i].Sidebar)
			if !ok {
				panic(fmt.Sprintf("sidebar %d not found", itemsMap[k][i].Sidebar))
			}
			itemsMap[k][i].Category2, ok = GetTabName(itemsMap[k][i].Sidebar, itemsMap[k][i].Tab)
			if !ok {
				panic(fmt.Sprintf("sidebar %d tab %d not found", itemsMap[k][i].Sidebar, itemsMap[k][i].Tab))
			}
			Pos2Item[fmt.Sprintf("%d_%d_%d_%d",
				itemsMap[k][i].Sidebar, itemsMap[k][i].Tab, i%cols, i/cols)] = itemsMap[k][i]
		}
	}
}

var ItemArr = []*Good{}

var Name2Item = map[string]*Good{}

var Pos2Item = map[string]*Good{}

type Good struct {
	Sidebar    int    `json:"sidebar,omitempty"`
	Tab        int    `json:"tab,omitempty"`
	Name       string `json:"name"`
	PriceRange []int  `json:"range"`
	Price      int    `json:"price"`
	Category1  string `json:"category1,omitempty"`
	Category2  string `json:"category2,omitempty"`

	Private bool         `json:"private"`
	Recipe  []ItemRecipe `json:"recipe"`
}

type ItemRecipe struct {
	Drop      []RecipeDrop     `json:"drop"`
	Materials []RecipeMaterial `json:"materials"`
	Focus     int              `json:"focus"`
}

type RecipeDrop struct {
	Cnt  int    `json:"cnt"`
	Prob int    `json:"prob"`
	Name string `json:"name"` // If not empty, indicates a byproduct
}

type RecipeMaterial struct {
	Name string  `json:"name"`
	Cnt  float64 `json:"cnt"`
}

// GetProductionByFocus calculates the output obtained using only the focus recipe
func (ir *ItemRecipe) GetProductionByFocus(focus float64, name string) (production ProductionDetail, err error) {
	production, err = ir.GetFocusProducingX(1, name)
	if err != nil {
		return production, err
	}
	return production.MultiPower(float64(focus) / production.Focus), nil
}

// GetFocusProducingX calculates how much focus is needed to produce x items according to the recipe
func (ir ItemRecipe) GetFocusProducingX(x float64, name string) (production ProductionDetail, err error) {
	recipe := ir
	unit := 0.0
	for _, drop := range recipe.Drop {
		if drop.Name != "" && drop.Name != name {
			// Add byproduct
			production.ByProducts = append(production.ByProducts, ByProduct{
				Name: drop.Name,
				Cnt:  float64(drop.Prob) * float64(drop.Cnt) / 100,
			})
		} else {
			unit += float64(drop.Prob) * float64(drop.Cnt) / 100
		}
	}
	//
	if unit == 0.0 {
		return production, fmt.Errorf("%s %w", name, ErrNoDrop)
	}
	// Merge byproducts
	name2byproduct := map[string]ByProduct{}
	for _, bp := range production.ByProducts {
		if _, ok := name2byproduct[bp.Name]; !ok {
			name2byproduct[bp.Name] = bp
		} else {
			name2byproduct[bp.Name] = ByProduct{
				Name: bp.Name,
				Cnt:  name2byproduct[bp.Name].Cnt + bp.Cnt,
			}
		}
	}
	production.ByProducts = []ByProduct{}
	for _, bp := range name2byproduct {
		bp.Price = getPrice(bp.Name, nil)
		production.ByProducts = append(production.ByProducts, bp)
	}

	// First calculate how much focus is needed to produce unit items, then calculate how much is needed for x items
	production.Cnt = unit
	production.Focus += float64(recipe.Focus)
	for _, material := range recipe.Materials {
		if item, ok := Name2Item[material.Name]; ok {
			// Get the minimum focus for all recipes of the required material
			p, err := item.GetMinFocusProducingX(material.Cnt)
			if err != nil {
				return production, err
			}
			// Calculate how much focus is needed for the material
			p.ItemName = material.Name
			p.Price = getPrice(p.ItemName, nil)
			production.Focus += float64(p.Focus)
			production.Details = append(production.Details, p)
		} else {
			// If not in the table, the output does not require focus
		}
	}
	production.Price = getPrice(name, nil)
	return production.MultiPower(float64(x) / production.Cnt), nil
}

type ByProduct struct {
	Name  string  `json:"name"`
	Cnt   float64 `json:"cnt"`
	Price float64 `json:"price"`
}

type ProductionDetail struct {
	ItemName   string             `json:"item_name"`         // Item name
	Cnt        float64            `json:"cnt"`               // Item output quantity
	Price      float64            `json:"price"`             // Reference price
	Focus      float64            `json:"focus"`             // Total focus consumption for producing Cnt items
	Cost       float64            `json:"cost"`              // Total coin cost for buying Cnt items
	Details    []ProductionDetail `json:"details,omitempty"` // Material consumption details for item production
	ByProducts []ByProduct        `json:"by_products"`       // Item byproducts
}

func (pd *ProductionDetail) GetAllMetarialNames() (rs []string) {
	for _, detail := range pd.Details {
		rs = append(rs, detail.ItemName)
		tmp := detail.GetAllMetarialNames()
		rs = append(rs, tmp...)
	}
	return rs
}

// GetTotalMaterialsCoinCost calculates the coin cost of materials consumed in production
func (pd *ProductionDetail) GetTotalMaterialsCoinCost() (total float64) {
	for _, detail := range pd.Details {
		total += detail.Cost + detail.GetTotalMaterialsCoinCost()
	}
	return total
}

// TrBestProduction calculates the best production scenario by replacing pure focus production with purchase options
func (pd *ProductionDetail) TrBestProduction(prices map[string]float64) (result *ProductionDetail, err error) {
	result = pd.Copy()
	if len(pd.Details) == 0 {
		return
	}
	for i := range pd.Details {
		err = pd.Details[i].trBestProduction(result, prices)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// TrProfit converts production output to profit
func (pd *ProductionDetail) TrProfit(prices map[string]float64) (profit Profit) {
	profit.Name = pd.ItemName
	profit.Cnt = pd.Cnt
	profit.Cost = pd.Cost
	profit.Price = getPrice(profit.Name, prices)
	for _, bp := range pd.ByProducts {
		tmp := getPrice(bp.Name, prices)
		profit.ByProducts = append(profit.ByProducts, Profit{
			Name:  bp.Name,
			Cnt:   bp.Cnt,
			Cost:  bp.Cnt,
			Price: tmp,
		})
	}
	return profit
}

// Get the best production result for pd when producing output
func (pd *ProductionDetail) trBestProduction(output *ProductionDetail, prices map[string]float64) (err error) {
	for _, detail := range output.Details {
		if detail.ItemName == pd.ItemName {
			*pd = detail
			break
		}
	}
	avgPrice := getPrice(pd.ItemName, prices)
	if avgPrice == 0 {
		return nil
	}
	p := output.TrProfit(prices)
	outProfit := p.Value(prices)
	// for _, bp := range output.ByProducts {
	// 	price := getPrice(bp.Name, prices)
	// 	if price == 0 {
	// 		continue
	// 	}
	// 	outProfit += bp.Cnt * price * 0.95
	// }
	case1 := pd.Copy()
	// Case where pd is entirely purchased from market
	ratio := output.Focus / (output.Focus - pd.Focus)
	newProfit := ratio * (outProfit - pd.Cnt*avgPrice)
	case1.Focus = 0
	case1.Cnt = pd.Cnt
	case1.Cost = pd.Cnt * avgPrice
	case1.Details = nil
	case1.ByProducts = nil
	// fmt.Printf("直接购买: %v\n", case1)
	// fmt.Printf("直接购买比例: %v\n", ratio)

	// Get the best output for each material of pd, then manually produce
	case2 := pd.Copy()
	for i := range case2.Details {
		err = case2.Details[i].trBestProduction(case2, prices)
		if err != nil {
			return err
		}
	}
	ratio2 := output.Focus / (output.Focus - (pd.Focus - case2.Focus))
	newProfit2 := ratio2 * (outProfit - case2.Cost)
	// fmt.Printf("材料购买: %v\n", case2)
	// fmt.Printf("材料购买比例: %v\n", ratio2)

	// Compare which option yields the highest profit
	// fmt.Printf("Original profit: %v\n", outProfit)
	// fmt.Printf("Direct purchase profit: %v\n", newProfit)
	// fmt.Printf("Material purchase profit: %v\n", newProfit2)
	if outProfit > max(newProfit, newProfit2) {
		return nil
	} else if newProfit > max(outProfit, newProfit2) {
		for i := range output.Details {
			if output.Details[i].ItemName == pd.ItemName {
				output.Details[i] = *case1
				output.Cost += case1.Cost
				output.Focus -= pd.Focus
				*output = output.MultiPower(ratio)
				break
			}
		}
	} else if newProfit2 > max(outProfit, newProfit) {
		for i := range output.Details {
			if output.Details[i].ItemName == pd.ItemName {
				output.Details[i] = *case2
				output.Cost += case2.Cost
				output.Focus -= case2.Focus
				*output = output.MultiPower(ratio2)
				break
			}
		}
	}
	// fmt.Printf("新生产: %v\n\n\n", output)
	return nil
}

func (pd *ProductionDetail) GetAllProductions() (pds []ProductionDetail, err error) {
	return
}

func (pd *ProductionDetail) AdjustByPlan(plan map[string]bool) {
	cp := pd.Copy()
	var adjust func(parent, detail *ProductionDetail) (subFocus, addCost float64)
	adjust = func(parent, detail *ProductionDetail) (subFocus, addCost float64) {
		if plan[detail.ItemName] {
			subFocus, addCost = detail.Focus, detail.Cnt*detail.Price
			parent.Focus -= subFocus
			parent.Cost += addCost
			detail.Details = nil
			detail.Focus = 0
			detail.Cost = addCost
			return
		}
		sum1, sum2 := 0.0, 0.0
		for i := range detail.Details {
			tmp1, tmp2 := adjust(detail, &detail.Details[i])
			sum1 += tmp1
			sum2 += tmp2
		}
		parent.Focus -= sum1
		parent.Cost += sum2
		return sum1, sum2
	}
	for i := range cp.Details {
		adjust(cp, &cp.Details[i])
	}
	// fmt.Printf("cp.Comment(): %v\n", cp.Comment())
	*pd = cp.MultiPower(pd.Focus / cp.Focus)
}

func (pd *ProductionDetail) Copy() (copy *ProductionDetail) {
	b, _ := json.Marshal(pd)
	_ = json.Unmarshal(b, &copy)
	return
}

func (pd *ProductionDetail) Comment() string {
	var s string
	if pd.Focus > 0 {
		s = fmt.Sprintf("Produce [%s]: %.2f units, Total focus consumption: %.2f, Total coin cost: %.2f\n", pd.ItemName, pd.Cnt, pd.Focus, pd.Cost)
	} else {
		s = fmt.Sprintf("Purchase [%s]: %.2f units, Total coin cost: %.2f, Reference price: %.2f\n", pd.ItemName, pd.Cnt, pd.Cost, pd.Cost/pd.Cnt)
	}
	for _, by := range pd.ByProducts {
		s += fmt.Sprintf("Byproduct [%s]: %.2f units, Unit price: %.2f\n", by.Name, by.Cnt, getPrice(by.Name, nil))
	}
	if len(pd.Details) > 0 {
		s += "Material details:\n"
	}

	for _, detail := range pd.Details {
		tmp := detail.Comment()
		arr := strings.Split(strings.TrimSpace(tmp), "\n")
		for i := range arr {
			arr[i] = "  " + arr[i]
		}
		arr = append(arr, "------------------------------------------")
		tmp = strings.Join(arr, "\n")
		s += tmp + "\n"
	}
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "-")
	return string(s)
}

func (pd *ProductionDetail) Equation() (e string) {
	equalize := func(tmp *ProductionDetail) string {
		if tmp.Focus == 0 {
			return fmt.Sprintf("%s(Purchase, Unit price %.2f)", tmp.ItemName, tmp.Cost/tmp.Cnt)
		} else {
			return fmt.Sprintf("%s(Craft)", tmp.ItemName)
		}
	}
	lefts := []string{equalize(pd)}
	details := pd.Details
	for _, bp := range pd.ByProducts {
		lefts = append(lefts, fmt.Sprintf("%s(Output %.2f, Focus %.2f, Income %.2f)", bp.Name, bp.Cnt, pd.Focus, pd.Cost))
	}
	e = strings.Join(lefts, "+")
	for {
		tmp := []ProductionDetail{}
		rights := []string{}
		for _, detail := range details {
			rights = append(rights, equalize(&detail))
			tmp = append(tmp, detail.Details...)
		}
		e += "=" + strings.Join(rights, "+")
		if len(tmp) == 0 {
			break
		}
		details = tmp
	}
	e = strings.TrimRight(e, "=")
	return e
}

func (pd *ProductionDetail) MultiPower(power float64) ProductionDetail {
	pd.Cnt *= power
	pd.Focus *= power
	pd.Cost *= power
	for i := range pd.ByProducts {
		pd.ByProducts[i].Cnt *= power
	}
	if len(pd.Details) == 0 {
		return *pd
	}
	for i := range pd.Details {
		pd.Details[i] = pd.Details[i].MultiPower(power)
	}
	return *pd
}

func (pd *ProductionDetail) AddCost(diff float64) ProductionDetail {
	pd.Cost += diff
	if len(pd.Details) == 0 {
		return *pd
	}
	for i := range pd.Details {
		pd.Details[i] = pd.Details[i].AddCost(diff)
	}
	return *pd
}

func (i *Good) GetMinFocusProducingX(x float64) (production ProductionDetail, err error) {
	if len(i.Recipe) == 0 {
		return production, fmt.Errorf("%s %w", i.Name, ErrNoRecipe)
	}
	productions := make([]ProductionDetail, 0, len(i.Recipe))
	for _, recipe := range i.Recipe {
		p, err := recipe.GetFocusProducingX(x, i.Name)
		if err != nil {
			return production, err
		}
		p.ItemName = i.Name
		productions = append(productions, p)
	}
	return slices.MinFunc(productions, func(a, b ProductionDetail) int {
		return cmp.Compare(a.Focus, b.Focus)
	}), nil
}

func (i *Good) GetMaxCntByFocus(focus float64) (production ProductionDetail, err error) {
	if len(i.Recipe) == 0 {
		return production, fmt.Errorf("%s %w", i.Name, ErrNoRecipe)
	}
	productions := make([]ProductionDetail, 0, len(i.Recipe))
	for _, recipe := range i.Recipe {
		p, err := recipe.GetProductionByFocus(focus, i.Name)
		if err != nil {
			return production, err
		}
		p.ItemName = i.Name
		p.Price = float64(i.Price)
		productions = append(productions, p)
	}

	return slices.MaxFunc(productions, func(a, b ProductionDetail) int {
		return cmp.Compare(a.Cnt, b.Cnt)
	}), nil
}

// GenerateAllFlagCombinations generates all flag combinations that meet the conditions
// 1. The flag of the root node is fixed to false
// 2. When a node's flag is true, all non-root nodes in its subtree have flags set to false
func (i *Good) GenerateAllFlagCombinations() []map[string]bool {
	// Result set, each map represents a possible flag combination
	result := []map[string]bool{}

	// If there are no recipes, return empty result
	if len(i.Recipe) == 0 {
		return result
	}

	// Recursive function to generate all possible flag combinations
	var generateCombinations func(node *Good, currentCombination map[string]bool, parentFlag bool) []map[string]bool
	generateCombinations = func(node *Good, currentCombination map[string]bool, parentFlag bool) []map[string]bool {
		// If parent node's flag is true, current node's flag must be false
		if parentFlag {
			currentCombination[node.Name] = false
			return []map[string]bool{currentCombination}
		}

		// When parent node's flag is false, current node has two possibilities: true or false
		combinations := []map[string]bool{}

		// Case 1: current node's flag is false
		falseCombination := make(map[string]bool)
		for k, v := range currentCombination {
			falseCombination[k] = v
		}
		falseCombination[node.Name] = false

		// Case 2: current node's flag is true
		trueCombination := make(map[string]bool)
		for k, v := range currentCombination {
			trueCombination[k] = v
		}
		trueCombination[node.Name] = true

		// If there are no child nodes, directly return the two combinations
		if len(node.Recipe) == 0 || len(node.Recipe[0].Materials) == 0 {
			combinations = append(combinations, falseCombination, trueCombination)
			return combinations
		}

		// Process child nodes
		// For flag=false case, recursively process all child nodes
		childCombinations := []map[string]bool{falseCombination}
		for _, material := range node.Recipe[0].Materials {
			childNode, ok := Name2Item[material.Name]
			if !ok {
				continue
			}

			var newCombinations []map[string]bool
			for _, combo := range childCombinations {
				// Generate all possible combinations for child nodes for each existing combination
				childResults := generateCombinations(childNode, combo, false)
				newCombinations = append(newCombinations, childResults...)
			}
			childCombinations = newCombinations
		}
		combinations = append(combinations, childCombinations...)

		// For flag=true case, all child nodes' flags must be false
		childCombinations = []map[string]bool{trueCombination}
		for _, material := range node.Recipe[0].Materials {
			childNode, ok := Name2Item[material.Name]
			if !ok {
				continue
			}

			var newCombinations []map[string]bool
			for _, combo := range childCombinations {
				// Generate all possible combinations for child nodes for each existing combination (child node flags must be false)
				childResults := generateCombinations(childNode, combo, true)
				newCombinations = append(newCombinations, childResults...)
			}
			childCombinations = newCombinations
		}
		combinations = append(combinations, childCombinations...)

		return combinations
	}

	// Initial combination, root node's flag is fixed to false
	initialCombination := map[string]bool{i.Name: false}

	// Generate all combinations starting from the root node
	allCombinations := []map[string]bool{initialCombination}

	// Process all child nodes of the root node
	for idx := range i.Recipe {
		for _, material := range i.Recipe[idx].Materials {
			childNode, ok := Name2Item[material.Name]
			if !ok {
				continue
			}

			var newCombinations []map[string]bool
			for _, combo := range allCombinations {
				// Generate all possible combinations for child nodes for each existing combination
				childResults := generateCombinations(childNode, combo, false)
				newCombinations = append(newCombinations, childResults...)
			}
			allCombinations = newCombinations
		}
	}

	// Deduplication: remove duplicate combinations
	uniqueCombinations := []map[string]bool{}
	combinationExists := make(map[string]bool)

	for _, combination := range allCombinations {
		// Convert combination to string for comparison
		combinationKey := ""
		keys := make([]string, 0, len(combination))
		for k := range combination {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		for _, k := range keys {
			if combination[k] {
				combinationKey += k + ":true,"
			} else {
				combinationKey += k + ":false,"
			}
		}

		// If this combination hasn't appeared yet, add it to the result
		if !combinationExists[combinationKey] {
			combinationExists[combinationKey] = true
			uniqueCombinations = append(uniqueCombinations, combination)
		}
	}

	return uniqueCombinations
}

func loadDiskData(reader io.Reader) (goods []*Good, err error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &goods)
	if err != nil {
		return
	}
	if len(goods) > 0 {
		return
	} else {
		return nil, fmt.Errorf("Data in file %s under directory %s is empty", JSON_DATA_FILE, HomeDir)
	}
}
