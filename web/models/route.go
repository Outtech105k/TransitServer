package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	ErrStationIDsMissing = errors.New("invalid station ID")
)

// 列車での1区間移動に対応する構造体
type Operation struct {
	TrainID         uint      `json:"train_id"`
	Order           uint      `json:"order"`
	DepartStationID uint      `json:"depart_station_id"`
	DepartDatetime  time.Time `json:"depart_time"`
	ArriveStationID uint      `json:"arrive_station_id"`
	ArriveDatetime  time.Time `json:"arrive_time"`
}

// 指定駅から指定時間以降に発車する列車を取得
// NOTE: 取得は、その駅からの次停車駅を基準にグループ化され、待ち時間が最も短いもののみが取得される
// NOTE: 「乗換回数が少ないルート」といった基準では取得できない(UNIONでいけるか？)
// NOTE: sqlxのNamedQueryはなぜか使えなかった(SQLパースエラー)
func SearchNextDepartOperations(db *sqlx.DB, departStationID uint, fastestDepartDatetime time.Time) ([]Operation, error) {
	fastestDepartDatetimeString := fastestDepartDatetime.Format("15:04:05")
	sql := `
SELECT o1.train_id, o1.op_order, o1.dep_sta_id, o1.dep_time, o1.arr_sta_id, o1.arr_time
FROM operations o1
INNER JOIN (
	SELECT train_id, op_order, dep_time,
	ROW_NUMBER() OVER (
		PARTITION BY dep_sta_id, arr_sta_id
		ORDER BY CASE
			WHEN dep_time >= ? THEN TIMEDIFF(dep_time, ?)
			ELSE TIMEDIFF(ADDTIME(dep_time, "24:00:00"), ?)
		END
	) dep_order_arr_grouped
	FROM operations
	WHERE dep_sta_id = ?
) o2
ON o2.train_id = o1.train_id
AND o2.op_order = o1.op_order
AND o2.dep_order_arr_grouped = 1
`
	rows, err := db.Query(
		sql,
		fastestDepartDatetimeString,
		fastestDepartDatetimeString,
		fastestDepartDatetimeString,
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

	// 移動先の候補を取得
	// NOTE: たとえ逆方向でも情報が取得される
	for rows.Next() {
		err := rows.Scan(&op.TrainID, &op.Order, &op.DepartStationID, &departTimeString, &op.ArriveStationID, &arriveTimeString)
		if err != nil {
			return []Operation{}, err
		}

		// DBは時刻の文字列を返すので、fastestDepartDatetime < departDatetime < arriveDatetime の順になるように変換・調整
		op.DepartDatetime, err = timeString2DatetimeForward(fastestDepartDatetime, departTimeString)
		if err != nil {
			return []Operation{}, fmt.Errorf("updateDepartTimeString: %w", err)
		}
		op.ArriveDatetime, err = timeString2DatetimeForward(op.DepartDatetime, arriveTimeString)
		if err != nil {
			return []Operation{}, fmt.Errorf("updateArriveTimeString: %w", err)
		}

		operations = append(operations, op)
	}

	return operations, nil
}

// 順移動探索における到着時刻の変換(string -> time.Time)
func timeString2DatetimeForward(fasterDatetime time.Time, laterTimeString string) (time.Time, error) {
	laterTime, err := time.Parse("15:04:05", laterTimeString)
	if err != nil {
		return time.Time{}, err
	}

	// まず、fasterDatetimeとlaterDatetimeが同日前提で変換する
	laterDatetime := time.Date(
		fasterDatetime.Year(),
		fasterDatetime.Month(),
		fasterDatetime.Day(),
		laterTime.Hour(),
		laterTime.Minute(),
		laterTime.Second(),
		0,
		fasterDatetime.Location(),
	)

	// 出発日時より到着日時が後になるべき
	// laterDatetime < fasterDatetime の場合、1日後送りにする
	// これにより、日付を跨いだ運行・乗り換えを可能とする
	// NOTE: DB側が24時間を超える運転をしないことを前提とする
	if fasterDatetime.After(laterDatetime) {
		laterDatetime = laterDatetime.AddDate(0, 0, 1)
	}

	return laterDatetime, nil
}

// 駅IDの存在チェック
func CheckExistsStationID(db *sqlx.DB, staID uint) error {
	var result bool
	err := db.QueryRow(`SELECT EXISTS(SELECT * FROM stations WHERE id = ?)`, staID).Scan(&result)
	if err != nil {
		return err
	}

	if !result {
		return ErrStationIDsMissing
	} else {
		return nil
	}
}
