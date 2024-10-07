package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrStationIDsMissing = errors.New("invalid station ID")
)

type Operation struct {
	TrainID         uint      `json:"train_id"`
	Order           uint      `json:"order"`
	DepartStationID uint      `json:"depart_station_id"`
	DepartTime      time.Time `json:"depart_time"`
	ArriveStationID uint      `json:"arrive_station_id"`
	ArriveTime      time.Time `json:"arrive_time"`
}

// 指定駅から指定時間以降に発車する列車を取得
// TODO: 00:00を超えて走行する列車の日付更新
func SearchNextDepartOperations(db *sql.DB, departStationID uint, fastestDepartDateTime time.Time) ([]Operation, error) {
	fastestDepartDateTimeString := fastestDepartDateTime.Format("15:04:05")
	sql := `
 WITH operation_waits AS (
	SELECT train_id, op_order, dep_time,
	CASE
		WHEN dep_time >= ? THEN TIMEDIFF(dep_time, ?)
		ELSE TIMEDIFF(ADDTIME(dep_time, "24:00:00"), ?)
	END AS wait_time,
	ROW_NUMBER() OVER (
		PARTITION BY dep_sta_id, arr_sta_id
		ORDER BY CASE
			WHEN dep_time >= ? THEN TIMEDIFF(dep_time, ?)
			ELSE TIMEDIFF(ADDTIME(dep_time, "24:00:00"), ?)
		END
	) dep_order_arr_grouped
	FROM operations
	WHERE dep_sta_id = ?
)
SELECT o.train_id, o.op_order, o.dep_sta_id, o.dep_time, o.arr_sta_id, o.arr_time
FROM operations o
INNER JOIN operation_waits ow
ON ow.train_id = o.train_id
AND ow.op_order = o.op_order
AND ow.dep_order_arr_grouped = 1
ORDER BY wait_time
`
	rows, err := db.Query(
		sql,
		fastestDepartDateTimeString,
		fastestDepartDateTimeString,
		fastestDepartDateTimeString,
		fastestDepartDateTimeString,
		fastestDepartDateTimeString,
		fastestDepartDateTimeString,
		departStationID,
	)
	if err != nil {
		return []Operation{}, err
	}

	operations := make([]Operation, 0, 5)
	var (
		op               Operation
		departTimeString string
		arriveTimeString string
	)

	for rows.Next() {
		err := rows.Scan(&op.TrainID, &op.Order, &op.DepartStationID, &departTimeString, &op.ArriveStationID, &arriveTimeString)
		if err != nil {
			return []Operation{}, err
		}

		op.DepartTime, err = updateTimeWithString(fastestDepartDateTime, departTimeString)
		if err != nil {
			return []Operation{}, fmt.Errorf("updateDepartTimeString: %w", err)
		}

		op.ArriveTime, err = updateTimeWithString(fastestDepartDateTime, arriveTimeString)
		if err != nil {
			return []Operation{}, fmt.Errorf("updateArriveTimeString: %w", err)
		}

		operations = append(operations, op)
	}

	return operations, nil
}

// 順探索のみ対応
func updateTimeWithString(originalDateTime time.Time, timeString string) (time.Time, error) {
	// "15:04:05"形式の時刻部分をパース
	parsedTime, err := time.Parse("15:04:05", timeString)
	if err != nil {
		return time.Time{}, err
	}

	// 年月日を維持しつつ、時刻を上書き
	updatedTime := time.Date(
		originalDateTime.Year(),
		originalDateTime.Month(),
		originalDateTime.Day(),
		parsedTime.Hour(),           // 時刻部分を置き換え
		parsedTime.Minute(),         // 分部分を置き換え
		parsedTime.Second(),         // 秒部分を置き換え
		0,                           // ナノ秒は0に設定
		originalDateTime.Location(), // タイムゾーンも元のものを使用
	)

	// parsedTimeがoriginalTimeの時刻より前なら、updatedTimeを次の日とみなす
	// これにより、日付を跨いだ運行・乗り換えを可能とする
	if originalDateTime.After(updatedTime) {
		updatedTime = updatedTime.AddDate(0, 0, 1)
	}

	return updatedTime, nil
}

func CheckExistsStationIDs(db *sql.DB, depID, arrID uint) error {
	var result bool
	err := db.QueryRow(`SELECT COUNT(*) = 2 FROM stations WHERE id IN (?, ?);`, depID, arrID).Scan(&result)
	if err != nil {
		return err
	}

	if !result {
		return ErrStationIDsMissing
	} else {
		return nil
	}
}
