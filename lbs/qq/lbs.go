package qq

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type LocationLL struct {
	Lat float32 `json:"lat"`
	Lng float32 `json:"lng"`
}

type LocationInfo struct {
	ID string `json:"id"`
	Title string `json:"title"`
	Location LocationLL `json:"location"`

	// 此参考位置到输入坐标的直线距离
	Distance float32 `json:"_distance"`

	// 此参考位置到输入坐标的方位关系，如：北、南、内
	DirDesc string `json:"_dir_desc"`
}

type AdInfoBasic struct {
	AdCode string `json:"adcode"`
	Province string `json:"province"`
	City string `json:"city"`
	District string `json:"district"`
}

type POIInfo struct {
	Address string `json:"address"`
	Category string `json:"category"`
	LocationInfo
	AdInfo AdInfoBasic `json:"ad_info"`
}

type GeoCoderLocation struct {
	Location *LocationLL `json:"location"`

	// 	地址描述
	Address string `json:"address"`

	// 	位置描述
	FormattedAddresses *struct {
		Recommend string `json:"recommend"`
		Rough string `json:"rough"`
	} `json:"formatted_addresses"`

	// 地址部件，address不满足需求时可自行拼接
	AddressComponent struct {
		Nation string `json:"nation"`
		Province string `json:"province"`
		City string `json:"city"`
		District string `json:"district"`
		Street string `json:"street"`
		StreetNumber string `json:"street_number"`
	} `json:"address_component"`

	// AdInfo 行政区划信息
	AdInfo struct {
		NationCode string `json:"nation_code"`
		Name string `json:"name"`
		CityCode string `json:"city_code"`
		Nation string `json:"nation"`
		Location LocationLL `json:"location"`

		AdInfoBasic
	} `json:"ad_info"`

	// 坐标相对位置参考
	AddressReference *struct {
		// 街道
		Street *LocationInfo `json:"street"`

		// 门牌
		StreetNumber *LocationInfo `json:"street_number"`

		// 交叉路口
		CrossRoad *LocationInfo `json:"crossroad"`

		// 乡镇街道
		Town *LocationInfo `json:"town"`

		// 知名区域
		FamousArea *LocationInfo `json:"famous_area"`

		// 一级地标
		LandmarkL1 *LocationInfo `json:"landmark_l1"`

		// 二级地标
		LandmarkL2 *LocationInfo `json:"landmark_l2"`

		// 水域
		Water *LocationInfo `json:"water"`
	} `json:"address_reference"`

	POICount int `json:"poi_count"`
	POIs []POIInfo `json:"pois"`
}

type LocationOption struct {
	ShortAddressFormat bool

	// follow options are related to POI
	GetPoi bool

	// 1-5000
	Radius int

	// 1-20
	PageSize int

	// 1-20
	PageIndex int
	// 1-5
	// 1[默认] 以地标+主要的路+近距离POI为主，着力描述当前位置；
	// 2 到家场景：筛选合适收货的POI，并会细化收货地址，精确到楼栋；
	// 3 出行场景：过滤掉车辆不易到达的POI(如一些景区内POI)，增加道路出入口、交叉口、大区域出入口类POI，排序会根据真实API大用户的用户点击自动优化。
	// 4 社交签到场景，针对用户签到的热门 地点进行优先排序。
	// 5 位置共享场景，用户经常用于发送位置、位置分享等场景的热门地点优先排序
	Policy int

	Category string
}

type LocationResponse struct {
	Status int
	Message string
	Result  *GeoCoderLocation
}

var (
	defaultPOIOption =  &LocationOption{
		GetPoi:             true,
		ShortAddressFormat: true,
		Radius:             100,
		PageSize:           20,
		PageIndex:          1,
		Policy:             3,
		Category:           "",
	}
)

const (
	LBSBaseURL = "https://apis.map.qq.com/ws/geocoder/v1/"
)

func QueryGeocoderLocation(key string, lat,lng string, opt *LocationOption) (*GeoCoderLocation, error) {
	if opt == nil {
		opt = defaultPOIOption
	}
	query := url.Values{}
	query.Set("key", key)
	query.Set("location", fmt.Sprintf("%s,%s", lat, lng))
	if opt.GetPoi {
		query.Set("get_poi", "1")

		var poiOpts []string
		if opt.Radius > 0 {
			poiOpts = append(poiOpts, fmt.Sprintf("radius=%d", opt.Radius))
		}
		if opt.PageSize > 0 && opt.PageIndex > 0 {
			poiOpts = append(poiOpts, fmt.Sprintf("page_size=%d", opt.PageSize))
			poiOpts = append(poiOpts, fmt.Sprintf("page_index=%d", opt.PageIndex))
		}
		if opt.Policy >= 1 && opt.Policy <=5 {
			poiOpts = append(poiOpts, fmt.Sprintf("policy=%d", opt.Policy))
		}
		if len(opt.Category) > 0 {
			poiOpts = append(poiOpts, fmt.Sprintf("category=%s", opt.Category))
		}
		if len(poiOpts) > 0 {
			query.Set("poi_options", strings.Join(poiOpts, ";"))
		}
	}
	u := fmt.Sprintf("%s?%s", LBSBaseURL, query.Encode())
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}

	var data []byte
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		return nil, fmt.Errorf("read data on resp.Body failed, %s", err)
	}

	res := &LocationResponse{}
	if err = json.Unmarshal(data, res); err != nil {
		return nil, fmt.Errorf("unmarshal response data: %s to LocationResponse failed, %s", string(data), err)
	}
	if res.Status != 0 {
		return nil, fmt.Errorf("non-zero status code returned for request: %s, %s", u, string(data))
	}
	return res.Result, nil
}

func QueryGeocoderLocationPois(key string, lat,lng string, radius int, category string) (*GeoCoderLocation, error) {
	opt := *defaultPOIOption
	opt.Radius = radius
	opt.Category = category

	return QueryGeocoderLocation(key, lat, lng, &opt)
}