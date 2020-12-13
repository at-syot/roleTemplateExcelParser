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
	rules := getRules(rows)
	_, roles, ruleRanges := getRoleNames(rows)
	securityGroups, pgRanges := getSecurityGroups(rows)
	mergedRoles := mergeWithSecurityGroups(roles, securityGroups)
	resultRoles := mergeWithSecurityGroupPermission(rows, mergedRoles, pgRanges, ruleRanges, rules)
	// END

	_ = ruleRanges
	_ = securityGroups
	_ = pgRanges

	result["roles"] = resultRoles

	// dat, jsonStr := toJSONByte(result)
	dat, _ := toJSONByte(result)

	saveToJsonfile(dat)
	// fmt.Printf("JSON result --> %v", jsonStr)
}

func getRules(src [][]string) []Rule {
	rowIdx := 1
	colStartIdx := 3
	colEndIdx := 10

	rules := []Rule{}
	for colStartIdx <= colEndIdx {
		rn := src[rowIdx][colStartIdx]
		rules = append(rules, Rule{
			title:  rn,
			enable: false,
		})

		colStartIdx++
	}

	return rules
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

	return src, roles, ruleRanges
}

func getSecurityGroups(src [][]string) ([]string, []PermissionGroupRange) {
	securityGroups := []string{}
	permissionGroupRanges := []PermissionGroupRange{}

	rowIdx := 0
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

func mergeWithSecurityGroupPermission(src [][]string, roles interface{}, sgRanges []PermissionGroupRange, ruleRanges []RoleRuleRange, rules []Rule) interface{} {
	// Interate through the roles json
	_roles, _ := roles.([]interface{})
	for _, r := range _roles {
		_r, _ := r.(map[string]interface{})
		securityGroups := _r["securityGroups"].([]interface{})

		// grep each securityGroup
		for _, sg := range securityGroups {
			_sg := sg.(map[string]interface{})
			sgName := _sg["securityGroupName"].(string)

			// get Permissions for each Group
			roleName := _r["roleName"].(string)
			groupPermission := getPermissions(src, roleName, sgName, sgRanges, ruleRanges, rules)
			_ = groupPermission

			// fmt.Printf("rn: %v \n gp: %v\n\n\n", roleName, groupPermission.gn)

			// arrange to json format
			sgPermissions := _sg["securityPermissions"].([]interface{})
			for _, permission := range groupPermission.permissions {
				tmpPermission := map[string]interface{}{}
				tmpPermission["name"] = permission.name

				tmpRules := []map[string]interface{}{}
				for _, rule := range permission.rules {
					_rule := map[string]interface{}{}
					_rule["name"] = rule.title
					_rule["enable"] = rule.enable

					tmpRules = append(tmpRules, _rule)
				}
				tmpPermission["rules"] = tmpRules

				sgPermissions = append(sgPermissions, tmpPermission)
			}

			_sg["securityPermissions"] = sgPermissions
		}
	}

	return _roles
}

func getPermissions(src [][]string, rn string, gn string, sgRanges []PermissionGroupRange, ruleRanges []RoleRuleRange, rules []Rule) GroupPermission {
	var targetRuleRange RoleRuleRange
	var targetSgRange PermissionGroupRange
	for _, sgRange := range sgRanges {
		if sgRange.gn == gn {
			targetSgRange = sgRange
		}
	}

	for _, ruleRange := range ruleRanges {
		if ruleRange.roleName == rn {
			targetRuleRange = ruleRange
		}
	}

	resultGroupPermission := GroupPermission{
		gn:          gn,
		permissions: []Permission{},
	}
	rowStart := targetSgRange.rowStart + 1
	rowEnd := targetSgRange.rowEnd
	permissionColStart := targetSgRange.colStart

	for rowStart <= rowEnd {
		pn := src[rowStart][permissionColStart]
		if pn == "" {
			pn = fmt.Sprintf(" / %v", src[rowStart][permissionColStart+1])
		}

		permission := Permission{}
		permission.name = pn

		// START find permission's rule
		ruleColStart := targetRuleRange.ruleStart
		ruleColEnd := targetRuleRange.ruleEnd

		// // for get rule type
		justRuleIdx := 0
		for ruleColStart <= ruleColEnd && rowStart < 110 {
			/*
				if rn == "Branch Manager (Role)" {
					fmt.Printf("(r%v, c%v)sg: %v, pn: %v, rn: %v, enable: %v\n\n", rowStart, ruleColStart, gp.gn, pn, rules[justRuleIdx], src[rowStart][ruleColStart])
				}
			*/

			rule := rules[justRuleIdx]
			if ruleEnable := src[rowStart][ruleColStart]; ruleEnable != "" {
				rule.enable = true
			}

			permission.rules = append(permission.rules, rule)

			justRuleIdx++
			ruleColStart++
		}
		// END

		resultGroupPermission.permissions = append(resultGroupPermission.permissions, permission)

		rowStart++
	}

	return resultGroupPermission
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

type (
	Rule struct {
		title  string
		enable bool
	}

	Permission struct {
		name  string
		rules []Rule
	}

	GroupPermission struct {
		gn          string
		permissions []Permission
	}
)

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
