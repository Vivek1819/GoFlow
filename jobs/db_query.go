package jobs

import (
	"context" // ✅ ADD
	"encoding/json"
	"fmt"
)

func executeDBQuery(ctx context.Context, payload map[string]interface{}) (int, []byte, error) {

	// 🔴 EARLY CANCEL CHECK
	if ctx.Err() == context.Canceled {
		return 0, nil, fmt.Errorf("db query cancelled")
	}

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

	// =========================
	// 🔥 QUERY WITH ROWS
	// =========================
	if returnRows {

		// ✅ CONTEXT-AWARE QUERY
		rows, err := DB.QueryContext(ctx, query, args...)
		if err != nil {

			if ctx.Err() == context.Canceled {
				return 0, nil, fmt.Errorf("db query cancelled")
			}

			return 0, nil, err
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return 0, nil, err
		}

		var results []map[string]interface{}

		for rows.Next() {

			// 🔴 CANCEL CHECK DURING ITERATION
			if ctx.Err() == context.Canceled {
				return 0, nil, fmt.Errorf("db iteration cancelled")
			}

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

	// =========================
	// 🔥 EXEC (NO ROWS)
	// =========================

	result, err := DB.ExecContext(ctx, query, args...)
	if err != nil {

		if ctx.Err() == context.Canceled {
			return 0, nil, fmt.Errorf("db exec cancelled")
		}

		return 0, nil, err
	}

	rowsAffected, _ := result.RowsAffected()

	response := map[string]interface{}{
		"rows_affected": rowsAffected,
	}

	jsonBytes, _ := json.Marshal(response)
	return 200, jsonBytes, nil
}