package good

var SidebarMap = map[int]string{
	0: "Growth Items",
	1: "Life Skills",
	// 2: "Modules",
	// 3: "Appearance",
}

// sidebar_tab -> name
var TabMap = map[int]map[int]string{
	0: {
		0: "Ability Training",
		1: "Equipment Training",
		2: "Will",
		3: "Fantasy Materials",
	},
	1: {
		0: "Botany",
		1: "Mineralogy",
		2: "Gemology",
		3: "Cooking",
		4: "Alchemy",
		5: "Forging",
		6: "Crafting",
		7: "Woodworking",
		8: "Dyeing",
	},
}

func GetSiderbarName(index int) (string, bool) {
	name, ok := SidebarMap[index]
	return name, ok
}

func GetTabName(sidebarIndex, tabIndex int) (string, bool) {
	tmp, ok := TabMap[sidebarIndex]
	if !ok {
		return "", ok
	}
	name, ok := tmp[tabIndex]
	return name, ok
}
