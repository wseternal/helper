package main

import (
	"bitbucket.org/wseternal/helper"
	"bitbucket.org/wseternal/helper/logger"
	"bitbucket.org/wseternal/helper/redisw"
	"bitbucket.org/wseternal/helper/sqlimpl"
	"fmt"
	"reflect"
	"syscall"
)


type ch_record struct {
	ID int64 `sql:"id"`
	AcMac string `sql:"acmac"`
	ShopID int64 `sql:"shop_id"`
	WID int64 `sql:"wid"`
	UserID int64 `sql:"user_id"`
	UPHID string `sql:"uphid"`
	SceneStr string `sql:"scene_str"`
	AppID string `sql:"appid"`
	OpenID string `sql:"OpenID"`
	UnionID string `sql:"unionid"`
	Province string `sql:"province"`
	City string `sql:"City"`
	Country string `sql:"Country"`
	Height float32 `sql:"height"`
	Weight float32 `sql:"weight"`
	Bmi float32 `sql:"bmi"`
	Os int8 `sql:"os"`
	UA string `sql:"ua"`
	Gender int8 `sql:"gender"`
	Subscribe int8 `sql:"subscribe"`
	SubTime int64 `sql:"sub_time"`
	IsOtherChan int8 `sql:"is_otherchan"`
	IsSomeDevice int8 `sql:"is_somedevice"`
	OrderID int64 `sql:"order_id"`
	Stp int8 `sql:"stp"`
	VerifiCode string `sql:"verifi_code"`
	AuthlistSrc int8 `sql:"authlist_src"`
	ScaleTime int64 `sql:"scale_time"`
	AddTime int64 `sql:"add_time"`
	UpdateTime int64 `sql:"update_time"`
	AddDate string `sql:"add_date"`
	MarketType string `sql:"markettype"`
	IsDeal int8 `sql:"is_deal"`
}

var (
	abortProcess = false
	idProcessed int64 = 0
)
const (
	ProcessedID = "ch_records_processed"
)

func main() {
	my, err := sqlimpl.ConnectDB(sqlimpl.DriverlMysql, `root:weixiaoxin123@tcp(121.42.157.74:3316)/wifiadx`)
	if err != nil {
		logger.Panicf("open db failed, %s\n", err)
	}
	defer my.Close()

	var dtRecords *sqlimpl.DataTable
	dtRecords, err = my.GetTable("wifi_ch_records")
	if err != nil {
		logger.Panicf("get wifi_ch_records table failed, %s\n", err)
	}

	redis, err := redisw.NewClient(nil)
	if err != nil {
		logger.Panicf("failed to connect redis, %s\n", err)
	}

	idProcessed = redis.GetInt64(ProcessedID, 0, nil)
	fmt.Printf("current processed %d\n", idProcessed)

	helper.OnSignal(onSigTerm, syscall.SIGTERM)

	if err = dtRecords.StructScan("order by id asc limit 3", reflect.TypeOf(ch_record{}), onRow); err != nil {
		fmt.Printf("struct scan failed, %s\n", err)
	}
	fmt.Printf("terminate current processing... now, processed to No.%d record\n", idProcessed)
}

func onRow(val interface{}) {
	fmt.Printf("onrow: %+v\n", val)
}

func onSigTerm() {
	abortProcess = true
}
