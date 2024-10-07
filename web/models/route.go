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
// TODO: 日付を超えた乗り換え案内
func SearchNextOperations(db *sql.DB, departStationID uint, fastestDepartDateTime time.Time) ([]Operation, error) {
	rows, err := db.Query(`
SELECT o1.train_id, o1.op_order, o1.dep_sta_id, o1.dep_time, o1.arr_sta_id, o1.arr_time
FROM operations o1 
INNER JOIN (
	SELECT dep_sta_id, MIN(dep_time) AS next_dep_time, arr_sta_id
	FROM operations
	WHERE dep_sta_id = ? AND dep_time >= ?
	GROUP BY dep_sta_id, arr_sta_id
	ORDER BY dep_sta_id ASC
) o2
ON o1.dep_sta_id = o2.dep_sta_id
AND o1.dep_time = o2.next_dep_time
AND o1.arr_sta_id = o2.arr_sta_id
`,
		departStationID,
		fastestDepartDateTime.Format("15:04:05"),
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

func updateTimeWithString(originalTime time.Time, timeString string) (time.Time, error) {
	// "15:04:05"形式の時刻部分をパース
	parsedTime, err := time.Parse("15:04:05", timeString)
	if err != nil {
		return time.Time{}, err
	}

	// 年月日を維持しつつ、時刻を上書き
	updatedTime := time.Date(
		originalTime.Year(),
		originalTime.Month(),
		originalTime.Day(),
		parsedTime.Hour(),       // 時刻部分を置き換え
		parsedTime.Minute(),     // 分部分を置き換え
		parsedTime.Second(),     // 秒部分を置き換え
		0,                       // ナノ秒は0に設定
		originalTime.Location(), // タイムゾーンも元のものを使用
	)

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
