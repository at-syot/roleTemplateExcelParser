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
	_, roles := getRoleNames(rows)
	securityGroups := getSecurityGroups(rows)
	mergedRoles := mergeWithSecurityGroups(roles, securityGroups)
	// END

	result["roles"] = mergedRoles

	dat, jsonStr := toJSONByte(result)
	saveToJsonfile(dat)

	fmt.Println(jsonStr)
}

func getRoleNames(src [][]string) ([][]string, interface{}) {
	roles := []interface{}{}

	for rowIdx, row := range src {
		for colIdx, colCell := range row {
			if rowIdx == 0 && colIdx > 2 {
				if colCell != "" {
					roleName := colCell

					role := make(map[string]interface{})
					role["roleName"] = roleName

					roles = append(roles, role)
				}
			}
		}
	}

	return src, roles
}

func getSecurityGroups(src [][]string) []string {
	securityGroups := []string{}
	for ridx, r := range src {
		for cidx, c := range r {
			if ridx > 2 && cidx == 0 && c != "" {
				securityGroups = append(securityGroups, c)
			}
		}
	}

	return securityGroups
}

func mergeWithSecurityGroups(roles interface{}, securityGroups []string) interface{} {
	_roles, _ := roles.([]interface{})
	for _, r := range _roles {
		_r, _ := r.(map[string]interface{})

		_securityGroups := []interface{}{}
		for _, securityGroupName := range securityGroups {
			_securityGroup := map[string]interface{}{}
			_securityGroup["securityGroupName"] = securityGroupName
			_securityGroup["securityPermission"] = []interface{}{}

			_securityGroups = append(_securityGroups, _securityGroup)
		}

		_r["securityGroups"] = _securityGroups
	}

	return _roles
}

// Utils
func toJSONByte(obj interface{}) ([]byte, string) {
	// map to json
	json, _ := json.Marshal(obj)
	jsonStr := string(json)
	fmt.Println(jsonStr)

	return json, jsonStr
}

func saveToJsonfile(content []byte) {
	targetJSONFilePath := "./compress.json"
	writeFileErr := ioutil.WriteFile(targetJSONFilePath, content, 0644)
	if writeFileErr != nil {
		panic(writeFileErr)
	}
}
