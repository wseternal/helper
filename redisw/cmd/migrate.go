package main

import (
	"bitbucket.org/wseternal/helper"
	"bitbucket.org/wseternal/helper/logger"
	"bitbucket.org/wseternal/helper/redisw"
	"bitbucket.org/wseternal/helper/sqlimpl"
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"syscall"
)


type ch_record struct {
	ID int64 `sql:"id"`
	AcMac string `sql:"acmac"`
	// ShopID        int64   `sql:"shop_id"`
	WID           int64   `sql:"wid"`
	// UserID        int64   `sql:"user_id"`
	// UPHID         string  `sql:"uphid"`
	// SceneStr      string  `sql:"scene_str"`
	AppID         string  `sql:"appid"`
	OpenID        string  `sql:"OpenID"`
	UnionID       string  `sql:"unionid"`
	// Province      string  `sql:"province"`
	// City          string  `sql:"City"`
	// Country       string  `sql:"Country"`
	Height        float32 `sql:"height"`
	Weight        float32 `sql:"weight"`
	Bmi           float32 `sql:"bmi"`
	// Os            int8    `sql:"os"`
	// UA            string  `sql:"ua"`
	// Gender        int8    `sql:"gender"`
	Subscribe     int8    `sql:"subscribe"`
	SubscribeTime int64   `sql:"sub_time"`
	// IsOtherChan   int8    `sql:"is_otherchan"`
	// IsSomeDevice  int8    `sql:"is_somedevice"`
	// OrderID       int64   `sql:"order_id"`
	// Stp           int8    `sql:"stp"`
	// VerifiCode    string  `sql:"verifi_code"`
	// AuthlistSrc   int8    `sql:"authlist_src"`
	ScaleTime     int64   `sql:"scale_time"`
	AddTime       int64   `sql:"add_time"`
	UpdateTime    int64   `sql:"update_time"`
	// AddDate       string  `sql:"add_date"`
	// MarketType    string  `sql:"markettype"`
	// IsDeal        int8    `sql:"is_deal"`
}


var (
	ctx  context.Context
	cancel context.CancelFunc
	idProcessed  int64 = 0
	redisInst    *redisw.Client

	processed int64
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

	redisInst, err = redisw.NewClient(nil)
	if err != nil {
		logger.Panicf("failed to connect redis, %s\n", err)
	}

	idProcessed = getProcessedID(redisInst)
	fmt.Printf("already processed %d\n", idProcessed)

	var cond  string
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	helper.OnSignal(onSigTerm, syscall.SIGTERM)
	for {
		cond = fmt.Sprintf("where id > %d order by id asc limit 1000", idProcessed)

		if err = dtRecords.StructScan(ctx, cond, reflect.TypeOf(ch_record{}), onRow); err != nil {
			fmt.Printf("struct scan failed, %s\n", err)
			break
		}
	}

	setProcessedID(redisInst, idProcessed)
	fmt.Printf("terminate current processing... now, processed to No.%d record\n", idProcessed)
}

func onRow(val interface{}) {
	record := val.(*ch_record)
	var err error
	if err = record.migrateToRedis(redisInst); err != nil {
		fmt.Fprintf(os.Stderr, "migriate %+v to redis failed, %s\n", *record, err)
		cancel()
		return
	}
}

func onSigTerm() {
	cancel()
}

func setProcessedID(c *redisw.Client, id int64) {
	fmt.Printf("processed to id %d\n", idProcessed)
	c.Set(ProcessedID, id, 0)
}

func getProcessedID(c *redisw.Client) int64 {
	return c.GetInt64(ProcessedID, 0, nil)
}

func (r *ch_record) sanityCheck() error {
	if len(r.AcMac) == 0 {
		return fmt.Errorf("acmac for id %d is null", r.ID)
	}
	return nil
}

func (r *ch_record) migrateToRedis(c *redisw.Client) error {
	var err error

	if len(r.UnionID) == 0 {
		r.UnionID = strconv.FormatInt(r.WID, 10)
	}
	if err = r.sanityCheck(); err != nil {
		return err
	}
	if r.ScaleTime == 0 {
		r.ScaleTime = r.AddTime
	}
	if err = addUserMeasureResult(c, r.UnionID, &MeasureResult{
		Weight: r.Weight,
		Height: r.Height,
		Bmi: r.Bmi,
		Scale: r.AcMac,
		Timestamp: r.ScaleTime,
	}); err != nil {
		return err
	}

	if len(r.OpenID) > 0 && r.SubscribeTime > 0 {
		if err = addFollow(c, r.AcMac, r.UnionID, r.AppID, r.OpenID, r.SubscribeTime); err != nil {
			return err
		}
		if r.Subscribe == 0 {
			if r.UpdateTime <= r.SubscribeTime {
				fmt.Fprintf(os.Stderr, "for unsubscribe event, the update time %d is not larger than subscribe time %d\n", r.UpdateTime, r.SubscribeTime)
			} else {
				if err = addUnfollow(c, r.UnionID, r.AppID, r.UpdateTime); err != nil {
					return err
				}
			}
		}
	}

	idProcessed = r.ID
	processed++

	if processed % 1000 == 0 {
		setProcessedID(c, idProcessed)
	}
	return nil
}
