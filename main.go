package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
)

func main() {
	f, err := excelize.OpenFile("hive.xlsx")
	if err != nil {
		fmt.Println(err)
	}

	targetSheetName := "Restaurant by Position"
	rows, err := f.GetRows(targetSheetName)
	if err != nil {
		fmt.Println(err)
	}

	result := make(map[string]interface{})

	// START
	_, roles, ruleRanges := getRoleNames(rows)
	securityGroups, pgRanges := getSecurityGroups(rows)
	mergedRoles := mergeWithSecurityGroups(roles, securityGroups)
	mergeWithSecurityGroupPermission(rows, mergedRoles)
	// END

	fmt.Println(ruleRanges)
	fmt.Println(securityGroups)
	fmt.Println(pgRanges)

	result["roles"] = mergedRoles

	// dat, jsonStr := toJSONByte(result)
	dat, _ := toJSONByte(result)

	saveToJsonfile(dat)
	// fmt.Printf("JSON result --> %v", jsonStr)
}

func getRoleNames(src [][]string) ([][]string, interface{}, []RoleRuleRange) {
	roles := []interface{}{}
	ruleRanges := []RoleRuleRange{}

	for rowIdx, row := range src {
		for colIdx, colCell := range row {
			if rowIdx == 0 && colIdx > 2 {
				if colCell != "" {
					roleName := colCell

					role := make(map[string]interface{})
					role["roleName"] = roleName

					roles = append(roles, role)
					ruleRanges = append(ruleRanges, RoleRuleRange{
						roleName:  roleName,
						ruleStart: colIdx,
						ruleEnd:   colIdx + 7,
					})
				}
			}
		}
	}

	fmt.Println(ruleRanges)

	return src, roles, ruleRanges
}

func getSecurityGroups(src [][]string) ([]string, []PermissionGroupRange) {
	securityGroups := []string{}
	permissionGroupRanges := []PermissionGroupRange{}

	rowIdx := 0
	fmt.Printf("src len: %v\n", len(src))
	for rowIdx < len(src) {

		colIdx := 0
		for colIdx < len(src[rowIdx]) {
			col := src[rowIdx][colIdx]
			// fmt.Printf("rowIdx: %v, col v: %v\n", rowIdx, col)

			if rowIdx > 2 && colIdx == 0 && col != "" {

				// * start grap permission group range
				tmpRowIdx := rowIdx + 1
				tmpColValue := src[tmpRowIdx][0]
				rangeCount := 0
				for tmpColValue == "" && tmpRowIdx < len(src) {
					tmpColValue = src[tmpRowIdx][0]
					tmpRowIdx++
					rangeCount++
				}
				// * end

				permissionGroupRanges = append(permissionGroupRanges, PermissionGroupRange{
					gn:       col,
					rowStart: rowIdx,
					rowEnd:   rowIdx + (rangeCount - 1),
					colStart: 1,
					colEnd:   2,
				})
				securityGroups = append(securityGroups, col)
			}
			colIdx = colIdx + 1 // *
		}
		rowIdx = rowIdx + 1 // *
	}

	return securityGroups, permissionGroupRanges
}

func mergeWithSecurityGroups(roles interface{}, securityGroups []string) interface{} {
	_roles, _ := roles.([]interface{})

	for _, r := range _roles {
		_r, _ := r.(map[string]interface{})

		_securityGroups := []interface{}{}
		for _, securityGroupName := range securityGroups {
			_securityGroup := map[string]interface{}{}
			_securityGroup["securityGroupName"] = securityGroupName
			_securityGroup["securityPermissions"] = []interface{}{}

			_securityGroups = append(_securityGroups, _securityGroup)
		}

		_r["securityGroups"] = _securityGroups
	}

	return _roles
}

func mergeWithSecurityGroupPermission(src [][]string, roles interface{}) {
	// Interate through the roles json
	_roles, _ := roles.([]interface{})
	for _, r := range _roles {
		_r, _ := r.(map[string]interface{})
		securityGroups := _r["securityGroups"].([]interface{})

		// grep each securityGroup
		for _, sg := range securityGroups {
			_sg := sg.(map[string]interface{})
			sgName := _sg["securityGroupName"].(string)

			// sgPermissions := _sg["securityPermissions"].([]interface{})
			// fmt.Println(sgPermissions)

			// fmt.Printf("at roleName: %v\n\n", _r["roleName"])
			// fmt.Println(sgName)

			// get Permissions for each group
			roleName := _r["roleName"].(string)
			getPermissionsByGroupAndRole(src, sgName, roleName)
		}
	}
}

func getPermissionsByGroupAndRole(src [][]string, sgName string, roleName string) []interface{} {
	startRowIdx := 2
	// permissionRange := struct{
	//   start int
	//   end int
	// }{1, 2}

	for rIdx, r := range src {
		for cIdx, c := range r {
			// just permissionGroup
			if rIdx > startRowIdx && cIdx == 0 && c != "" && c == sgName {

				// find permissions
				// for _rIdx, _r := range src {
				//   for _cIdx, _c := range _r {
				//     if _rIdx > startRowIdx && _cIdx >= permissionRange.start && _cIdx <= permissionRange.end {
				//       fmt.Printf("g: %v, p: %v\n", sgName, _c)
				//     }
				//   }
				// }
				// end find permission
			}
		}
	}

	return []interface{}{}
}

// Types
type GroupPermissionsRage struct {
	roleName string
	rowStart int
	rowEnd   int
	colStart int
	colEnd   int
}

type RoleRuleRange struct {
	roleName  string
	ruleStart int
	ruleEnd   int
}

type PermissionGroupRange struct {
	gn       string
	rowStart int
	rowEnd   int
	colStart int
	colEnd   int
}

// Utils
func toJSONByte(obj interface{}) ([]byte, string) {
	// map to json
	json, _ := json.Marshal(obj)
	jsonStr := string(json)
	// fmt.Println(jsonStr)

	return json, jsonStr
}

func saveToJsonfile(content []byte) {
	targetJSONFilePath := "./compress.json"
	writeFileErr := ioutil.WriteFile(targetJSONFilePath, content, 0644)
	if writeFileErr != nil {
		panic(writeFileErr)
	}
}
