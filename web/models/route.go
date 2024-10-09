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

// 指定駅から発車する列車を取得
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

		op.ArriveTime, err = updateTimeWithString(op.DepartTime, arriveTimeString)
		if err != nil {
			return []Operation{}, fmt.Errorf("updateArriveTimeString: %w", err)
		}

		operations = append(operations, op)
	}

	return operations, nil
}

// 順移動探索における到着時刻の変換(string -> time.Time)
func updateTimeWithString(departDateTime time.Time, arriveTimeString string) (time.Time, error) {
	// 到着時刻をTime型に変換(日時はデフォルト値)
	arriveTime, err := time.Parse("15:04:05", arriveTimeString)
	if err != nil {
		return time.Time{}, err
	}

	// 出発日を基に、到着日時を設定
	arriveDateTime := time.Date(
		departDateTime.Year(),
		departDateTime.Month(),
		departDateTime.Day(),
		arriveTime.Hour(),         // 時刻部分を置き換え
		arriveTime.Minute(),       // 分部分を置き換え
		arriveTime.Second(),       // 秒部分を置き換え
		0,                         // ナノ秒は0に設定
		departDateTime.Location(), // タイムゾーンも元のものを使用
	)

	// 出発日時より到着日時が後になるべきだが、日付を跨いでいる場合は時系列が逆転している
	// その場合、到着日を1日後送りにすることで、日付を跨いだ運行・乗り換えを可能とする
	// ただし、DB側が24時間を超える運転をしないことを前提とする
	if departDateTime.After(arriveDateTime) {
		arriveDateTime = arriveDateTime.AddDate(0, 0, 1)
	}

	return arriveDateTime, nil
}

// 駅IDの存在チェック(出発駅・到着駅)
// NOTE: クエリ2つにすべき？
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
