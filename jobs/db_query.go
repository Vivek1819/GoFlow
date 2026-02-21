package jobs

import (
	"encoding/json"
	"fmt"
)

func executeDBQuery(payload map[string]interface{}) (int, []byte, error) {

	query, ok := payload["query"].(string)
	if !ok || query == "" {
		return 0, nil, fmt.Errorf("missing 'query'")
	}

	var args []interface{}
	if rawArgs, ok := payload["args"].([]interface{}); ok {
		args = rawArgs
	}

	returnRows := false
	if rr, ok := payload["return_rows"].(bool); ok {
		returnRows = rr
	}

	// If returning rows
	if returnRows {

		rows, err := DB.Query(query, args...)
		if err != nil {
			return 0, nil, err
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return 0, nil, err
		}

		var results []map[string]interface{}

		for rows.Next() {

			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))

			for i := range columns {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return 0, nil, err
			}

			rowMap := make(map[string]interface{})
			for i, col := range columns {
				val := values[i]

				// Convert []byte to string
				if b, ok := val.([]byte); ok {
					rowMap[col] = string(b)
				} else {
					rowMap[col] = val
				}
			}

			results = append(results, rowMap)
		}

		jsonBytes, _ := json.Marshal(results)
		return 200, jsonBytes, nil
	}

	// Otherwise just Exec
	result, err := DB.Exec(query, args...)
	if err != nil {
		return 0, nil, err
	}

	rowsAffected, _ := result.RowsAffected()

	response := map[string]interface{}{
		"rows_affected": rowsAffected,
	}

	jsonBytes, _ := json.Marshal(response)
	return 200, jsonBytes, nil
}